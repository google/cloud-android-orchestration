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
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/config"
	"github.com/google/cloud-android-orchestration/pkg/app/database"
	"github.com/google/cloud-android-orchestration/pkg/app/encryption"
	"github.com/google/cloud-android-orchestration/pkg/app/instances"
	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
	"github.com/google/cloud-android-orchestration/pkg/app/secrets"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"google.golang.org/api/compute/v1"
)

func LoadConfiguration() *config.Config {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	log.Println("Main Configuration:")
	log.Println("  Instance Manager Type: " + config.InstanceManager.Type)
	log.Println("  Account Manager Type: " + config.AccountManager.Type)
	if config.InstanceManager.Type == instances.GCEIMType {
		log.Println("  GCP Project: " + config.InstanceManager.GCP.ProjectID)
	}
	return config
}

func LoadInstanceManager(config *config.Config) instances.Manager {
	var im instances.Manager
	switch config.InstanceManager.Type {
	case instances.GCEIMType:
		service, err := compute.NewService(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		nameGenerator := &instances.InstanceNameGenerator{
			UUIDFactory: func() string { return uuid.New().String() },
		}
		im = instances.NewGCEInstanceManager(config.InstanceManager, service, nameGenerator)
	case instances.UnixIMType:
		im = instances.NewLocalInstanceManager(config.InstanceManager)
	case instances.DockerIMType:
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			log.Fatal("Failed to get docker client: ", err)
		}
		im = instances.NewDockerInstanceManager(config.InstanceManager, *cli)
	default:
		log.Fatal("Unknown Instance Manager type: ", config.InstanceManager.Type)
	}
	return im
}

func LoadSecretManager(config *config.Config) secrets.SecretManager {
	var sm secrets.SecretManager
	switch config.SecretManager.Type {
	case secrets.GCPSMType:
		var err error
		sm, err = secrets.NewGCPSecretManager(config.SecretManager.GCP)
		if err != nil {
			log.Fatal("Failed to build Secret Manager: ", err)
		}
	case secrets.UnixSMType:
		var err error
		sm, err = secrets.NewFromFileSecretManager(config.SecretManager.UNIX.SecretFilePath)
		if err != nil {
			log.Fatal(err)
		}
	case secrets.EmptySMType:
		return secrets.NewEmptySecretManager()
	default:
		log.Fatal("Unknown Secret Manager type: ", config.SecretManager.Type)
	}
	return sm
}

func LoadOAuth2Config(config *config.Config, sm secrets.SecretManager) *appOAuth2.Helper {
	var oauth2Helper *appOAuth2.Helper
	switch config.AccountManager.OAuth2.Provider {
	case appOAuth2.GoogleOAuth2Provider:
		oauth2Helper = appOAuth2.NewGoogleOAuth2Helper(config.AccountManager.OAuth2.RedirectURL, sm)
	default:
		log.Fatal("Unknown oauth2 provider: ", config.AccountManager.OAuth2.Provider)
	}
	return oauth2Helper
}

func LoadAccountManager(config *config.Config) accounts.Manager {
	var am accounts.Manager
	switch config.AccountManager.Type {
	case accounts.IAPType:
		am = accounts.NewIAPAccountManager()
	case accounts.GAEAMType:
		am = accounts.NewGAEUsersAccountManager()
	case accounts.UnixAMType:
		am = accounts.NewUnixAccountManager()
	case accounts.UsernameOnlyAMType:
		am = accounts.NewUsernameOnlyAccountManager()
	default:
		log.Fatal("Unknown Account Manager type: ", config.AccountManager.Type)
	}
	return am
}

func LoadEncryptionService(config *config.Config) encryption.Service {
	var es encryption.Service
	switch config.EncryptionService.Type {
	case encryption.FakeESType:
		es = encryption.NewFakeEncryptionService()
	case encryption.GCPKMSESType:
		es = encryption.NewGCPKMSEncryptionService(config.EncryptionService.GCPKMS.KeyName)
	default:
		log.Fatal("Unknown encryption service type: ", config.EncryptionService.Type)
	}
	return es
}

func LoadDatabaseService(config *config.Config) database.Service {
	var dbs database.Service
	switch config.DatabaseService.Type {
	case database.InMemoryDBType:
		dbs = database.NewInMemoryDBService()
	case database.SpannerDBType:
		dbs = database.NewSpannerDBService(config.DatabaseService.Spanner.DatabaseName)
	default:
		log.Fatal("Unknown database service type: ", config.DatabaseService.Type)
	}
	return dbs
}

// The network interface for the web server to listen on.
func ChooseNetworkInterface(config *config.Config) string {
	if config.AccountManager.Type == accounts.UnixAMType {
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
	secretManager := LoadSecretManager(config)
	oauth2Helper := LoadOAuth2Config(config, secretManager)
	accountManager := LoadAccountManager(config)
	encryptionService := LoadEncryptionService(config)
	dbService := LoadDatabaseService(config)
	controller := app.NewApp(instanceManager, accountManager, oauth2Helper,
		encryptionService, dbService, config.WebStaticFilesPath, config.CORSAllowedOrigins, config.WebRTC, config)

	iface := ChooseNetworkInterface(config)
	port := ServerPort()

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(iface+":"+port, controller.Handler()))
}
