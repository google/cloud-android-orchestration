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
)

const UsernameOnlyAMType AMType = "username-only"

// Implements the AccountManager interfaces for closed deployed cloud
// orchestrators. This AccountManager leverages the RFC 2617 HTTP Basic
// Authentication to get the account information, but only username is used.
type UsernameOnlyAccountManager struct{}

func NewUsernameOnlyAccountManager() *UsernameOnlyAccountManager {
	return &UsernameOnlyAccountManager{}
}

func (m *UsernameOnlyAccountManager) UserFromRequest(r *http.Request) (User, error) {
	return userFromRequest(r)
}

type UsernameOnlyUser struct {
	username string
}

func (u *UsernameOnlyUser) Username() string { return u.username }

func (u *UsernameOnlyUser) Email() string { return "" }

func userFromRequest(r *http.Request) (*UsernameOnlyUser, error) {
	username, _, ok := r.BasicAuth()
	if !ok {
		return nil, apperr.NewBadRequestError("No username in request", nil)
	}
	return &UsernameOnlyUser{username}, nil
}
