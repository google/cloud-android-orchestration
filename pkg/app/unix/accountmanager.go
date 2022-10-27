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

package unix

import (
	"net/http"
	"os"

	"github.com/google/cloud-android-orchestration/pkg/app"
)

// Implements the AccountManager interface taking the username from the
// environment and authorizing all requests
type AccountManager struct{}

func (m *AccountManager) Authenticate(fn app.AuthHTTPHandler) app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return fn(w, r, &UserInfo{})
	}
}

type UserInfo struct {
	username string
}

func (i *UserInfo) Username() string {
	if i.username == "" {
		i.username = os.Getenv("USER")
	}
	return i.username
}
