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

package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"

	"github.com/sergi/go-diff/diffmatchpatch"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

var testConfig = &app.Config{
	GCPConfig: &app.GCPConfig{
		ProjectID:   "google.com:test-project",
		SourceImage: "projects/test-project-releases/global/images/img-001",
	},
}

type TestUserInfo struct{}

func (i *TestUserInfo) Username() string {
	return "johndoe"
}

func TestCreateHostInvalidRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &computepb.Operation{})
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()
	var validRequest = func() *apiv1.CreateHostRequest {
		return &apiv1.CreateHostRequest{
			CreateHostInstanceRequest: &apiv1.CreateHostInstanceRequest{
				GCP: &apiv1.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		}
	}
	// Make sure the valid request is indeed valid.
	_, err := im.CreateHost("us-central1-a", validRequest(), &TestUserInfo{})
	if err != nil {
		t.Fatalf("the valid request is not valid")
	}
	var tests = []struct {
		corruptRequest func(r *apiv1.CreateHostRequest)
	}{
		{func(r *apiv1.CreateHostRequest) { r.CreateHostInstanceRequest = nil }},
		{func(r *apiv1.CreateHostRequest) { r.CreateHostInstanceRequest.GCP = nil }},
		{func(r *apiv1.CreateHostRequest) { r.CreateHostInstanceRequest.GCP.DiskSizeGB = 0 }},
		{func(r *apiv1.CreateHostRequest) { r.CreateHostInstanceRequest.GCP.MachineType = "" }},
	}

	for _, test := range tests {
		req := validRequest()
		test.corruptRequest(req)
		_, err := im.CreateHost("us-central1-a", req, &TestUserInfo{})
		var appErr *app.AppError
		if !errors.As(err, &appErr) {
			t.Errorf("unexpected error <<\"%v\">>, want \"%T\"", err, appErr)
		}
	}
}

func TestCreateHostRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &computepb.Operation{})
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()

	im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			CreateHostInstanceRequest: &apiv1.CreateHostInstanceRequest{
				GCP: &apiv1.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUserInfo{})

	expected := "/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%s>>, want: %s", pathSent, expected)
	}
}

func TestCreateHostRequestBody(t *testing.T) {
	var bodySent []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err == nil {
			bodySent = b
		}
		replyJSON(w, &computepb.Operation{Name: proto.String("operation-16482")})
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	im.setUUIDFactory(func() string { return "123e4567" })
	defer im.Close()

	im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			CreateHostInstanceRequest: &apiv1.CreateHostInstanceRequest{
				GCP: &apiv1.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUserInfo{})

	expected := `{
  "disks": [
    {
      "boot": true,
      "initializeParams": {
        "diskSizeGb": "100",
        "sourceImage": "projects/test-project-releases/global/images/img-001"
      }
    }
  ],
  "labels": {
    "cf-created_by": "johndoe",
    "created_by": "johndoe"
  },
  "machineType": "zones/us-central1-f/machineTypes/n1-standard-1",
  "minCpuPlatform": "Intel Haswell",
  "name": "cf-123e4567",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "name": "External NAT",
          "type": "ONE_TO_ONE_NAT"
        }
      ],
      "name": "projects/google.com:test-project/global/networks/default"
    }
  ]
}`
	r := prettyJSON(t, bodySent)
	if r != expected {
		t.Errorf("unexpected body, diff: %s", diffPrettyText(r, expected))
	}
}

func TestCreateHostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := &computepb.Operation{
			Name:   proto.String("operation-123"),
			Status: computepb.Operation_DONE.Enum(),
		}
		replyJSON(w, o)
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()

	op, _ := im.CreateHost("us-central1-a",
		&apiv1.CreateHostRequest{
			CreateHostInstanceRequest: &apiv1.CreateHostInstanceRequest{
				GCP: &apiv1.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUserInfo{})

	expected := &apiv1.Operation{Name: "operation-123", Done: true}
	if *op != *expected {
		t.Errorf("unexpected operation: <<%v>>, want: %v", *op, *expected)
	}
}

func TestGetHostAddrRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &computepb.Instance{})
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()

	im.GetHostAddr("us-central1-a", "cf-123e4567")

	expected := "/compute/v1/projects/google.com:test-project/zones/us-central1-a/instances/cf-123e4567"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%q>>, want: %q", pathSent, expected)
	}
}

func TestGetHostAddrMissingNetworkInterface(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &computepb.Instance{})
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()

	_, err := im.GetHostAddr("us-central1-a", "cf-123e4567")

	var appErr *app.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("unexpected error <<\"%v\">>, want \"%T\"", err, appErr)
	}
}

func TestGetHostAddrSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := &computepb.Instance{
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					NetworkIP: proto.String("10.128.0.63"),
				},
			},
		}
		replyJSON(w, i)
	}))
	defer ts.Close()
	im := newTestInstanceManager(t, ts)
	defer im.Close()

	addr, _ := im.GetHostAddr("us-central1-a", "cf-123e4567")

	expected := "10.128.0.63"
	if addr != expected {
		t.Errorf("unexpected host address <<%q>>, want: %q", addr, expected)
	}
}

func newTestInstanceManager(t *testing.T, s *httptest.Server) *InstanceManager {
	im, err := NewInstanceManager(
		testConfig,
		context.TODO(),
		option.WithEndpoint(s.URL),
		option.WithHTTPClient(s.Client()))
	if err != nil {
		t.Fatalf("failed to create new test GCPInstanceManager with error: %v", err)
	}
	return im
}

func prettyJSON(t *testing.T, b []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, b, "", "  "); err != nil {
		t.Fatalf("failed to prettyfi JSON, error: %v", err)
	}
	return prettyJSON.String()
}

func diffPrettyText(result string, expected string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(result, expected, false)
	return dmp.DiffPrettyText(diffs)
}

func replyJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}
