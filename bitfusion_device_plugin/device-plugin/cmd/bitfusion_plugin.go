/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"strconv"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// Bitfusion Manager
type bfsManager struct {
	devices map[string]*pluginapi.Device
}

// discoverResources is discover resources
func (bfs *bfsManager) discoverResources() bool {
	found := false
	bfs.devices = make(map[string]*pluginapi.Device)
	glog.Info("Discover")
	nums, err := strconv.Atoi(resourceNums)
	if err != nil {
		glog.Error(err)
	}
	// Unlimited resources
	for i := 0; i < nums; i += 1 {
		dev := pluginapi.Device{ID: strconv.Itoa(i), Health: pluginapi.Healthy}
		bfs.devices[strconv.Itoa(i)] = &dev
		found = true
	}
	glog.Info("Discover Resources over")

	return found
}

// ListAndWatch returns a stream of List of Devices .
// Whenever a Device state change or a Device disappears.
// ListAndWatch returns the new list
func (bfs *bfsManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Info("ListAndWatch start\n")
	for {
		glog.Info("ListAndWatch Pending.............\n")
		bfs.discoverResources()
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range bfs.devices {
			glog.Info("Dev ", dev)
			resp.Devices = append(resp.Devices, dev)
		}
		glog.Info("Resp Devices ", resp.Devices)
		if err := stream.Send(resp); err != nil {
			glog.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(interval * time.Second)
	}
	return nil
}

// Allocate is called during container creation so that the Device .
// Plugin can run device specific operations and instruct
// Kubelet of the steps to make the Device available in the container
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

// PreStartContainer is called, if indicated by Device Plugin during registeration phase,
// before each container start.
// Device plugin can run device specific operations such as resetting the device before making devices available to the container
func (*bfsManager) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return new(pluginapi.PreStartContainerResponse), nil
}

// GetDevicePluginOptions returns options to be communicated with Device Manager
func (*bfsManager) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

// GetPreferredAllocation returns a preferred set of devices to allocate from a list of available ones.
// The resulting preferred allocation is not guaranteed to be the allocation ultimately performed by the devicemanager.
// It is only designed to help the device manager make a more informed allocation decision when possible.
func (*bfsManager) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func NewbfsManager() (*bfsManager, error) {
	return &bfsManager{
		devices: make(map[string]*pluginapi.Device),
	}, nil
}
