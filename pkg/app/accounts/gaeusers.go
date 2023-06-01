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
	"fmt"
	"net/http"
	"strings"

	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
)

const (
	GAEAMType AMType = "GCP"

	emailHeaderKey = "X-Appengine-User-Email"
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
	return userFromEmail(email), nil
}

func (g *GAEUsersAccountManager) OnOAuth2Exchange(w http.ResponseWriter, r *http.Request, idToken appOAuth2.IDTokenClaims) (User, error) {
	rEmail, err := emailFromRequest(r)
	if err != nil {
		return nil, err
	}
	email, ok := idToken["email"]
	if !ok {
		return nil, fmt.Errorf("No email in id token")
	}
	tkEmail, ok := email.(string)
	if !ok {
		return nil, fmt.Errorf("Malformed email in id token")
	}
	if rEmail != tkEmail {
		return nil, fmt.Errorf("Logged in user doesn't match oauth2 user")
	}
	return userFromEmail(rEmail), nil
}

type GAEUser struct {
	username string
}

func (u *GAEUser) Username() string {
	return u.username
}

func emailFromRequest(r *http.Request) (string, error) {
	// These headers are guaranteed to be present and come from AppEngine.
	return r.Header.Get(emailHeaderKey), nil
}

func userFromEmail(email string) *GAEUser {
	if email == "" {
		return nil
	}
	username := strings.SplitN(email, "@", 2)[0]
	return &GAEUser{username}
}
