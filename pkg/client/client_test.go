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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	"github.com/hashicorp/go-multierror"
)

func TestDeleteHosts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch ep := r.Method + " " + r.URL.Path; ep {
		case "DELETE /hosts/bar":
			writeOK(w, apiv1.Operation{Name: "deletingbar"})
		case "DELETE /hosts/baz":
			writeOK(w, apiv1.Operation{Name: "deletingbaz"})
		case "POST /operations/deletingbar/:wait":
			writeOK(w, apiv1.HostInstance{Name: "bar"})
		case "POST /operations/deletingbaz/:wait":
			writeOK(w, apiv1.HostInstance{Name: "baz"})
		default:
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
