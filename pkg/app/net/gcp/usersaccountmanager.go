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
	"fmt"
	"net/http"
	"strings"

	"github.com/google/cloud-android-orchestration/pkg/app"
	"github.com/google/cloud-android-orchestration/pkg/app/net"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

const emailHeaderKey = "X-Appengine-User-Email"

type UsersAccountManager struct {
	oauthConfig *oauth2.Config
}

func NewUsersAccountManager(oauthConfig *oauth2.Config) *UsersAccountManager {
	return &UsersAccountManager{
		oauthConfig: oauthConfig,
	}
}

func (g *UsersAccountManager) Authenticate(fn app.AuthHTTPHandler) app.HTTPHandler {
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

func (g *UsersAccountManager) RegisterAuthHandlers(r *mux.Router) {
	r.Handle("/auth", g.AuthHandler())
	r.Handle("/oauth2callback", g.OAuth2Callback())
}

func (g *UsersAccountManager) AuthHandler() app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		state := g.generateState()
		authURL := g.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusSeeOther)
		return nil
	}
}

func (g *UsersAccountManager) OAuth2Callback() app.HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		res, err := net.ParseAuthorizationResponse(r)
		if err != nil {
			return err
		}
		if !g.validateState(res.State) {
			return fmt.Errorf("Invalid state: %s", res.State)
		}
		_, err = g.oauthConfig.Exchange(oauth2.NoContext, res.AuthorizationCode)
		if err != nil {
			return err
		}
		_, err = userInfoFromRequest(r)
		if err != nil {
			return err
		}
		// Don't return a real page here since any resource (i.e JS module) will have access to the server response
		fmt.Fprintf(w, "Authorization successful, you may close this window now")
		return nil
	}
}

func (g *UsersAccountManager) generateState() string {
	// TODO(jemoreira): Actually generate a secure state value to prevent CSRF and DDoS attacks.
	// This isn't as easy as it sounds and for the time being this Account Manager relies on a proxy
	// that authenticates users for it, so requests that reach these endpoints are already
	// authenticated so it's relatively safe to trust these users... for now.
	return "somestate"
}

func (g *UsersAccountManager) validateState(state string) bool {
	return true
}

type UserInfo struct {
	username string
}

func (u *UserInfo) Username() string {
	return u.username
}

func userInfoFromRequest(r *http.Request) (*UserInfo, error) {
	// These headers are guaranteed to be present and come from AppEngine.
	email := r.Header.Get(emailHeaderKey)
	if email == "" {
		return nil, errors.New("No authentication headers present")
	}
	username := strings.SplitN(email, "@", 2)[0]
	return &UserInfo{username}, nil
}
