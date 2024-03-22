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
	"errors"
	"fmt"
	"net/http"
	"net/url"

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

type Helper struct {
	oauth2.Config
	Revoke func(*oauth2.Token) error
}

// Build a oauth2.Config object with Google as the provider.
func NewGoogleOAuth2Helper(redirectURL string, sm secrets.SecretManager) *Helper {
	return &Helper{
		Config: oauth2.Config{
			ClientID:     sm.OAuth2ClientID(),
			ClientSecret: sm.OAuth2ClientSecret(),
			Scopes: []string{
				"https://www.googleapis.com/auth/androidbuild.internal",
				"openid",
				"email",
			},
			RedirectURL: redirectURL,
			Endpoint:    google.Endpoint,
		},
		Revoke: RevokeGoogleOAuth2Token,
	}
}

func RevokeGoogleOAuth2Token(tk *oauth2.Token) error {
	if tk == nil {
		return fmt.Errorf("nil Token")
	}
	_, err := http.DefaultClient.PostForm(
		"https://oauth2.googleapis.com/revoke",
		url.Values{"token": []string{tk.AccessToken}})
	return err
}

// ID tokens (from OpenID connect) are presented in JWT format, with the relevant fields in the Claims section.
type IDTokenClaims map[string]interface{}

func (c IDTokenClaims) Email() (string, error) {
	v, ok := c["email"]
	if !ok {
		return "", errors.New("no email in id token")
	}
	vstr, ok := v.(string)
	if !ok {
		return "", errors.New("malformed email in id token")
	}
	return vstr, nil
}
