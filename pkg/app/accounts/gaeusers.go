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
	"errors"
	"net/http"
	"strings"
)

const (
	GAEAMType      AMType = "GCP"
	emailHeaderKey string = "X-Appengine-User-Email"
)

type GAEUsersAccountManager struct{}

func NewGAEUsersAccountManager() *GAEUsersAccountManager {
	return &GAEUsersAccountManager{}
}

func (g *GAEUsersAccountManager) UserFromRequest(r *http.Request) (User, error) {
	email, err := emailFromRequest(r)
	if err != nil {
		return nil, err
	}
	username, err := usernameFromEmail(email)
	if err != nil {
		return nil, err
	}
	return &GAEUser{username: username, email: email}, nil
}

func emailFromRequest(r *http.Request) (string, error) {
	// These headers are guaranteed to be present and come from AppEngine.
	return r.Header.Get(emailHeaderKey), nil
}

func usernameFromEmail(email string) (string, error) {
	if email == "" {
		return "", errors.New("empty email")
	}
	return strings.SplitN(email, "@", 2)[0], nil
}

type GAEUser struct {
	username string
	email    string
}

func (u *GAEUser) Username() string { return u.username }

func (u *GAEUser) Email() string { return u.email }
