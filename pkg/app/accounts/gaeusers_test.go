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
	"testing"
)

func TestGAEUsersAccountManagerUserFromRequest(t *testing.T) {
	am := NewGAEUsersAccountManager()

	reqA := httptest.NewRequest(http.MethodGet, "/v1/zones", nil)
	reqA.Header.Set(emailHeaderKey, "Alice@Example.com")
	userA, err := am.UserFromRequest(reqA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := userA.Username(), "alice@example.com"; got != want {
		t.Errorf("Username() = %q, want %q", got, want)
	}
	if got, want := userA.Email(), "Alice@Example.com"; got != want {
		t.Errorf("Email() = %q, want %q", got, want)
	}

	reqB := httptest.NewRequest(http.MethodGet, "/v1/zones", nil)
	reqB.Header.Set(emailHeaderKey, "alice@other.example.com")
	userB, err := am.UserFromRequest(reqB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userA.Username() == userB.Username() {
		t.Errorf("expected distinct usernames for different domains, both got %q", userA.Username())
	}
}

func TestGAEUsersAccountManagerEmptyEmail(t *testing.T) {
	am := NewGAEUsersAccountManager()
	for _, hdr := range []string{"", "   "} {
		req := httptest.NewRequest(http.MethodGet, "/v1/zones", nil)
		if hdr != "" {
			req.Header.Set(emailHeaderKey, hdr)
		}
		if _, err := am.UserFromRequest(req); err == nil {
			t.Errorf("expected error for header %q, got nil", hdr)
		}
	}
}
