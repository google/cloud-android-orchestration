// Copyright 2022 Google LLC
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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	"github.com/hashicorp/go-multierror"
)

func TestDeleteHosts(t *testing.T) {
	existingNames := map[string]struct{}{"bar": {}, "baz": {}}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			panic("unexpected method: " + r.Method)
		}
		re := regexp.MustCompile(`^/hosts/(.*)$`)
		matches := re.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			panic("unexpected path: " + r.URL.Path)
		}
		if _, ok := existingNames[matches[1]]; ok {
			writeOK(w, "")
		} else {
			writeErr(w, 404)
		}
	}))
	defer ts.Close()
	opts := &ClientOptions{
		RootEndpoint: ts.URL,
		DumpOut:      io.Discard,
	}
	c, _ := NewClient(opts)

	err := c.DeleteHosts([]string{"foo", "bar", "baz", "quz"})

	merr, _ := err.(*multierror.Error)
	errs := merr.WrappedErrors()
	if len(errs) != 2 {
		t.Errorf("expected 2, got: %d", len(errs))
	}
}

func writeErr(w http.ResponseWriter, statusCode int) {
	write(w, &apiv1.Error{Code: statusCode}, statusCode)
}

func writeOK(w http.ResponseWriter, data any) {
	write(w, data, http.StatusOK)
}

func write(w http.ResponseWriter, data any, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(data)
}

func TestCreateHostSuccess(t *testing.T) {
	hostName := "foo"
	waitHostCalled := false
	legacyCheckCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/hosts":
			writeOK(w, &apiv1.Operation{Name: "op-1", Done: false})
		case r.Method == "POST" && r.URL.Path == "/operations/op-1/:wait":
			writeOK(w, &apiv1.HostInstance{Name: hostName})
		case r.Method == "POST" && r.URL.Path == fmt.Sprintf("/hosts/%s/:wait-host-availability", hostName):
			waitHostCalled = true
			writeOK(w, &apiv1.HostInstance{Name: hostName})
		case r.Method == "GET" && r.URL.Path == fmt.Sprintf("/hosts/%s/", hostName):
			legacyCheckCalled = true
			writeOK(w, "")
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			writeErr(w, 404)
		}
	}))
	defer ts.Close()

	var stderr bytes.Buffer
	opts := &ClientOptions{
		RootEndpoint: ts.URL,
		DumpOut:      io.Discard,
		ErrOut:       &stderr,
	}
	c, _ := NewClient(opts)

	ins, err := c.CreateHost(&apiv1.CreateHostRequest{HostInstance: &apiv1.HostInstance{}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ins.Name != hostName {
		t.Errorf("expected host name %q, got %q", hostName, ins.Name)
	}
	if !waitHostCalled {
		t.Error("expected WaitHostAvailability to be called")
	}
	if legacyCheckCalled {
		t.Error("expected legacy check NOT to be called on success")
	}
	if stderr.Len() > 0 {
		t.Errorf("unexpected stderr output: %q", stderr.String())
	}
}

func TestCreateHostFallback(t *testing.T) {
	hostName := "foo"
	waitHostCalled := false
	legacyCheckCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/hosts":
			writeOK(w, &apiv1.Operation{Name: "op-1", Done: false})
		case r.Method == "POST" && r.URL.Path == "/operations/op-1/:wait":
			writeOK(w, &apiv1.HostInstance{Name: hostName})
		case r.Method == "POST" && r.URL.Path == fmt.Sprintf("/hosts/%s/:wait-host-availability", hostName):
			waitHostCalled = true
			writeErr(w, 500) // Fail new wait
		case r.Method == "GET" && r.URL.Path == fmt.Sprintf("/hosts/%s/", hostName):
			legacyCheckCalled = true
			writeOK(w, "") // Succeed legacy wait
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			writeErr(w, 404)
		}
	}))
	defer ts.Close()

	var stderr bytes.Buffer
	opts := &ClientOptions{
		RootEndpoint: ts.URL,
		DumpOut:      io.Discard,
		ErrOut:       &stderr,
	}
	c, _ := NewClient(opts)

	ins, err := c.CreateHost(&apiv1.CreateHostRequest{HostInstance: &apiv1.HostInstance{}})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ins.Name != hostName {
		t.Errorf("expected host name %q, got %q", hostName, ins.Name)
	}
	if !waitHostCalled {
		t.Error("expected WaitHostAvailability to be called")
	}
	if !legacyCheckCalled {
		t.Error("expected legacy check to be called on fallback")
	}
	if !strings.Contains(stderr.String(), "Warning: Host availability check failed") {
		t.Errorf("expected warning in stderr, got: %q", stderr.String())
	}
}
