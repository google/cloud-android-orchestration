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

	apperr "github.com/google/cloud-android-orchestration/pkg/app/errors"
	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
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
	return userFromRequest(r)
}

func (m *HTTPBasicAccountManager) OnOAuth2Exchange(w http.ResponseWriter, r *http.Request, tk appOAuth2.IDTokenClaims) (User, error) {
	rUser, err := userFromRequest(r)
	if err != nil {
		return nil, err
	}
	user, ok := tk["user"]
	if !ok {
		return nil, apperr.NewForbiddenError("no user in id token", nil)
	}
	tkUser, ok := user.(string)
	if !ok {
		return nil, apperr.NewForbiddenError("malformed user in id token", nil)
	}
	if rUser.Username() != tkUser {
		return nil, apperr.NewForbiddenError("logged in user doesn't match oauth2 user", nil)
	}
	return rUser, nil
}

type HTTPBasicUser struct {
	username string
}

func (u *HTTPBasicUser) Username() string {
	return u.username
}

func userFromRequest(r *http.Request) (*HTTPBasicUser, error) {
	username, _, ok := r.BasicAuth()
	if !ok {
		return nil, apperr.NewBadRequestError("No username in request", nil)
	}
	return &HTTPBasicUser{username}, nil
}
