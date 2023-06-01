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

package oauth2

import (
	"github.com/google/cloud-android-orchestration/pkg/app/secrets"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuth2Config struct {
	Provider    string
	RedirectURL string
}

const (
	GoogleOAuth2Provider = "Google"
)

// Build a oauth2.Config object with Google as the provider.
func NewGoogleOAuth2Config(redirectURL string, sm secrets.SecretManager) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     sm.OAuth2ClientID(),
		ClientSecret: sm.OAuth2ClientSecret(),
		Scopes: []string{
			"https://www.googleapis.com/auth/androidbuild.internal",
			"openid",
			"email",
		},
		RedirectURL: redirectURL,
		Endpoint:    google.Endpoint,
	}
}

// ID tokens (from OpenID connect) are presented in JWT format, with the relevant fields in the Claims section.
type IDTokenClaims map[string]interface{}
