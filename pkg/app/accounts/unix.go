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

package accounts

import (
	"net/http"
	"os"

	appOAuth "github.com/google/cloud-android-orchestration/pkg/app/oauth2"

	"golang.org/x/oauth2"
)

const UnixAMType AMType = "unix"

// Implements the Manager interface taking the username from the
// environment and authorizing all requests
type UnixAccountManager struct {
	tokenSource oauth2.TokenSource
	lastState   string
}

func NewUnixAccountManager() *UnixAccountManager {
	return &UnixAccountManager{}
}

func (m *UnixAccountManager) Authenticate(fn AuthHTTPHandler) HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return fn(w, r, &UnixUserInfo{})
	}
}

func (m *UnixAccountManager) OnOAuthExchange(w http.ResponseWriter, r *http.Request, tk appOAuth.IDTokenClaims) (UserInfo, error) {
	return &UnixUserInfo{}, nil
}

type UnixUserInfo struct {
	username string
}

func (i *UnixUserInfo) Username() string {
	if i.username == "" {
		i.username = os.Getenv("USER")
	}
	return i.username
}
