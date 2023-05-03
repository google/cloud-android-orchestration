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
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"github.com/google/cloud-android-orchestration/pkg/app"
	"github.com/google/cloud-android-orchestration/pkg/app/net"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

// Implements the AccountManager interface taking the username from the
// environment and authorizing all requests
type AccountManager struct {
	OAuthConfig *oauth2.Config
	tokenSource oauth2.TokenSource
	lastState   string
}

func (m *AccountManager) Authenticate(fn app.AuthHTTPHandler) app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return fn(w, r, &UserInfo{})
	}
}

func (m *AccountManager) RegisterAuthHandlers(r *mux.Router) {
	r.Handle("/auth", m.AuthHandler())
	r.Handle("/oauth2callback", m.OAuth2Callback())
}

func (m *AccountManager) AuthHandler() app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		m.lastState = randomHexString()
		authURL := m.OAuthConfig.AuthCodeURL(m.lastState, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusSeeOther)
		return nil
	}
}

func (m *AccountManager) OAuth2Callback() app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		res, err := net.ParseAuthorizationResponse(r)
		if err != nil {
			return err
		}
		if res.State != m.lastState {
			return fmt.Errorf("Invalid state: %s", res.State)
		}
		token, err := m.OAuthConfig.Exchange(oauth2.NoContext, res.AuthorizationCode)
		if err != nil {
			return err
		}
		m.tokenSource = m.OAuthConfig.TokenSource(oauth2.NoContext, token)
		// Don't return a real page here since any resource (i.e JS module) will have access to the server response
		fmt.Fprintf(w, "Authorization successful, you may close this window now")
		return nil
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

func randomHexString() string {
	// This produces a 32 char random string from the [0-9a-f] alphabet.
	return fmt.Sprintf("%.16x%.16x", rand.Uint64(), rand.Uint64())
}
