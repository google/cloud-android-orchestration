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
	"net/http"

	appOAuth "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
)

type AuthHTTPHandler func(http.ResponseWriter, *http.Request, UserInfo) error
type HTTPHandler func(http.ResponseWriter, *http.Request) error

type UserInfo interface {
	Username() string
}

type Manager interface {
	// Returns the received http handler wrapped in another that extracts user
	// information from the request and passes it to to the original handler as
	// the last parameter.
	// The wrapper will only pass the request to the inner handler if a user is
	// authenticated, otherwise it may choose to return an error or respond with
	// an HTTP redirect to the login page.
	Authenticate(fn AuthHTTPHandler) HTTPHandler
	// Gives the account manager the chance to extract login information from the token (id token
	// for example), validate it, add cookies to the request, etc.
	OnOAuthExchange(w http.ResponseWriter, r *http.Request, idToken appOAuth.IDTokenClaims) (UserInfo, error)
}

type AMType string

type Config struct {
	Type  AMType
	OAuth appOAuth.OAuthConfig
}
