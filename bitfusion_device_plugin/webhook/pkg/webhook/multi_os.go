/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package webhook

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	yamlv2 "gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var (
	BitfusionClientMap *map[string]map[string]BFClientConfig
)

// Bitfusion client binary path and environment variables value of LD_LIBRARY_PATH
type BFClientConfig struct {
	BinaryPath  string
	EnvVariable string
}

// BitfusionClients configuration for each Bitfusion client in different OS
type BitfusionClients struct {
	BitfusionVersion string `yaml:"BitfusionVersion"`
	OSVersion        string `yaml:"OSVersion"`
	BinaryPath       string `yaml:"BinaryPath"`
	EnvVariable      string `yaml:"EnvVariable"`
}

// BitfusionClientDistro struct
type BitfusionClientDistro struct {
	BitfusionClients []BitfusionClients `yaml:"BitfusionClients"`
}

// addContainer adds container to pod
func addContainer(target, added []corev1.Container, basePath string, bfClientConfig BFClientConfig) (patch []patchOperation) {
	first := len(target) == 0

	var value interface{}
	for _, add := range added {
		index := strings.Index(bfClientConfig.EnvVariable, "/opt/bitfusion")
		optPath := bfClientConfig.EnvVariable[0:index]
		// /bin/bash, -c, "command"
		add.Command[2] = strings.Replace(add.Command[2], "BITFUSION_CLIENT_OPT_PATH", optPath+"/opt/bitfusion/*", 1)

		glog.Infof("Command of InitContainer : %v", add.Command[2])
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
		env := corev1.EnvVar{Name: "LD_LIBRARY_PATH", Value: bfClientConfig.EnvVariable}
		container.Env = append(container.Env, env)
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

// mutate is main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	response := &v1beta1.AdmissionResponse{}

	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Could not unmarshal raw object: %v", err)
		response.Result = &metav1.Status{Message: err.Error()}
		return response
	}

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// Determine whether to perform mutation
	if !mutationRequired(ignoredNamespaces, &pod.ObjectMeta) {
		glog.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		response.Allowed = true
		return response
	}

	// If user did not specify the GuestOS annotation, webhook will do nothing with the container
	os := getGuestOS(&pod.ObjectMeta)
	bfVersion := getBfVersion(&pod.ObjectMeta)
	clientMap := *BitfusionClientMap
	if os == "" || bfVersion == "" {
		response.Allowed = true
		return response
	} else {
		if _, ok := clientMap[os][bfVersion]; !ok {
			glog.Errorf("Could not find Bitfusion client info, OS=%v BFVersion=%v", os, bfVersion)
			response.Result = &metav1.Status{Message: "Could not find Bitfusion client info"}
			return response
		}
	}

	applyDefaultsWorkaround(whsvr.SidecarConfig.Containers, whsvr.SidecarConfig.Volumes)
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patchBytes, err := createPatch(&pod, whsvr.SidecarConfig, annotations, clientMap[os][bfVersion])
	if err != nil {
		response.Result = &metav1.Status{Message: err.Error()}
		return response
	}

	if err = CopySecret(&req.Namespace); err != nil {
		glog.Errorf("Can't copy secret: %v", err)
		response.Result = &metav1.Status{Message: err.Error()}
		return response
	}

	response.Allowed = true
	response.Patch = patchBytes
	response.PatchType = func() *v1beta1.PatchType {
		pt := v1beta1.PatchTypeJSONPatch
		return &pt
	}()

	return response
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

	bfPatch, err := updateBFResource(pod.Spec.Containers, "/spec/containers", bfClientConfig)
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
