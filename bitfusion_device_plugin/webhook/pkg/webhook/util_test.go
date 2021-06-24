/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package webhook

import (
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	testCfg := Cfg.DeepCopy()
	key := "sidecarconfig.yaml"
	native, has := testCfg.Data[key]
	if !has {
		t.Fatalf("The %s doesn't exist", key)
		t.Fatal(testCfg.Data)
	}
	content := []byte(native)

	err := ioutil.WriteFile(key, content, 0644)
	if err != nil {
		t.Fatalf("Write File %s Error:", key)
	}

	if sidecarConfig, err := LoadConfig(key); err != nil {
		t.Fatalf("LocdConfig Error: %s", err)
	} else {
		t.Log(sidecarConfig)
		if sder, err := yaml.Marshal(sidecarConfig); err != nil {
			t.Fatal(err)
		} else {
			var vfcfg Config
			if err := yaml.Unmarshal(content, &vfcfg); err != nil {
				log.Fatal(err)
			} else {
				t.Log(vfcfg)
				vf, _ := yaml.Marshal(vfcfg)
				for i := 0; i < len(vf); i += 1 {
					assert.Equal(t, vf[i], sder[i])
				}
			}
		}
	}
}

func TestAddContainer(t *testing.T) {
	pod := StaticPod
	bfClientConfig := BFClientConfig{"/bitfusion/bitfusion-client-centos7-2.5.0-10/usr/bin/bitfusion",
		"/bitfusion/bitfusion-client-centos7-2.5.0-10/opt/bitfusion/2.5.0-fd3e4839/x86_64-linux-gnu/lib/:$LD_LIBRARY_PATH"}
	patch := addContainer(pod.Spec.InitContainers, TestSidecarConfig.InitContainers, "/spec/initContainers", bfClientConfig)
	assert.Equal(t, len(patch), 1)
	patch = addContainer(pod.Spec.Containers, TestSidecarConfig.Containers, "/spec/containers", bfClientConfig)
	assert.Equal(t, len(patch), 1)
}
func TestAddVolume(t *testing.T) {
	pod := StaticPod
	patch := addVolume(pod.Spec.Volumes, TestSidecarConfig.Volumes, "/spec/volumes")
	assert.Equal(t, len(patch), 1)
}

func TestUpdateAnnotation(t *testing.T) {
	pod := StaticPod
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	patch := updateAnnotation(pod.Annotations, annotations)
	assert.Equal(t, len(patch), 1)

	annotations = map[string]string{"test": ""}
	target := map[string]string{"test": "test"}
	patch = updateAnnotation(target, annotations)
	assert.Equal(t, len(patch), 1)
}

func TestCreatePatch(t *testing.T) {
	pod := StaticPod
	annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
	bfClientConfig := BFClientConfig{"/bitfusion/bitfusion-client-centos7-2.5.0-10/usr/bin/bitfusion",
		"/bitfusion/bitfusion-client-centos7-2.5.0-10/opt/bitfusion/2.5.0-fd3e4839/x86_64-linux-gnu/lib/:$LD_LIBRARY_PATH"}
	bytes, err := createPatch(&pod, &TestSidecarConfig, annotations, bfClientConfig)
	fmt.Print(bytes)
	assert.Equal(t, err, nil)
	mpod := StaticMemPod
	bytes, err = createPatch(&mpod, &TestSidecarConfig, annotations, bfClientConfig)
	fmt.Print(bytes)
	assert.Equal(t, err, nil)

}

func TestMutationRequired(t *testing.T) {
	pod := StaticPod
	res := mutationRequired(ignoredNamespaces, &pod.ObjectMeta)
	assert.Equal(t, res, true)
	mutationRequired([]string{"injection"}, &pod.ObjectMeta)
}
