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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/config"
	"github.com/google/cloud-android-orchestration/pkg/app/database"
	"github.com/google/cloud-android-orchestration/pkg/app/encryption"
	apperr "github.com/google/cloud-android-orchestration/pkg/app/errors"
	"github.com/google/cloud-android-orchestration/pkg/app/instances"
	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
	"github.com/google/cloud-android-orchestration/pkg/app/session"

	"golang.org/x/oauth2"
)

const pageNotFoundErrMsg = "404 page not found\n"

const testUsername = "johndoe"

type testUser struct{}

func (i *testUser) Username() string { return testUsername }

type testAccountManager struct{}

func (m *testAccountManager) UserFromRequest(r *http.Request) (accounts.User, error) {
	return &testUser{}, nil
}

func (m *testAccountManager) OnOAuth2Exchange(w http.ResponseWriter, r *http.Request, tk appOAuth2.IDTokenClaims) (accounts.User, error) {
	return &testUser{}, nil
}

type testInstanceManager struct {
	hostClientFactory func(zone, host string) instances.HostClient
}

func (m *testInstanceManager) GetHostURL(zone string, host string) (*url.URL, error) {
	return url.Parse("http://127.0.0.1:8080")
}

func (m *testInstanceManager) CreateHost(_ string, _ *apiv1.CreateHostRequest, _ accounts.User) (*apiv1.Operation, error) {
	return &apiv1.Operation{}, nil
}

func (m *testInstanceManager) ListHosts(zone string, user accounts.User, req *instances.ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	return &apiv1.ListHostsResponse{}, nil
}

func (m *testInstanceManager) DeleteHost(zone string, user accounts.User, name string) (*apiv1.Operation, error) {
	return &apiv1.Operation{}, nil
}

func (m *testInstanceManager) WaitOperation(_ string, _ accounts.User, _ string) (any, error) {
	return struct{}{}, nil
}

func (m *testInstanceManager) GetHostClient(zone string, host string) (instances.HostClient, error) {
	return m.hostClientFactory(zone, host), nil
}

type testHostClient struct {
	url *url.URL
}

func (hc *testHostClient) Get(path, query string, res *instances.HostResponse) (int, error) {
	return 200, nil
}

func (hc *testHostClient) Post(path, query string, bodyJSON any, res *instances.HostResponse) (int, error) {
	return 200, nil
}

func (hc *testHostClient) GetReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(hc.url)
}

func TestCreateHostSucceeds(t *testing.T) {
	controller := NewApp(&testInstanceManager{}, &testAccountManager{}, nil, nil, nil, "", config.WebRTCConfig{})
	ts := httptest.NewServer(controller.Handler())
	defer ts.Close()

	res, _ := http.Post(
		ts.URL+"/v1/zones/us-central1-a/hosts", "application/json", strings.NewReader("{}"))

	expected := http.StatusOK
	if res.StatusCode != expected {
		t.Errorf("unexpected status code <<%d>>, want: %d", res.StatusCode, expected)
	}
}

func TestWaitOperatioSucceeds(t *testing.T) {
	controller := NewApp(&testInstanceManager{}, &testAccountManager{}, nil, nil, nil, "", config.WebRTCConfig{})
	ts := httptest.NewServer(controller.Handler())
	defer ts.Close()

	res, _ := http.Post(
		ts.URL+"/v1/zones/us-central1-a/operations/foo/:wait", "application/json", strings.NewReader("{}"))

	expected := http.StatusOK
	if res.StatusCode != expected {
		t.Errorf("unexpected status code <<%d>>, want: %d", res.StatusCode, expected)
	}
}

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

		expected := instances.ListHostsRequest{
			MaxResults: 1,
			PageToken:  "foo",
		}
		if *listReq != expected {
			t.Errorf("expected <<%+v>>, got %+v", expected, listReq)
		}
	})
}

func TestDeleteHostIsHandled(t *testing.T) {
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/v1/zones/foo/hosts/bar", nil)
	if err != nil {
		t.Fatal(err)
	}
	controller := NewApp(&testInstanceManager{}, &testAccountManager{}, nil, nil, nil, "", config.WebRTCConfig{})

	makeRequest(rr, req, controller)

	if rr.Code == http.StatusNotFound && rr.Body.String() == pageNotFoundErrMsg {
		t.Errorf("request was not handled. This failure implies an API breaking change.")
	}
}

func TestHostForwarderRequest(t *testing.T) {
	const headerContentType = "Content-Type"
	respContentType := "app/ct"
	respContent := "lorem ipsum"
	respStatusCode := http.StatusNotFound
	zone := "foo"
	host := "bar"
	resource := "devices"
	reqURL := fmt.Sprintf("http://test.com/v1/zones/%s/hosts/%s/%s?baz=1", zone, host, resource)
	postRequestBody := "duis feugiat"
	expectedReceivedURL := "/devices?baz=1"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != expectedReceivedURL {
			t.Fatalf("expected url <<%q>>, got: %q", expectedReceivedURL, r.URL.String())
		}
		expectedBody := ""
		if r.Method == "POST" {
			expectedBody = postRequestBody
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != expectedBody {
			t.Fatalf("expected body <<%q>>, got: %q", expectedBody, string(b))
		}
		w.Header().Set("Content-Type", respContentType)
		w.WriteHeader(respStatusCode)
		w.Write([]byte(respContent))
	}))
	hostURL, _ := url.Parse(ts.URL)
	controller := NewApp(&testInstanceManager{
		hostClientFactory: func(_, _ string) instances.HostClient {
			return &testHostClient{hostURL}
		},
	}, &testAccountManager{}, nil, nil, nil, "", config.WebRTCConfig{})

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
			req, _ := http.NewRequest(tt.method, reqURL, bytes.NewBuffer([]byte(tt.reqBody)))

			makeRequest(w, req, controller)

			if w.Header()[headerContentType][0] != respContentType {
				t.Errorf("expected <<%q>>, got: %q", respContentType, w.Header()[headerContentType])
			}
			if w.Result().StatusCode != respStatusCode {
				t.Errorf("expected <<%+v>>, got: %+v", respStatusCode, w.Result().StatusCode)
			}
			b, _ := io.ReadAll(w.Result().Body)
			if string(b) != respContent {
				t.Errorf("expected <<%q>>, got: %q", respContent, string(b))
			}
		})
	}
}

func TestHostForwarderInvalidRequests(t *testing.T) {
	zone := "foo"
	host := "bar"
	cases := []string{
		"http://test.com/v1/zones",
		fmt.Sprintf("http://test.com/v1/zones/%s/hosts", zone),
	}
	for _, c := range cases {
		u, err := url.Parse(c)
		if err != nil {
			t.Errorf("Failed to parse test url: %v", err)
		}
		_, err = HostOrchestratorPath(u.Path, host)
		if err == nil {
			t.Errorf("Expected OrchestratorPath to fail")
		}
	}
}

func TestHostForwarderHostAsHostResource(t *testing.T) {
	host := "bar"
	reqURL := "http://test.com/v1/zones/foo/hosts/bar/hosts/bar"
	u, err := url.Parse(reqURL)
	if err != nil {
		t.Error(err)
	}
	expected := "/hosts/bar"
	path, err := HostOrchestratorPath(u.Path, host)
	if err != nil {
		t.Error(err)
	}
	if path != expected {
		t.Errorf("expected <<%q>>, got: %q", expected, path)
	}
}

func TestHostForwarderInjectCredentials(t *testing.T) {
	reqURL := "http://test.com/v1/zones/foo/hosts/bar/cvds"
	msg, _ := json.Marshal(&hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			Name: "cvdname",
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuildSource: &hoapi.AndroidCIBuildSource{},
			},
		},
	})
	credentials := "abcdef"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		var msg hoapi.CreateCVDRequest
		if err := json.Unmarshal(body, &msg); err != nil {
			t.Errorf("Failed to decode message: %v", err)
		}
		if msg.CVD == nil || msg.CVD.BuildSource == nil || msg.CVD.BuildSource.AndroidCIBuildSource == nil {
			t.Errorf("Expected android ci request, got: %s", string(body))
		}
		if msg.CVD.BuildSource.AndroidCIBuildSource.Credentials == "" {
			t.Errorf("No credentials were injected")
		}
		if msg.CVD.BuildSource.AndroidCIBuildSource.Credentials != credentials {
			t.Errorf("Wrong injected credentials: expected %q got %q", credentials,
				msg.CVD.BuildSource.AndroidCIBuildSource.Credentials)
		}
		w.Write([]byte("ok"))
	}))
	hostURL, _ := url.Parse(ts.URL)
	dbs := database.NewInMemoryDBService()
	tk := &oauth2.Token{
		AccessToken:  credentials,
		TokenType:    "Brearer",
		RefreshToken: "",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	es := encryption.NewFakeEncryptionService()
	jsonToken, err := json.Marshal(tk)
	if err != nil {
		t.Error(err)
	}
	encryptedJSONToken, err := es.Encrypt(jsonToken)
	if err != nil {
		t.Error(err)
	}
	dbs.StoreBuildAPICredentials(testUsername, encryptedJSONToken)
	controller := NewApp(&testInstanceManager{
		hostClientFactory: func(_, _ string) instances.HostClient {
			return &testHostClient{hostURL}
		},
	}, &testAccountManager{}, nil, encryption.NewFakeEncryptionService(), dbs, "", config.WebRTCConfig{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(msg))

	makeRequest(w, req, controller)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected <<%+v>>, got: %+v", http.StatusOK, w.Result().StatusCode)
	}
}

func TestHostForwarderDoesNotInjectCredentials(t *testing.T) {
	reqURL := "http://test.com/v1/zones/foo/hosts/bar/cvds"
	msg, _ := json.Marshal(&hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			Name: "cvdname",
			BuildSource: &hoapi.BuildSource{
				UserBuildSource: &hoapi.UserBuildSource{},
			},
		},
	})
	credentials := "abcdef"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != string(msg) {
			t.Errorf("Original message was modified by the controller: %q, expected: %q", string(body), string(msg))
		}
		w.Write([]byte("ok"))
	}))
	hostURL, _ := url.Parse(ts.URL)
	dbs := database.NewInMemoryDBService()
	tk := &oauth2.Token{
		AccessToken:  credentials,
		TokenType:    "Brearer",
		RefreshToken: "",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	es := encryption.NewFakeEncryptionService()
	jsonToken, err := json.Marshal(tk)
	if err != nil {
		t.Error(err)
	}
	encryptedJSONToken, err := es.Encrypt(jsonToken)
	if err != nil {
		t.Error(err)
	}
	dbs.StoreBuildAPICredentials(testUsername, encryptedJSONToken)
	controller := NewApp(&testInstanceManager{
		hostClientFactory: func(_, _ string) instances.HostClient {
			return &testHostClient{hostURL}
		},
	}, &testAccountManager{}, nil, nil, dbs, "", config.WebRTCConfig{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(msg))

	makeRequest(w, req, controller)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected <<%+v>>, got: %+v", http.StatusOK, w.Result().StatusCode)
	}
}

func TestBadCSRFTokensInRescindAuth(t *testing.T) {
	testData := []struct {
		Name   string
		Values url.Values
	}{{"MissingToken", url.Values{}}, {"WrongToken", url.Values{"csrf_token": []string{"wrongtoken"}}}}

	const sessionId = "somesessionid"

	for _, td := range testData {
		t.Run(td.Name, func(t *testing.T) {
			dbs := database.NewInMemoryDBService()
			dbs.CreateOrUpdateSession(session.Session{
				Key:         sessionId,
				OAuth2State: "righttoken",
			})
			controller := NewApp(&testInstanceManager{}, &testAccountManager{}, nil, nil, dbs, "", config.WebRTCConfig{})
			ts := httptest.NewServer(controller.Handler())
			defer ts.Close()

			req, err := http.NewRequest("POST", ts.URL+"/deauth", strings.NewReader(td.Values.Encode()))
			if err != nil {
				t.Fatal(err)
			}
			req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
			req.AddCookie(&http.Cookie{
				Name:  sessionIdCookie,
				Value: sessionId,
			})
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			expected := http.StatusBadRequest
			if res.StatusCode != expected {
				t.Fatalf("unexpected status code <<%d>>, want: %d", res.StatusCode, expected)
			}
			rBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(rBody), "CSRF token") {
				t.Fatalf("Expected the request message to mention CSRF token: %s", string(rBody))
			}
		})
	}

}

func assertIsAppError(t *testing.T, err error) {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("error type <<\"%T\">> not found in error chain", appErr)
	}
}

func makeRequest(w http.ResponseWriter, r *http.Request, controller *App) {
	router := controller.Handler()
	router.ServeHTTP(w, r)
}
