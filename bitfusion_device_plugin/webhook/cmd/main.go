/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/vmware/bitfusion-device-plugin/pkg/validationwebhook"
	mutatingWebhook "github.com/vmware/bitfusion-device-plugin/pkg/webhook"

	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Build a map to store Bitfusion client information
func buildBitfusionClientMap(distroInfo *mutatingWebhook.BitfusionClientDistro) *map[string]map[string]mutatingWebhook.BFClientConfig {
	clientMap := make(map[string]map[string]mutatingWebhook.BFClientConfig)
	for _, bfClient := range distroInfo.BitfusionClients {
		if _, has := clientMap[bfClient.OSVersion]; !has {
			clientMap[bfClient.OSVersion] = make(map[string]mutatingWebhook.BFClientConfig)
		}
		clientMap[bfClient.OSVersion][bfClient.BitfusionVersion] = mutatingWebhook.BFClientConfig{
			BinaryPath: bfClient.BinaryPath, EnvVariable: bfClient.EnvVariable}
	}
	return &clientMap
}

func main() {
	var parameters mutatingWebhook.WhSvrParameters

	// Get command line parameters
	flag.IntVar(&parameters.Port, "port", 8443, "Webhook server port.")

	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem",
		"File containing the x509 Certificate for HTTPS.")

	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem",
		"File containing the x509 private key to --tlsCertFile.")

	flag.StringVar(&parameters.SidecarCfgFile, "sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml",
		"File containing the mutation configuration.")

	flag.StringVar(&parameters.BitfusionClientConfig, "bitfusionClientConfig",
		"/etc/webhook/bitfusion-client-config/bitfusion-client-config.yaml",
		"File containing the Bitfusion client configuration.")

	flag.Parse()

	distroInfo, err := mutatingWebhook.ConstructBitfusionDistroInfo(parameters.BitfusionClientConfig)
	if err != nil {
		glog.Errorf("Failed to build Bitfusion distro info")
	}

	mutatingWebhook.BitfusionClientMap = buildBitfusionClientMap(distroInfo)

	sidecarConfig, err := mutatingWebhook.LoadConfig(parameters.SidecarCfgFile)
	if err != nil {
		glog.Errorf("Failed to load configuration: %v", err)
	}

	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
	}

	mutatingWebhookSv := &mutatingWebhook.WebhookServer{
		SidecarConfig: sidecarConfig,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	validateWebhookSv := &validationwebhook.ValidateWebhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// Define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", mutatingWebhookSv.Serve)
	glog.Infof("HandleFunc validate")
	mux.HandleFunc("/validate", validateWebhookSv.Serve)
	mutatingWebhookSv.Server.Handler = mux

	// Start webhook server in new rountine
	go func() {
		if err := mutatingWebhookSv.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	// Listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	err = mutatingWebhookSv.Server.Shutdown(context.Background())
	if err != nil {
		glog.Fatal(err)
	}
}
