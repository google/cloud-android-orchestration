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

const (
	URLQueryAMType AMType = "url-query"

	URLQueryKey = "user"
)

// Implements the AccountManager interfaces using a URL query parameter to hint
// the account username. For example, `?user=<username>`.
type URLQueryAccountManager struct{}

func NewURLQueryAccountManager() *URLQueryAccountManager {
	return &URLQueryAccountManager{}
}

func (m *URLQueryAccountManager) UserFromRequest(r *http.Request) (User, error) {
	username := r.URL.Query().Get(URLQueryKey)
	if username == "" {
		return nil, apperr.NewBadRequestError("no username in url query", nil)
	}
	return &URLQueryUser{username}, nil
}

type URLQueryUser struct {
	username string
}

func (u *URLQueryUser) Username() string { return u.username }

func (u *URLQueryUser) Email() string { return "" }
