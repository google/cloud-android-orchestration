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

package net

import (
	"fmt"
	"net/http"

	"github.com/google/cloud-android-orchestration/pkg/app"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Build a oauth2.Config object with Google as the provider.
func NewGoogleOAuthConfig(redirectURL string, sm app.SecretManager) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     sm.OAuthClientID(),
		ClientSecret: sm.OAuthClientSecret(),
		Scopes:       []string{"https://www.googleapis.com/auth/androidbuild.internal"},
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
	}
}

// Extracts the authorization code and state from the authorization provider's response.
func ParseAuthorizationResponse(r *http.Request) (*AuthorizationResponse, error) {
	q := r.URL.Query()
	res := &AuthorizationResponse{}
	if errMsg, ok := q["error"]; ok {
		return nil, fmt.Errorf("Authentication error: %v", errMsg)
	}
	if stateSlice, ok := q["state"]; ok {
		res.State = stateSlice[0]
	}
	var ok bool
	var code []string
	if code, ok = q["code"]; !ok {
		return nil, fmt.Errorf("Authorization response does not include an authorization code")
	}
	res.AuthorizationCode = code[0]
	return res, nil
}

type AuthorizationResponse struct {
	AuthorizationCode string
	State             string
}
