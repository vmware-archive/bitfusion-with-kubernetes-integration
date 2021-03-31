/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

var socketName string
var resourceName string
var interval time.Duration
var resourceNums string

// Register the device plugin
func Register(kubeletEndpoint, pluginEndpoint, resourceName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("can't connect to kubelet service: %v ", err)
	}
	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("can't register to kubelet service: %v ", err)
	}
	return nil
}

func ExecCommand(cmdName string, arg ...string) (bytes.Buffer, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		glog.Info("CMD--" + cmdName + ": " + fmt.Sprint(err) + ": " + stderr.String())
	}

	return out, nil
}

func main() {
	flag.Parse()
	glog.Info("Starting main \n")
	glog.Infof("Args: %s ", os.Args)
	intervalTemp, _ := strconv.ParseInt(os.Args[1], 10, 64)
	interval = time.Duration(intervalTemp)
	socketName = os.Args[2]
	resourceName = os.Args[3]
	resourceNums = os.Args[4]

	err := flag.Lookup("logtostderr").Value.Set("true")

	if err != nil {
		glog.Error(err)
	}

	bfs, err := NewbfsManager()
	if err != nil {
		glog.Fatal(err)
	}

	pluginEndpoint := socketName
	_, err = ExecCommand("rm", "-rf", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
	if err != nil {
		glog.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	ch := make(chan int)
	// Starts device plugin service.
	go func() {
		defer wg.Done()
		glog.Infof("Device Plugin path %s, plugin endpoint %s\n", pluginapi.DevicePluginPath, pluginEndpoint)
		glog.Infof("Device Plugin start server at: %s\n", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		lis, err := net.Listen("unix", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		if err != nil {
			glog.Fatal(err)
			return
		}
		grpcServer := grpc.NewServer()
		pluginapi.RegisterDevicePluginServer(grpcServer, bfs)
		close(ch)
		if err := grpcServer.Serve(lis); err != nil {
			glog.Fatal(err)
		}
	}()
	<-ch
	// Register to Kubelet.
	err = Register(pluginapi.KubeletSocket, pluginEndpoint, resourceName)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Info("Device Plugin registered\n")
	wg.Wait()
}
