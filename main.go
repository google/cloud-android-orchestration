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
	"os"

	"cloud-android-orchestration/internal/app"
	"cloud-android-orchestration/internal/instancemanager/gcp"
)

func HostedInGAE() bool {
	return os.Getenv("GAE_APPLICATION") != ""
}

func main() {
	im, err := gcp.NewGCPInstanceManager(app.EmptyConfig(), context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer im.Close()
	ss := app.NewForwardingSignalingServer(im)
	var am app.AccountManager
	if HostedInGAE() {
		am = app.NewGAEUsersAccountManager()
	} else {
		am = &app.OsAccountManager{}
	}
	or := app.NewController(im, ss, am)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := or.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
