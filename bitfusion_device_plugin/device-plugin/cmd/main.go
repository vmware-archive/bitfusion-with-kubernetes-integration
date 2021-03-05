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
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"
)

var socketName string
var resourceName string
var interval time.Duration
var resourceNums string

func Register(kubeletEndpoint, pluginEndpoint, resourceName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	glog.Infof("reqt ==== %s", reqt)

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
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
	glog.Infof("---------  %s", os.Args)
	intervalTemp, _ := strconv.ParseInt(os.Args[1], 10, 64)
	interval = time.Duration(intervalTemp)
	socketName = os.Args[2] //"bfs"
	resourceName = os.Args[3]
	resourceNums = os.Args[4]

	flag.Lookup("logtostderr").Value.Set("true")

	bfs, err := NewbfsManager()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}

	//pluginEndpoint := fmt.Sprintf("%s-%d.sock", socketName, time.Now().Unix())
	pluginEndpoint := socketName
	_, _ = ExecCommand("rm", "-rf", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
	//serverStarted := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	// Starts device plugin service.
	go func() {
		defer wg.Done()
		glog.Infof("DevicePluginPath %s, pluginEndpoint %s\n", pluginapi.DevicePluginPath, pluginEndpoint)
		glog.Infof("device-plugin start server at: %s\n", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		lis, err := net.Listen("unix", path.Join(pluginapi.DevicePluginPath, pluginEndpoint))
		if err != nil {
			glog.Fatal(err)
			return
		}
		grpcServer := grpc.NewServer()
		pluginapi.RegisterDevicePluginServer(grpcServer, bfs)
		grpcServer.Serve(lis)
	}()

	time.Sleep(35 * time.Second)
	// Registers with Kubelet.
	err = Register(pluginapi.KubeletSocket, pluginEndpoint, resourceName)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Info("device-plugin registered\n")
	wg.Wait()
}
