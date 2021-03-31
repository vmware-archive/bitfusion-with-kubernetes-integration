/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package validation_webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	admissionWebhookAnnotationStatusKey = "auto-management/status"
)

type ValidateWebhookServer struct {
	Server *http.Server
}

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

// Serve method for webhook server
func (webhookServer *ValidateWebhookServer) Serve(w http.ResponseWriter, r *http.Request) {

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
		var pod corev1.Pod
		req := ar.Request
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return
		}

		glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
			req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

		// Call some functions to check resource
		admissionResponse = webhookServer.validate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	admissionReview.Kind = "AdmissionReview"
	admissionReview.APIVersion = "admission.k8s.io/v1"
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
	glog.Infof("Validating webhook ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}

}

// validate application resource exists
func (webhookServer *ValidateWebhookServer) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
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

	annotations := pod.ObjectMeta.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	glog.Infof("Pod Annotations %v", pod.Annotations)
	status := annotations[admissionWebhookAnnotationStatusKey]
	glog.Infof("Pod status %v", status)

	if strings.ToLower(status) == "injected" {
		glog.Infof("Injected pod")
		// Check if there's Bitfusion resource
		config, err := rest.InClusterConfig()
		if err != nil {
			glog.Error("InClusterConfig Failed")
		}

		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			glog.Error("InClusterConfig Failed")
			glog.Error(err.Error())
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}

		nodes := clientSet.CoreV1().Nodes()
		list, err := nodes.List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			glog.Infof("Error: %v", err)
		}

		var zeroValue resource.Quantity
		zeroValue.Set(0)
		glog.Infof("Node info: %v", nodes)
		glog.Infof("Node info: %v", list)

		// Compare whether the requested resource actually exists
		cons := pod.Spec.Containers
		if len(cons) != 0 && len(list.Items) != 0 {
			allRes := make(map[corev1.ResourceName]resource.Quantity)
			for _, node := range list.Items {
				for k, v := range node.Status.Allocatable {
					if !v.IsZero() {
						allRes[k] = v
					}
				}
			}

			reqRes := make([]corev1.ResourceName, 0, 10)
			for _, con := range cons {
				for k, _ := range con.Resources.Requests {
					reqRes = append(reqRes, k)
				}
			}
			glog.Infof("AllRes: %v", allRes)
			glog.Infof("ReqRes: %v", reqRes)
			ok := true
			if len(reqRes) != 0 {
				for _, rv := range reqRes {
					if _, has := allRes[rv]; !has {
						ok = false
						break
					}
				}

				if !ok {
					glog.Infof("Resource validation failed")
					war := []string{"Resource validation failed"}
					return &v1beta1.AdmissionResponse{
						Allowed:  false,
						Warnings: war,
						Result: &metav1.Status{
							Message: "Resource validation failed",
						},
					}
				}
			}

		}
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}

	}

	return &v1beta1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: true,
	}

}
