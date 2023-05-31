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
	"fmt"
	"net/http"
	"strings"

	appOAuth "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
)

const (
	GAEAMType AMType = "GCP"

	emailHeaderKey = "X-Appengine-User-Email"
)

type GAEUsersAccountManager struct{}

func NewGAEUsersAccountManager() *GAEUsersAccountManager {
	return &GAEUsersAccountManager{}
}

func (g *GAEUsersAccountManager) Authenticate(fn AuthHTTPHandler) HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, err := userInfoFromRequest(r)
		if err != nil {
			// Normally a redirect to the login page is returned here, but the App Engine
			// api takes care of that so if a request gets this far without the headers
			// it can only be due to server misconfiguration.
			// A general error returned here will trigger a 500 response
			return err
		}
		return fn(w, r, user)
	}
}

func (g *GAEUsersAccountManager) OnOAuthExchange(w http.ResponseWriter, r *http.Request, idToken appOAuth.IDTokenClaims) (UserInfo, error) {
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
		return nil, fmt.Errorf("Logged in user doesn't match oauth user")
	}
	return userInfoFromEmail(rEmail), nil
}

type GAEUserInfo struct {
	username string
}

func (u *GAEUserInfo) Username() string {
	return u.username
}

func emailFromRequest(r *http.Request) (string, error) {
	// These headers are guaranteed to be present and come from AppEngine.
	email := r.Header.Get(emailHeaderKey)
	if email == "" {
		return "", errors.New("No authentication headers present")
	}
	return email, nil
}

func userInfoFromEmail(email string) *GAEUserInfo {
	username := strings.SplitN(email, "@", 2)[0]
	return &GAEUserInfo{username}
}

func userInfoFromRequest(r *http.Request) (*GAEUserInfo, error) {
	email, err := emailFromRequest(r)
	if err != nil {
		return nil, err
	}
	return userInfoFromEmail(email), nil
}
