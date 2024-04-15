// Copyright 2024 Google LLC
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
)

const HTTPBasicAMType AMType = "http-basic"

// Implements the AccountManager interfaces using the RFC 2617 HTTP Basic
// Authentication, where the user-ID and password are provided in the HTTP
// request header.
type HTTPBasicAccountManager struct{}

func NewHTTPBasicAccountManager() *HTTPBasicAccountManager {
	return &HTTPBasicAccountManager{}
}

func (m *HTTPBasicAccountManager) UserFromRequest(r *http.Request) (User, error) {
	username, _, ok := r.BasicAuth()
	if !ok {
		return nil, nil
	}
	return &HTTPBasicUser{username}, nil
}

type HTTPBasicUser struct {
	username string
}

func (u *HTTPBasicUser) Username() string { return u.username }

func (u *HTTPBasicUser) Email() string { return "" }
