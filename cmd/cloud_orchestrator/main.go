// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/google/cloud-android-orchestration/pkg/app"
	"github.com/google/cloud-android-orchestration/pkg/app/net"
	"github.com/google/cloud-android-orchestration/pkg/app/net/gcp"
	"github.com/google/cloud-android-orchestration/pkg/app/net/unix"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
)

func LoadConfiguration() *app.Config {
	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	log.Println("Main Configuration:")
	log.Println("  Instance Manager Type: " + config.InstanceManager.Type)
	if config.InstanceManager.Type == app.GCEIMType {
		log.Println("  GCP Project: " + config.InstanceManager.GCP.ProjectID)
	}
	return config
}

func LoadInstanceManager(config *app.Config) app.InstanceManager {
	var im app.InstanceManager
	switch config.InstanceManager.Type {
	case app.GCEIMType:
		service, err := compute.NewService(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		nameGenerator := &gcp.InstanceNameGenerator{
			UUIDFactory: func() string { return uuid.New().String() },
		}
		im = gcp.NewInstanceManager(config.InstanceManager, service, nameGenerator)
	case app.UnixIMType:
		im = unix.NewInstanceManager(config.InstanceManager)
	default:
		log.Fatal("Unknown Instance Manager type: ", config.InstanceManager.Type)
	}
	return im
}

func LoadSecretManager(config *app.Config) app.SecretManager {
	var sm app.SecretManager
	switch config.SecretManager.Type {
	case app.GCPSMType:
		var err error
		sm, err = gcp.NewSecretManager(&config.SecretManager.GCP)
		if err != nil {
			log.Fatal("Failed to build Secret Manager: ", err)
		}
	case app.UnixSMType:
		var err error
		sm, err = unix.NewSecretManager(config.SecretManager.UNIX.SecretFilePath)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Unknown Secret Manager type: ", config.SecretManager.Type)
	}
	return sm
}

func LoadOAuthConfig(config *app.Config, sm app.SecretManager) *oauth2.Config {
	var oauthConfig *oauth2.Config
	switch config.AccountManager.OAuth.Provider {
	case app.GoogleOAuthProvider:
		oauthConfig = net.NewGoogleOAuthConfig(config.AccountManager.OAuth.RedirectURL, sm)
	default:
		log.Fatal("Unknown oauth provider: ", app.GoogleOAuthProvider)
	}
	return oauthConfig
}

func LoadAccountManager(config *app.Config, oauthConfig *oauth2.Config) app.AccountManager {
	var am app.AccountManager
	switch config.AccountManager.Type {
	case app.GAEAMType:
		am = gcp.NewUsersAccountManager(oauthConfig)
	case app.UnixAMType:
		am = &unix.AccountManager{
			OAuthConfig: oauthConfig,
		}
	default:
		log.Fatal("Unknown Account Manager type: ", config.AccountManager.Type)
	}
	return am
}

// The network interface for the web server to listen on.
func ChooseNetworkInterface(config *app.Config) string {
	if config.AccountManager.Type == app.UnixAMType {
		// This account manager is insecure, it's only meant for development. It's generally not
		// safe to listen on every interface when it's in use, so restrict it to the loopback
		// interface only.
		return "localhost"
	}
	// Empty means all interfaces, which is the right choice in production.
	return ""
}

func ServerPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	return port
}

func main() {
	config := LoadConfiguration()

	instanceManager := LoadInstanceManager(config)
	signalingServer := app.NewForwardingSignalingServer(config.WebStaticFilesPath, instanceManager)
	secretManager := LoadSecretManager(config)
	oauthConfig := LoadOAuthConfig(config, secretManager)
	accountManager := LoadAccountManager(config, oauthConfig)
	controller := app.NewController(config.Infra.STUNServers, config.Operations, instanceManager, signalingServer, accountManager)

	iface := ChooseNetworkInterface(config)
	port := ServerPort()

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(iface+":"+port, controller.Handler()))
}
