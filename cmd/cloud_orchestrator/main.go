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
	"github.com/google/cloud-android-orchestration/pkg/app/types"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
)

func LoadConfiguration() *types.Config {
	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	log.Println("Main Configuration:")
	log.Println("  Instance Manager Type: " + config.InstanceManager.Type)
	if config.InstanceManager.Type == types.GCEIMType {
		log.Println("  GCP Project: " + config.InstanceManager.GCP.ProjectID)
	}
	return config
}

func LoadInstanceManager(config *types.Config) types.InstanceManager {
	var im types.InstanceManager
	switch config.InstanceManager.Type {
	case types.GCEIMType:
		service, err := compute.NewService(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		nameGenerator := &gcp.InstanceNameGenerator{
			UUIDFactory: func() string { return uuid.New().String() },
		}
		im = gcp.NewInstanceManager(config.InstanceManager, service, nameGenerator)
	case types.UnixIMType:
		im = unix.NewInstanceManager(config.InstanceManager)
	default:
		log.Fatal("Unknown Instance Manager type: ", config.InstanceManager.Type)
	}
	return im
}

func LoadSecretManager(config *types.Config) types.SecretManager {
	var sm types.SecretManager
	switch config.SecretManager.Type {
	case types.GCPSMType:
		var err error
		sm, err = gcp.NewSecretManager(config.SecretManager.GCP)
		if err != nil {
			log.Fatal("Failed to build Secret Manager: ", err)
		}
	case types.UnixSMType:
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

func LoadOAuthConfig(config *types.Config, sm types.SecretManager) *oauth2.Config {
	var oauthConfig *oauth2.Config
	switch config.AccountManager.OAuth.Provider {
	case types.GoogleOAuthProvider:
		oauthConfig = net.NewGoogleOAuthConfig(config.AccountManager.OAuth.RedirectURL, sm)
	default:
		log.Fatal("Unknown oauth provider: ", config.AccountManager.OAuth.Provider)
	}
	return oauthConfig
}

func LoadAccountManager(config *types.Config) types.AccountManager {
	var am types.AccountManager
	switch config.AccountManager.Type {
	case types.GAEAMType:
		am = gcp.NewUsersAccountManager()
	case types.UnixAMType:
		am = &unix.AccountManager{}
	default:
		log.Fatal("Unknown Account Manager type: ", config.AccountManager.Type)
	}
	return am
}

func LoadEncryptionService(config *types.Config) types.EncryptionService {
	var es types.EncryptionService
	switch config.EncryptionService.Type {
	case types.SimpleESType:
		key := make([]byte, config.EncryptionService.Simple.KeySizeBits/8)
		_, err := rand.Read(key)
		if err != nil {
			log.Fatal("Failed to generate crypto key: ", err)
		}
		es = unix.NewSimpleEncryptionService()
	case types.GCPKMSESType:
		es = gcp.NewKMSEncryptionService(config.EncryptionService.GCPKMS.KeyName)
	default:
		log.Fatal("Unknown encryption service type: ", config.EncryptionService.Type)
	}
	return es
}

func LoadDatabaseService(config *types.Config) types.DatabaseService {
	var dbs types.DatabaseService
	switch config.DatabaseService.Type {
	case types.InMemoryDBType:
		dbs = unix.NewInMemoryDBService()
	case types.SpannerDBType:
		dbs = gcp.NewSpannerDBService(config.DatabaseService.Spanner.DatabaseName)
	default:
		log.Fatal("Unknown database service type: ", config.DatabaseService.Type)
	}
	return dbs
}

// The network interface for the web server to listen on.
func ChooseNetworkInterface(config *types.Config) string {
	if config.AccountManager.Type == types.UnixAMType {
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
