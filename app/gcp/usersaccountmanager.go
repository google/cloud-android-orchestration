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

package gcp

import (
	"errors"
	"net/http"
	"strings"

	"cloud-android-orchestration/app"
)

const emailHeaderKey = "X-Appengine-User-Email"

type UsersAccountManager struct{}

func NewUsersAccountManager() *UsersAccountManager {
	return &UsersAccountManager{}
}

func (g *UsersAccountManager) Authenticate(fn app.AuthHTTPHandler) app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// These headers are guaranteed to be present and come from AppEngine.
		email := r.Header.Get(emailHeaderKey)
		if email == "" {
			// Normally a redirect to the login page is returned here, but the App Engine
			// api takes care of that so if a request gets this far without the headers
			// it can only be due to server misconfiguration.
			// A general error returned here will trigger a 500 response
			return errors.New("No authentication headers present")
		}
		username := strings.SplitN(email, "@", 2)[0]
		user := UserInfo{username}
		return fn(w, r, &user)
	}
}

type UserInfo struct {
	username string
}

func (u *UserInfo) Username() string {
	return u.username
}
