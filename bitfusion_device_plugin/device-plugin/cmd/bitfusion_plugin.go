/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"github.com/golang/glog"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"strconv"
	"time"
)

// bfsManager manages
type bfsManager struct {
	devices map[string]*pluginapi.Device
}

func (bfs *bfsManager) discoverResources() bool {
	found := false
	bfs.devices = make(map[string]*pluginapi.Device)
	glog.Info("discover")
	nums, _ := strconv.Atoi(resourceNums)
	// Unlimited resources
	for i := 0; i < nums; i += 1 {
		dev := pluginapi.Device{ID: strconv.Itoa(i), Health: pluginapi.Healthy}
		bfs.devices[strconv.Itoa(i)] = &dev
		found = true
	}
	glog.Info("discover Resources over")

	return found
}

// Implements DevicePlugin service functions
func (bfs *bfsManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Info("device-plugin: ListAndWatch start\n")
	for {
		glog.Info("device-plugin: ListAndWatch Pending.............\n")
		bfs.discoverResources()
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range bfs.devices {
			glog.Info("dev ", dev)
			resp.Devices = append(resp.Devices, dev)
		}
		glog.Info("resp.Devices ", resp.Devices)
		if err := stream.Send(resp); err != nil {
			glog.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(interval * time.Second)
	}
	return nil
}
func (bfs *bfsManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {

	glog.Info("Allocate")
	var response pluginapi.AllocateResponse
	var car pluginapi.ContainerAllocateResponse
	for _, req := range rqt.ContainerRequests {
		glog.Infof("Allocating device IDs: %s", req.DevicesIDs)
		response.ContainerResponses = append(response.ContainerResponses, &car)
	}

	return &response, nil
}
func (*bfsManager) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return new(pluginapi.PreStartContainerResponse), nil
}
func (*bfsManager) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}
func (*bfsManager) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func NewbfsManager() (*bfsManager, error) {
	return &bfsManager{
		devices: make(map[string]*pluginapi.Device),
	}, nil
}
