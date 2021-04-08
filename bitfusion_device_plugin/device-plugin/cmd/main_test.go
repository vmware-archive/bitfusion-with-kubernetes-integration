/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func TestExecCommand(t *testing.T) {
	out, err := ExecCommand("ls")
	t.Log(out)
	t.Log(err)
	assert.Equal(t, err, nil)
}

func TestNewbfsManager(t *testing.T) {
	bfs, err := NewbfsManager()
	t.Log(bfs)
	t.Log(err)
	assert.Equal(t, err, nil)
}

func TestRegister(t *testing.T) {
	err := Register(pluginapi.KubeletSocket, "test", "test.io")
	t.Log(err)
	assert.Equal(t, err, nil)
}
