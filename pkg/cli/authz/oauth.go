// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func createServer(ch chan string, state string, port string) *http.Server {
	return &http.Server{
		Addr:    ":" + port,
		Handler: http.HandlerFunc(handler(ch, state)),
	}
}

func handler(ch chan string, randState string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "error: visiting /favicon.ico", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("state: %s doesn't match. (expected: %s)", req.FormValue("state"), randState)
			http.Error(rw, "invalid state", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprintf(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		ch <- ""
		http.Error(rw, "invalid code", 500)
	}
}

func OAuthAccessToken(credential []byte) (*oauth2.Token, error) {
	oauthConfig, err := google.ConfigFromJSON(credential, "https://www.googleapis.com/auth/androidbuild.internal")
	if err != nil {
		return nil, err
	}
	redirectURL, err := url.Parse(oauthConfig.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("parse redirect url error: %w", err)
	}
	if redirectURL.Hostname() != "localhost" {
		return nil, errors.New("the redirect url should be `http://localhost:<whatever_port>`")
	}
	port := redirectURL.Port()
	if port == "" {
		return nil, errors.New("empty port, the redirect url should be `http://localhost:<whatever_port>`")
	}

	ch := make(chan string)
	ctx := context.Background()

	// generate a random hex string
	state := fmt.Sprintf("%.16x%.16x%.16x%.16x", rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64())
	ts := createServer(ch, state, port)
	go func() {
		err := ts.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	defer ts.Close()
	authURL := oauthConfig.AuthCodeURL(state)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch

	tk, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return tk, nil
}

func JWTAccessToken(credential []byte) (*oauth2.Token, error) {
	jwtConfig, err := google.JWTConfigFromJSON(credential, "https://www.googleapis.com/auth/androidbuild.internal")
	if err != nil {
		return nil, err
	}
	tk, err := jwtConfig.TokenSource(context.Background()).Token()
	if err != nil {
		return nil, err
	}
	return tk, nil
}
