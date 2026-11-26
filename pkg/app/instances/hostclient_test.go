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

package instances

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNetHostClientReverseProxySecurityHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src *")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><script>alert(1)</script></html>"))
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	hc := NewNetHostClient(backendURL, false)
	proxy := httptest.NewServer(hc.GetReverseProxy())
	defer proxy.Close()

	tests := []struct {
		name        string
		path        string
		wantSandbox bool
	}{
		{name: "operator root", path: "/", wantSandbox: false},
		{name: "operator root with query", path: "/?groupId=cvd", wantSandbox: false},
		{name: "operator index", path: "/index.html", wantSandbox: false},
		{name: "device webrtc client", path: "/devices/cvd-1/files/client.html", wantSandbox: false},
		{name: "device files dir", path: "/devices/cvd-1/files/", wantSandbox: false},
		{name: "device files double slash", path: "//devices/cvd-1/files/client.html", wantSandbox: false},
		{name: "device openwrt ui", path: "/devices/cvd-1/openwrt/cgi-bin/luci", wantSandbox: false},
		{name: "devices list json", path: "/devices", wantSandbox: true},
		{name: "device info json", path: "/devices/cvd-1", wantSandbox: true},
		{name: "cvds api", path: "/cvds", wantSandbox: true},
		{name: "cvd logs", path: "/cvds/cvd-1/ins-1/logs/logcat", wantSandbox: true},
		{name: "bugreport download", path: "/cvdbugreports/1234", wantSandbox: true},
		{name: "user artifacts", path: "/v1/userartifacts/abcd", wantSandbox: true},
		{name: "arbitrary html", path: "/xss.html", wantSandbox: true},
		{name: "traversal out of device files", path: "/devices/cvd-1/files/../../xss.html", wantSandbox: true},
		{name: "encoded dot in path", path: "/devices/cvd-1/files/%2e%2e/xss.html", wantSandbox: true},
		{name: "encoded slash in path", path: "/devices/cvd-1%2ffiles/client.html", wantSandbox: true},
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, proxy.URL+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
			}
			gotCSP := res.Header.Get("Content-Security-Policy")
			if tc.wantSandbox && gotCSP != "sandbox" {
				t.Errorf("Content-Security-Policy = %q, want %q", gotCSP, "sandbox")
			}
			if !tc.wantSandbox && gotCSP == "sandbox" {
				t.Errorf("unexpected Content-Security-Policy: sandbox on WebUI path %q", tc.path)
			}
		})
	}
}
