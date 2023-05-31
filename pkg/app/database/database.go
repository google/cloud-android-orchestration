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

package database

import (
	"github.com/google/cloud-android-orchestration/pkg/app/session"
)

type Service interface {
	// Credentials are usually stored encrypted hence the []byte type.
	// If no credentials are available for the given user Fetch returns nil, nil.
	FetchBuildAPICredentials(username string) ([]byte, error)
	// Store new credentials or overwrite existing ones for the given user.
	StoreBuildAPICredentials(username string, credentials []byte) error
	DeleteBuildAPICredentials(username string) error
	// Create or update a user session.
	CreateOrUpdateSession(s session.Session) error
	// Fetch a session. Returns nil, nil if the session doesn't exist.
	FetchSession(key string) (*session.Session, error)
	// Delete a session. Won't return error if the session doesn't exist.
	DeleteSession(key string) error
}

type Config struct {
	Type    string
	Spanner *SpannerConfig
}
