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
	"github.com/google/cloud-android-orchestration/pkg/app/gcp"
	"github.com/google/cloud-android-orchestration/pkg/app/unix"

	"github.com/google/uuid"
	"google.golang.org/api/compute/v1"
)

func main() {
	config, err := app.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}
	var im app.InstanceManager
	switch config.InstanceManager.Type {
	case app.GCEIMType:
		service, err := compute.NewService(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		im = &gcp.InstanceManager{
			Config:  config.InstanceManager,
			Service: service,
			InstanceNameGenerator: &gcp.InstanceNameGenerator{
				UUIDFactory: func() string { return uuid.New().String() },
			},
		}
	case app.UnixIMType:
		im = &unix.InstanceManager{}
	default:
		log.Fatal("Unknown Instance Manager type: ", config.InstanceManager.Type)
	}

	ss := app.NewForwardingSignalingServer(config.WebStaticFilesPath, im)

	var am app.AccountManager
	switch config.AccountManager.Type {
	case app.GAEAMType:
		am = gcp.NewUsersAccountManager()
	case app.UnixAMType:
		am = &unix.AccountManager{}
	default:
		log.Fatal("Unknown Account Manager type: ", config.AccountManager.Type)
	}

	or := app.NewController(config.Infra.STUNServers, config.Operations, im, ss, am)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, or.Handler()))
}
