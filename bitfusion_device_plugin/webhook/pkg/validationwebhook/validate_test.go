/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package validationwebhook

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

var PodPath = "../../../example/pod.yaml"

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

func TestValidateWebhookServer_Validate(t *testing.T) {
	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: conver(PodPath),
			},
		},
	}
	validateWebhookSv := &ValidateWebhookServer{
		Server: &http.Server{
			Addr: "8888",
			//TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}
	admissionResponse := validateWebhookSv.validate(&ar)
	t.Log(admissionResponse)
	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte(""),
			},
		},
	}
	admissionResponse = validateWebhookSv.validate(&ar)

	t.Log(admissionResponse)
	assert.Equal(t, admissionResponse.Allowed, true)
}
