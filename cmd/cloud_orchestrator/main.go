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
	"crypto/rand"
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
		sm, err = gcp.NewSecretManager(config.SecretManager.GCP)
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

func LoadAccountManager(config *app.Config) app.AccountManager {
	var am app.AccountManager
	switch config.AccountManager.Type {
	case app.GAEAMType:
		am = gcp.NewUsersAccountManager()
	case app.UnixAMType:
		am = &unix.AccountManager{}
	default:
		log.Fatal("Unknown Account Manager type: ", config.AccountManager.Type)
	}
	return am
}

func LoadEncryptionService(config *app.Config) app.EncryptionService {
	var es app.EncryptionService
	switch config.EncryptionService.Type {
	case app.SimpleESType:
		key := make([]byte, config.EncryptionService.Simple.KeySizeBits/8)
		_, err := rand.Read(key)
		if err != nil {
			log.Fatal("Failed to generate crypto key: ", err)
		}
		es, err = unix.NewSimpleEncryptionService(key)
		if err != nil {
			log.Fatal("Failed to create simple encryption service: ", err)
		}
	case app.GCPKMSESType:
		es = gcp.NewKMSEncryptionService(config.EncryptionService.GCPKMS.KeyName)
	default:
		log.Fatal("Unknown encryption service type: ", config.EncryptionService.Type)
	}
	return es
}

func LoadDatabaseService(config *app.Config) app.DatabaseService {
	var dbs app.DatabaseService
	switch config.DatabaseService.Type {
	case app.InMemoryDBType:
		dbs = unix.NewInMemoryDBService()
	case app.SpannerDBType:
		dbs = gcp.NewSpannerDBService(config.DatabaseService.Spanner.DatabaseName)
	default:
		log.Fatal("Unknown database service type: ", config.DatabaseService.Type)
	}
	return dbs
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
	accountManager := LoadAccountManager(config)
	encryptionService := LoadEncryptionService(config)
	dbService := LoadDatabaseService(config)
	controller := app.NewController(config.Infra.STUNServers, config.Operations, instanceManager,
		signalingServer, accountManager, oauthConfig, encryptionService, dbService)

	iface := ChooseNetworkInterface(config)
	port := ServerPort()

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(iface+":"+port, controller.Handler()))
}
