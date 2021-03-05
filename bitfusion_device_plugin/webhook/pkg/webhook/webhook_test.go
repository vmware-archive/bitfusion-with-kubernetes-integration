/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package webhook

import (
 "bytes"
 "encoding/json"
 "io/ioutil"
 "log"
 "testing"

 "github.com/ghodss/yaml"
 "github.com/stretchr/testify/assert"
 corev1 "k8s.io/api/core/v1"
 "k8s.io/apimachinery/pkg/api/resource"
 "k8s.io/apimachinery/pkg/runtime"
 yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

var PodPath = "../../../example/pod.yaml"
var StaticPod corev1.Pod
var CfgPath = "../../deployment/configmap.yaml"
var Cfg corev1.ConfigMap

var vfcfgstr string = `initContainers:
- name: vfinitname
  image: vfinitimage
containers:
- name: vfcontainername
  image: vfcontainerimage
volumes:
- name: vfvolumes
  emptyDir: {}
`

var TestSidecarConfig Config

func init() {
 if err := json.Unmarshal(conver(PodPath), &StaticPod); err != nil {
  log.Fatal(err)
 }
 if err := json.Unmarshal(conver(CfgPath), &Cfg); err != nil {
  log.Fatal(err)
 }
 if err := yaml.Unmarshal([]byte(vfcfgstr), &TestSidecarConfig); err != nil {
  log.Fatal(err)
 }
}

func conver(path string) []byte {
 content, err := ioutil.ReadFile(path)
 if err != nil {
  log.Fatal(err)
 }

 decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(content), 100)
 var rawObj runtime.RawExtension
 if err = decoder.Decode(&rawObj); err != nil {
  log.Fatal(err)
 }

 return rawObj.Raw
}

func TestLoadConfig(t *testing.T) {
 testCfg := Cfg.DeepCopy()
 key := "sidecarconfig.yaml"
 native, has := testCfg.Data[key]
 if ! has {
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
 patch := addContainer(pod.Spec.InitContainers, TestSidecarConfig.InitContainers, "/spec/initContainers")
 assert.Equal(t, len(patch), 1)
 patch = addContainer(pod.Spec.Containers, TestSidecarConfig.Containers, "/spec/containers")
 assert.Equal(t, len(patch), 1)
}
func TestAddVolume(t *testing.T)        {
 pod := StaticPod
 patch := addVolume(pod.Spec.Volumes, TestSidecarConfig.Volumes, "/spec/volumes")
 assert.Equal(t, len(patch), 1)
}
func TestUpdateAnnotation(t *testing.T) {
 pod := StaticPod
 annotations := map[string]string{admissionWebhookAnnotationStatusKey: "injected"}
 patch := updateAnnotation(pod.Annotations, annotations)
 assert.Equal(t, len(patch), 1)
}

func TestUpdateBFResource(t *testing.T) {
 testPod := StaticPod.DeepCopy()

 var verifyList []int64
 var emptyQuantity resource.Quantity

 for _, container := range testPod.Spec.Containers {
  gpuNum := container.Resources.Requests[bitFusionGPUResourceNum]
  if gpuNum == emptyQuantity {
   continue
  }
  gpuPartial := container.Resources.Requests[bitFusionGPUResourcePartial]
  if gpuPartial == emptyQuantity {
   gpuPartial.Set(100)
  }

  verifyList = append(verifyList, gpuNum.Value()*gpuPartial.Value())
 }

 patchs, err := updateBFResource(testPod.Spec.Containers, "spec/containers")
 if err != nil {
  t.Fatal(err)
 }

 for _, patch := range patchs {
  t.Log("Op: ", patch.Op)
  t.Log("Path ", patch.Path)
  t.Log("Value: ", patch.Value)
 }

 for _, container := range testPod.Spec.Containers {
  gpuResource := container.Resources.Requests[bitFusionGPUResource]
  if gpuResource != emptyQuantity {
   assert.Equal(t, gpuResource.Value(), verifyList[0])
  }

  verifyList = verifyList[1:]
 }

}
