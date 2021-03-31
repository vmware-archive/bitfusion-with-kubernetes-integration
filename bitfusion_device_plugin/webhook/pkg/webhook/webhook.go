/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	defaulter    = runtime.ObjectDefaulter(runtimeScheme)
	zeroQuantity = resource.Quantity{}
)

var ignoredNamespaces = []string{
	metav1.NamespaceSystem,
	metav1.NamespacePublic,
}

const (
	admissionWebhookAnnotationInjectKey = "auto-management/bitfusion"
	admissionWebhookAnnotationStatusKey = "auto-management/status"
	// "~1" is used for escape (http://jsonpatch.com/)
	bitFusionGPUResource              = "bitfusion.io/gpu"
	bitFusionGPUResourceNum           = "bitfusion.io/gpu-num"
	bitFusionGPUResourceMemory        = "bitfusion.io/gpu-memory"
	bitFusionGPUResourcePartial       = "bitfusion.io/gpu-percent"
	bitFusionGPUResourceNumEscape     = "bitfusion.io~1gpu-num"
	bitFusionGPUResourcePartialEscape = "bitfusion.io~1gpu-percent"
	bitFusionGPUResourceMemoryEscape  = "bitfusion.io~1gpu-memory"
)

type WebhookServer struct {
	SidecarConfig *Config
	Server        *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	Port           int    // webhook server port
	CertFile       string // path to the x509 certificate for https
	KeyFile        string // path to the x509 private key matching `CertFile`
	SidecarCfgFile string // path to sidecar injector configuration file
}

type Config struct {
	InitContainers []corev1.Container `yaml:"initContainers"`
	Containers     []corev1.Container `yaml:"containers"`
	Volumes        []corev1.Volume    `yaml:"volumes"`
}

// Update field(s) of a resource using strategic merge patch.
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	_ = corev1.AddToScheme(runtimeScheme)
}

func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,
		},
	})
}

func LoadConfig(configFile string) (*Config, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// mutationRequired checks whether the target resource need to be mutated
func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
	// Skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
		if metadata.Namespace == namespace {
			glog.Infof("Skip mutation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[admissionWebhookAnnotationStatusKey]

	// Determine whether to perform mutation based on annotation for the target resource
	var required bool
	if strings.ToLower(status) == "injected" {
		required = false
	} else {
		switch strings.ToLower(annotations[admissionWebhookAnnotationInjectKey]) {
		default:
			required = false
		case "y", "yes", "true", "on":
			required = true
		}
	}

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v ", metadata.Namespace, metadata.Name, status, required)
	return required
}

// addContainer adds container to pod
func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0

	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// updateBFResource updates resource name and change container's cmd to add Bitfusion
func updateBFResource(targets []corev1.Container, basePath string) (patches []patchOperation, e error) {
	if len(targets) == 0 {
		return patches, nil
	}

	for i, target := range targets {
		if len(target.Command) != 0 {

			// Check bitFusionGPUResourceNum
			gpuNum := target.Resources.Requests[bitFusionGPUResourceNum]

			// Check bitFusionGPUResourcePartial and set fallback
			gpuPartial := target.Resources.Requests[bitFusionGPUResourcePartial]
			gpuMemory := target.Resources.Requests[bitFusionGPUResourceMemory]
			if gpuNum != zeroQuantity && gpuPartial == zeroQuantity {
				glog.Warning("No Partial was provide, use default value 100 which means 100%")
				gpuPartial.Set(100)
				patches = append(patches, patchOperation{
					Op:   "remove",
					Path: basePath + "/" + strconv.Itoa(i) + "/resources/requests/" + bitFusionGPUResourceNumEscape,
				})
			} else if gpuNum != zeroQuantity && gpuPartial != zeroQuantity {
				patches = append(patches, patchOperation{
					Op:   "remove",
					Path: basePath + "/" + strconv.Itoa(i) + "/resources/requests/" + bitFusionGPUResourceNumEscape,
				})
				patches = append(patches, patchOperation{
					Op:   "remove",
					Path: basePath + "/" + strconv.Itoa(i) + "/resources/requests/" + bitFusionGPUResourcePartialEscape,
				})
			} else if gpuNum == zeroQuantity && gpuPartial == zeroQuantity {
				// No patch for this container
				continue
			} else {
				return patches, fmt.Errorf("No gpu num was provided but found percent ")
			}

			gpuPartialNum := gpuPartial.Value()

			// Also return error if exceed 100% or equals 0%
			if gpuPartialNum > 100 || gpuPartialNum == 0 {
				return patches, fmt.Errorf("Invalid %s quantity: %d ", bitFusionGPUResourcePartial, gpuPartialNum)
			}
			var command string
			if gpuMemory != zeroQuantity {
				m, ok := gpuMemory.AsInt64()
				if ok {

					command = fmt.Sprintf("bitfusion run -n %s -m %d", gpuNum.String(), m)
					patches = append(patches, patchOperation{
						Op:   "remove",
						Path: basePath + "/" + strconv.Itoa(i) + "/resources/requests/" + bitFusionGPUResourceMemoryEscape,
					})
				} else {
					glog.Error("gpuMemory.AsInt64 Error")
					return patches, fmt.Errorf("gpuMemory.AsInt64 Error")

				}
			} else {
				command = fmt.Sprintf("bitfusion run -n %s -p %f", gpuNum.String(), float64(gpuPartialNum)/100.0)
			}
			glog.Infof("Request gpu with num %v", gpuNum.String())
			glog.Infof("Request gpu with partial %v", gpuPartial.String())

			hasPrefix := false
			for _, v := range target.Command {

				if strings.ToLower(v) == "/bin/bash" {
					continue
				}
				if strings.ToLower(v) == "-c" {
					continue
				}

				str := strings.TrimSpace(v)
				if strings.HasPrefix(str, "bitfusion") {
					hasPrefix = true
				}

				command += " " + v

			}
			if !hasPrefix {
				cmd := []string{"/bin/bash", "-c", command}
				target.Command = cmd
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  basePath + "/" + strconv.Itoa(i) + "/command",
					Value: cmd,
				})
			}

			// Construct bitFusionGPUResource
			// Remove legacy
			delete(target.Resources.Requests, bitFusionGPUResourceNum)
			delete(target.Resources.Requests, bitFusionGPUResourcePartial)

			// Construct quantity
			gpuQuantity := &resource.Quantity{}
			gpuQuantity.Set(gpuPartialNum * gpuNum.Value())
			target.Resources.Requests[bitFusionGPUResource] = *gpuQuantity

			// Create JSON patch to target containers
			targets[i] = target

			patches = append(patches, patchOperation{
				Op:   "add",
				Path: basePath + "/" + strconv.Itoa(i) + "/resources/requests",
				Value: map[string]resource.Quantity{
					bitFusionGPUResource: *gpuQuantity,
				},
			})
			patches = append(patches, patchOperation{
				Op:   "add",
				Path: basePath + "/" + strconv.Itoa(i) + "/resources",
				Value: map[string]map[string]resource.Quantity{
					"limits": {
						bitFusionGPUResource: *gpuQuantity,
					},
				},
			})
		}
	}
	return patches, nil
}

// updateContainer updates env and volume to container
func updateContainer(targets, source []corev1.Container, basePath string) (patches []patchOperation) {

	for i, container := range targets {
		if container.Resources.Requests[bitFusionGPUResourceNum] == zeroQuantity {
			// Pass if no Bitfusion resource was required
			// It will fail on next step if only gpu-percent was provided
			continue
		}

		container.VolumeMounts = append(container.VolumeMounts, source[0].VolumeMounts...)
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  fmt.Sprintf("%s/%d/volumeMounts", basePath, i),
			Value: container.VolumeMounts,
		})

		container.Env = append(container.Env, source[0].Env...)
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  fmt.Sprintf("%s/%d/env", basePath, i),
			Value: container.Env,
		})

		targets[0] = container

	}
	return patches
}

// addVolume adds volume to pod
func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

// updateAnnotation updates pod's annotation, returns a update list of patchOperation
func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target == nil || target[key] == "" {
			target = map[string]string{}
			patch = append(patch, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			patch = append(patch, patchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + key,
				Value: value,
			})
		}
	}
	return patch
}

// createPatch creates mutation patch for resource
func createPatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	var err error

	patch = append(patch, addContainer(pod.Spec.InitContainers, sidecarConfig.InitContainers, "/spec/initContainers")...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
	patch = append(patch, updateContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)

	bfPatch, err := updateBFResource(pod.Spec.Containers, "/spec/containers")
	if err != nil {
		glog.Errorf("Unable to create json patch for bitfusion resource")
		return nil, err
	}

	patch = append(patch, bfPatch...)

	patchByte, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	return patchByte, nil
}

// mutate is main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// Determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
		glog.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	applyDefaultsWorkaround(whsvr.SidecarConfig.Containers, whsvr.SidecarConfig.Volumes)
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, whsvr.SidecarConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	if err = CopySecret(&req.Namespace); err != nil {
		glog.Errorf("Can't copy secret: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) Serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("Empty body")
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}

	// Verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

// CopySecret copies a secret to target namespace
func CopySecret(namespace *string) error {
	name := "bitfusion-secret"
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Secrets(*namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		err = nil
		glog.Infof("Secrets %s  not found in  namespace  %s \n", name, *namespace)
		secret, err := clientset.CoreV1().Secrets("kube-system").Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		newSecret := &corev1.Secret{
			Data: secret.Data,
			Type: secret.Type,
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: *namespace,
			},
		}
		// Create the secret
		_, err = clientset.CoreV1().Secrets(*namespace).Create(context.TODO(), newSecret, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("Can't create secret: %v", err)
		}
	}
	return err
}
