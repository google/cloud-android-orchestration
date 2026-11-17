// Copyright 2026 Google LLC
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
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleUsernameOnlyLogging(t *testing.T) {
	tests := []struct {
		name         string
		redirect     string
		wantLocation string
	}{
		{name: "path", redirect: "/v1/zones?foo=bar", wantLocation: "/v1/zones?foo=bar"},
		{name: "empty", redirect: "", wantLocation: ""},
		{name: "absolute url", redirect: "https://example.com/v1/zones", wantLocation: ""},
		{name: "scheme relative url", redirect: "//example.com/v1/zones", wantLocation: ""},
		{name: "backslash after slash", redirect: "/\\example.com/v1/zones", wantLocation: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/username", strings.NewReader(url.Values{"username": {"johndoe"}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			if err := HandleUsernameOnlyLogging(w, r, tc.redirect); err != nil {
				t.Fatal(err)
			}

			if got := w.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("expected location %q, got %q", tc.wantLocation, got)
			}
		})
	}
}
