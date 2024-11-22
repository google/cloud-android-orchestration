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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"
)

const (
	IAPType         AMType = "IAP"
	iapJWTHeaderKey        = "x-goog-iap-jwt-assertion"
)

type IAPAccountManager struct{}

func NewIAPAccountManager() *IAPAccountManager {
	return &IAPAccountManager{}
}

func (g *IAPAccountManager) UserFromRequest(r *http.Request) (User, error) {
	token := r.Header.Get(iapJWTHeaderKey)
	if token == "" {
		return nil, fmt.Errorf("%s header is empty", iapJWTHeaderKey)
	}
	audience, ok := os.LookupEnv("IAP_AUDIENCE")
	if !ok {
		return nil, errors.New("IAP_AUDIENCE env var not set")
	}
	// TODO: if we want to validate the audience then we need to pass it in from the config, otherwise set audience to ""
	payload, err := idtoken.Validate(context.TODO(), token, audience)
	if err != nil {
		return nil, fmt.Errorf("unable to parse payload from %s header: %w", iapJWTHeaderKey, err)
	}
	email := payload.Claims["email"].(string)
	if email == "" {
		return nil, errors.New("empty email claim")
	}
	username := strings.SplitN(email, "@", 2)[0]
	return &IAPUser{username: username, email: email}, nil
}

type IAPUser struct {
	username string
	email    string
}

func (u *IAPUser) Username() string { return u.username }

func (u *IAPUser) Email() string { return u.email }
