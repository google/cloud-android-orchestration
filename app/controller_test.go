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

package app

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
)

func TestBuildListHostsRequest(t *testing.T) {

	t.Run("default", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "http://abc.com/", nil)

		listReq, _ := BuildListHostsRequest(r)

		if listReq.MaxResults != 0 {
			t.Errorf("expected <<%d>>, got %d", 0, listReq.MaxResults)
		}
		if listReq.PageToken != "" {
			t.Errorf("expected empty string, got %q", listReq.PageToken)
		}
	})

	t.Run("non integer maxResults", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "http://abc.com/query?maxResults=foo", nil)

		listReq, err := BuildListHostsRequest(r)

		assertIsAppError(t, err)
		if listReq != nil {
			t.Errorf("expected nil, got %+v", listReq)
		}
	})

	t.Run("negative integer maxResults", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "http://abc.com/query?maxResults=-1", nil)

		listReq, err := BuildListHostsRequest(r)

		assertIsAppError(t, err)
		if listReq != nil {
			t.Errorf("expected nil, got %+v", listReq)
		}
	})

	t.Run("full", func(t *testing.T) {
		r, _ := http.NewRequest("GET", "http://abc.com/query?pageToken=foo&maxResults=1", nil)

		listReq, _ := BuildListHostsRequest(r)

		expected := ListHostsRequest{
			MaxResults: 1,
			PageToken:  "foo",
		}
		if *listReq != expected {
			t.Errorf("expected <<%+v>>, got %+v", expected, listReq)
		}
	})
}

type testHostURLResolver struct {
	hostURL *url.URL
}

func (r *testHostURLResolver) GetHostURL(_ string, _ string) (*url.URL, error) {
	return r.hostURL, nil
}

func TestHostForwarderInvalidRequests(t *testing.T) {
	zone := "foo"
	hf := HostForwarder{
		URLResolver: &testHostURLResolver{},
	}

	cases := []struct {
		reqURL string
		// Needed to manually set mux Vars as the parsing is done by the router.
		vars map[string]string
	}{
		{
			reqURL: "http://test.com/v1/zones",
		},
		{
			reqURL: fmt.Sprintf("http://test.com/v1/zones/%s/hosts", zone),
			vars:   map[string]string{"zone": zone},
		},
	}

	for _, c := range cases {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", c.reqURL, nil)
		// Manually set mux Vars as the parsing is done by the router.
		r = mux.SetURLVars(r, c.vars)

		err := hf.Handler()(w, r)

		if err == nil {
			t.Error("expected error")
		}
	}
}

const headerContentType = "Content-Type"

func TestHostForwarderRequest(t *testing.T) {
	respContentType := "app/ct"
	respContent := "lorem ipsum"
	respStatusCode := http.StatusNotFound
	zone := "foo"
	host := "bar"
	reqURL := fmt.Sprintf("http://test.com/v1/zones/%s/hosts/%s/cvds?baz=1", zone, host)
	postRequestBody := "duis feugiat"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedReceivedURL := "/cvds?baz=1"
		if r.URL.String() != expectedReceivedURL {
			t.Fatalf("expected url <<%q>>, got: %q", expectedReceivedURL, r.URL.String())
		}
		expectedBody := ""
		if r.Method == "POST" {
			expectedBody = postRequestBody
		}
		b, _ := ioutil.ReadAll(r.Body)
		if string(b) != expectedBody {
			t.Fatalf("expected body <<%q>>, got: %q", expectedBody, string(b))
		}
		w.Header().Set("Content-Type", respContentType)
		w.WriteHeader(respStatusCode)
		w.Write([]byte(respContent))
	}))
	hostURL, _ := url.Parse(ts.URL)
	hf := HostForwarder{
		URLResolver: &testHostURLResolver{hostURL: hostURL},
	}

	tests := []struct {
		method  string
		reqBody string
	}{
		{method: "GET", reqBody: ""},
		{method: "POST", reqBody: postRequestBody},
	}

	for _, tt := range tests {

		t.Run(fmt.Sprintf("request - %s", tt.method), func(t *testing.T) {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest(tt.method, reqURL, bytes.NewBuffer([]byte(tt.reqBody)))
			// Manually set mux Vars as the parsing is done by the router.
			r = mux.SetURLVars(r, map[string]string{"zone": zone, "host": host})

			err := hf.Handler()(w, r)

			if err != nil {
				t.Errorf("expected nil error, got %+v", err)
			}
			if w.Header()[headerContentType][0] != respContentType {
				t.Errorf("expected <<%q>>, got: %q", respContentType, w.Header()[headerContentType])
			}
			if w.Result().StatusCode != respStatusCode {
				t.Errorf("expected <<%+v>>, got: %+v", respStatusCode, w.Result().StatusCode)
			}
			b, _ := ioutil.ReadAll(w.Result().Body)
			if string(b) != respContent {
				t.Errorf("expected <<%q>>, got: %q", respContent, string(b))
			}
		})
	}
}

func TestHostForwarderHostAsHostResource(t *testing.T) {
	var receivedURL string
	zone := "foo"
	host := "bar"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.Write([]byte(""))
	}))
	hostURL, _ := url.Parse(ts.URL)
	hf := HostForwarder{
		URLResolver: &testHostURLResolver{hostURL: hostURL},
	}
	w := httptest.NewRecorder()
	reqURL := fmt.Sprintf("http://test.com/v1/zones/%s/hosts/%s/hosts/%s", zone, host, host)
	r, err := http.NewRequest("GET", reqURL, nil)
	// Manually set mux Vars as the parsing is done by the router.
	r = mux.SetURLVars(r, map[string]string{"zone": zone, "host": host})

	err = hf.Handler()(w, r)

	if err != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
	expected := "/hosts/bar"
	if receivedURL != expected {
		t.Errorf("expected <<%q>>, got: %q", expected, receivedURL)
	}
}

func assertIsAppError(t *testing.T, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Errorf("error type <<\"%T\">> not found in error chain", appErr)
	}
}
