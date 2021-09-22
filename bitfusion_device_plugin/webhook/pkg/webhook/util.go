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
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	yamlv2 "gopkg.in/yaml.v2"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math"
	"os"
	"strconv"
	"strings"
)

// addContainer adds container to pod
func addContainer(target, added []corev1.Container, basePath string, bfClientConfig BFClientConfig) (patch []patchOperation) {
	first := len(target) == 0

	var value interface{}
	for _, add := range added {
		index := strings.Index(bfClientConfig.EnvVariable, "/opt/bitfusion")
		optPath := bfClientConfig.EnvVariable[0:index]
		// /bin/bash, -c, "command"
		// The original data cannot be changed, the previous approach resulted in changes to the original data，so deep replication is used
		container := add.DeepCopy()
		container.Command[2] = strings.Replace(container.Command[2], "BITFUSION_CLIENT_OPT_PATH", optPath+"/opt/bitfusion/*", 1)

		glog.Infof("Command of InitContainer : %v", container.Command[2])

		path := basePath
		if first {
			first = false
			value = []corev1.Container{*container}
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

func ConstructBitfusionDistroInfo(configFile string) (*BitfusionClientDistro, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var result BitfusionClientDistro

	if err := yamlv2.Unmarshal(data, &result); err != nil {
		// error handling
		return nil, err
	}
	return &result, nil
}

func getGuestOS(metadata *metav1.ObjectMeta) string {
	annotations := metadata.GetAnnotations()
	if annotations != nil {
		os := annotations[guestOS]
		return os
	}
	return ""
}

func getBfVersion(metadata *metav1.ObjectMeta) string {
	annotations := metadata.GetAnnotations()
	if annotations != nil {
		bfVer := annotations[bfVersion]
		return bfVer
	}
	return ""
}

// updateContainer updates env and volume to container
func updateContainer(targets, source []corev1.Container, basePath string, bfClientConfig BFClientConfig) (patches []patchOperation) {

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

		//container.Env = append(container.Env, source[0].Env...)
		index := -1
		for i := range container.Env {
			glog.Infof("container.Env[i].Name = %s", container.Env[i].Name)
			if container.Env[i].Name == "LD_LIBRARY_PATH" {
				index = i
				glog.Infof("index = %d", index)
			}
		}
		if index != -1 {
			env := corev1.EnvVar{Name: "LD_LIBRARY_PATH", Value: bfClientConfig.EnvVariable + ":" + container.Env[index].Value}
			container.Env[index] = env
		} else {
			env := corev1.EnvVar{Name: "LD_LIBRARY_PATH", Value: bfClientConfig.EnvVariable}
			container.Env = append(container.Env, env)
		}
		//env = corev1.EnvVar{Name: "PATH", Value: bfClientConfig.BinaryPath + ":$PATH"}
		//container.Env = append(container.Env, env)
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  fmt.Sprintf("%s/%d/env", basePath, i),
			Value: container.Env,
		})

		targets[0] = container

	}
	return patches
}

// createPatch creates mutation patch for resource
func createPatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string, bfClientConfig BFClientConfig) ([]byte, error) {
	var patch []patchOperation

	var err error
	initContainers := updateInitContainersResources(pod.Spec.Containers, sidecarConfig.InitContainers)
	patch = append(patch, addContainer(pod.Spec.InitContainers, initContainers, "/spec/initContainers", bfClientConfig)...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)
	patch = append(patch, updateContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers", bfClientConfig)...)

	glog.Infof("sidecarConfig: %v", sidecarConfig.InitContainers)
	glog.Infof("sidecarConfig.Containers: %v", sidecarConfig.Containers[0].VolumeMounts)
	glog.Infof("patch: %v", patch)

	bfPatch, err := updateBFResource(pod.Spec.Containers, "/spec/containers", bfClientConfig, annotations)
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

func updateInitContainersResources(target, added []corev1.Container) []corev1.Container {
	maxCpu := zeroQuantity
	maxMem := zeroQuantity
	maxReqCpu := zeroQuantity
	maxReqMem := zeroQuantity
	for _, container := range target {
		if cpuNum, has := container.Resources.Limits["cpu"]; has {
			if cpuNum.Cmp(maxCpu) > 0 {
				maxCpu = cpuNum
			}
		}
		if cpuNum, has := container.Resources.Requests["cpu"]; has {
			if cpuNum.Cmp(maxReqCpu) > 0 {
				maxReqCpu = cpuNum
			}
		}
		if memNum, has := container.Resources.Limits["memory"]; has {
			if memNum.Cmp(maxMem) > 0 {
				maxMem = memNum
			}
		}
		if memNum, has := container.Resources.Requests["memory"]; has {
			if memNum.Cmp(maxReqMem) > 0 {
				maxReqMem = memNum
			}
		}
	}
	for i := range added {
		glog.Infof("maxCpu = %v", maxMem)
		if added[i].Resources.Limits == nil {
			added[i].Resources.Limits = make(corev1.ResourceList)
		}
		added[i].Resources.Limits["cpu"] = maxCpu
		glog.Infof("container.Resources.Limits  == %v", added[i].Resources.Limits)
	}

	for i := range added {
		glog.Infof("maxMem = %v", maxMem)
		if added[i].Resources.Limits == nil {
			added[i].Resources.Limits = make(corev1.ResourceList)
		}
		added[i].Resources.Limits["memory"] = maxMem
		glog.Infof("container.Resources.Limits  == %v", added[i].Resources.Limits)
	}

	for i := range added {
		glog.Infof("maxReqCpu = %v", maxReqCpu)
		if added[i].Resources.Requests == nil {
			added[i].Resources.Requests = make(corev1.ResourceList)
		}
		added[i].Resources.Requests["cpu"] = maxReqCpu
		glog.Infof("container.Resources.Requests  == %v", added[i].Resources.Requests)
	}

	for i := range added {
		glog.Infof("maxReqMem = %v", maxReqMem)
		if added[i].Resources.Requests == nil {
			added[i].Resources.Requests = make(corev1.ResourceList)
		}
		added[i].Resources.Requests["memory"] = maxReqMem
		glog.Infof("container.Resources.Requests  == %v", added[i].Resources.Requests)
	}
	return added
}

// copySecret copies a secret to target namespace
func copySecret(namespace *string) error {
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

// updateBFResource updates resource name and change container's cmd to add Bitfusion
func updateBFResource(targets []corev1.Container, basePath string, bfClientConfig BFClientConfig, annotations map[string]string) (patches []patchOperation, e error) {
	if len(targets) == 0 {
		return patches, nil
	}

	for i, target := range targets {
		if len(target.Command) != 0 {

			// Check bitFusionGPUResourceNum
			gpuNum := target.Resources.Requests[bitFusionGPUResourceNum]
			if gpuNum.Value() <= 0 {
				return patches, fmt.Errorf("gpuNum.Value() Error ")
			}

			// Check bitFusionGPUResourcePartial and set fallback
			gpuPartial := target.Resources.Requests[bitFusionGPUResourcePartial]
			gpuMemory := target.Resources.Requests[bitFusionGPUResourceMemory]
			if gpuNum != zeroQuantity && gpuPartial == zeroQuantity {
				glog.Warning("No Partial was provide, use default value 100 which means 100%")
				gpuPartial.Set(100)
				delete(target.Resources.Requests, bitFusionGPUResourceNum)
				delete(target.Resources.Limits, bitFusionGPUResourceNum)
			} else if gpuNum != zeroQuantity && gpuPartial != zeroQuantity {
				delete(target.Resources.Requests, bitFusionGPUResourceNum)
				delete(target.Resources.Limits, bitFusionGPUResourceNum)
				delete(target.Resources.Requests, bitFusionGPUResourcePartial)
				delete(target.Resources.Limits, bitFusionGPUResourcePartial)
			} else if gpuNum == zeroQuantity && gpuPartial == zeroQuantity {
				// No patch for this container
				continue
			} else {
				return patches, fmt.Errorf("No gpu num was provided but found percent ")
			}

			gpuPartialNum := gpuPartial.Value()

			// Also return error if exceed 100% or equals 0%
			if gpuPartialNum > 100 || gpuPartialNum <= 0 {
				return patches, fmt.Errorf("Invalid %s quantity: %d ", bitFusionGPUResourcePartial, gpuPartialNum)
			}
			var command string
			var totalMem resource.Quantity
			totalMemStr := os.Getenv("TOTAL_GPU_MEMORY")
			glog.Infof("totalMemStr = %s", totalMemStr)
			if gpuMemory != zeroQuantity {
				totalMem = resource.MustParse(totalMemStr)
				glog.Infof("totalMem = %d", totalMem.Value())
				glog.Infof("gpuMemory = %v", gpuMemory)
				m, ok := gpuMemory.AsInt64()
				if ok {
					m = m / 1000000
					glog.Infof("gpuMemory = %d", m)
					if m <= 0 || m >= totalMem.Value() {
						glog.Error("Memory value Error")
						return patches, fmt.Errorf("Memory value Error ")
					}

					if value, has := annotations[admissionWebhookAnnotationFilterKey]; has {
						command = fmt.Sprintf(bfClientConfig.BinaryPath+" run -n %s -m %d --filter %s", gpuNum.String(), m, value)
					} else {
						command = fmt.Sprintf(bfClientConfig.BinaryPath+" run -n %s -m %d", gpuNum.String(), m)
					}

					delete(target.Resources.Requests, bitFusionGPUResourceMemory)
					delete(target.Resources.Limits, bitFusionGPUResourceMemory)
				} else {
					glog.Error("gpuMemory.AsInt64 Error")
					return patches, fmt.Errorf("gpuMemory.AsInt64 Error")

				}
			} else {

				if value, has := annotations[admissionWebhookAnnotationFilterKey]; has {
					command = fmt.Sprintf(bfClientConfig.BinaryPath+" run -n %d -p %f --filter %s", gpuNum.Value(), float64(gpuPartialNum)/100.0, value)
				} else {
					command = fmt.Sprintf(bfClientConfig.BinaryPath+" run -n %d -p %f ", gpuNum.Value(), float64(gpuPartialNum)/100.0)
				}

			}
			glog.Infof("Command : %s", command)
			glog.Infof("Request gpu with num %v", gpuNum.Value())
			glog.Infof("Request gpu with partial %v", gpuPartial.Value())

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
			if !hasPrefix && injectionStatus != bitFusionOnlyInjection {
				cmd := []string{"/bin/bash", "-c", command}
				target.Command = cmd
				patches = append(patches, patchOperation{
					Op:    "replace",
					Path:  basePath + "/" + strconv.Itoa(i) + "/command",
					Value: cmd,
				})
			}

			// Construct quantity
			gpuQuantity := &resource.Quantity{}
			if gpuMemory != zeroQuantity {
				rate := float64(gpuMemory.Value()/1000000) / float64(totalMem.Value())
				glog.Infof("rate = %f", rate)
				gpuQuantity.Set(int64(math.Ceil(rate * float64(gpuNum.Value()) * 100)))
			} else {
				gpuQuantity.Set(gpuPartialNum * gpuNum.Value())
			}
			target.Resources.Requests[bitFusionGPUResource] = *gpuQuantity
			target.Resources.Limits[bitFusionGPUResource] = *gpuQuantity

			// Create JSON patch to target containers
			targets[i] = target

			patches = append(patches, patchOperation{
				Op:   "replace",
				Path: basePath + "/" + strconv.Itoa(i) + "/resources",
				Value: map[string]corev1.ResourceList{
					"limits": target.Resources.Limits,
				},
			})
			patches = append(patches, patchOperation{
				Op:    "replace",
				Path:  basePath + "/" + strconv.Itoa(i) + "/resources/requests",
				Value: target.Resources.Requests,
			})
			glog.Infof("Now patches === %v", patches)
		}
	}
	return patches, nil
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
	injectionStatus = ""
	if strings.ToLower(status) == "injected" {
		required = false
	} else {
		switch strings.ToLower(annotations[admissionWebhookAnnotationInjectKey]) {
		default:
			required = false
		case "y", "yes", "true", "on", "all":
			required = true
		case bitFusionOnlyInjection:
			injectionStatus = bitFusionOnlyInjection
			required = true
		}
	}

	glog.Infof("Mutation policy for %v/%v: status: %q required:%v ", metadata.Namespace, metadata.Name, status, required)
	return required
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
