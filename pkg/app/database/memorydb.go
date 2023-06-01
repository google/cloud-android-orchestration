// Copyright 2023 Google LLC
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

const InMemoryDBType = "InMemory"

type SpannerConfig struct {
	DatabaseName string
}

// Simple in memory database to use for testing or local development.
type InMemoryDBService struct {
	credentials map[string][]byte
	session     session.Session
}

func NewInMemoryDBService() *InMemoryDBService {
	return &InMemoryDBService{
		credentials: make(map[string][]byte),
	}
}

func (dbs *InMemoryDBService) FetchBuildAPICredentials(username string) ([]byte, error) {
	return dbs.credentials[username], nil
}

func (dbs *InMemoryDBService) StoreBuildAPICredentials(username string, credentials []byte) error {
	dbs.credentials[username] = credentials
	return nil
}

func (dbs *InMemoryDBService) DeleteBuildAPICredentials(username string) error {
	delete(dbs.credentials, username)
	return nil
}

func (dbs *InMemoryDBService) CreateOrUpdateSession(s session.Session) error {
	dbs.session = s
	return nil
}

func (dbs *InMemoryDBService) FetchSession(key string) (*session.Session, error) {
	if dbs.session.Key != key {
		return nil, nil
	}
	sessionCopy := dbs.session
	return &sessionCopy, nil
}

func (dbs *InMemoryDBService) DeleteSession(key string) error {
	if dbs.session.Key == key {
		dbs.session = session.Session{}
	}
	return nil
}
