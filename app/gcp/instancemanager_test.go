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
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"

	"github.com/sergi/go-diff/diffmatchpatch"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var testConfig = app.IMConfig{
	GCP: &app.GCPIMConfig{
		ProjectID: "google.com:test-project",
		HostImage: "projects/test-project-releases/global/images/img-001",
	},
}

var testNameGenerator = &testConstantNameGenerator{name: "foo"}

type TestUserInfo struct{}

func (i *TestUserInfo) Username() string {
	return "johndoe"
}

func TestCreateHostInvalidRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &compute.Operation{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}
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
		t.Fatalf("the valid request is not valid with error %+v", err)
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
		replyJSON(w, &compute.Operation{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

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

	expected := "/projects/google.com:test-project/zones/us-central1-a/instances"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%s>>, want: %s", pathSent, expected)
	}
}

func TestCreateHostRequestBody(t *testing.T) {
	var bodySent compute.Instance
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(body, &bodySent)
		replyJSON(w, &compute.Operation{Name: "operation-1"})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

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
  "name": "foo",
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
	r := prettyJSON(t, &bodySent)
	if r != expected {
		t.Errorf("unexpected body, diff: %s", diffPrettyText(r, expected))
	}
}

func TestCreateHostSuccess(t *testing.T) {
	expectedName := "operation-1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := &compute.Operation{
			Name:   expectedName,
			Status: "DONE",
		}
		replyJSON(w, o)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

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

	if op.Name != expectedName {
		t.Errorf("expected <<%q>>, got: %q", expectedName, op.Name)
	}
	if !op.Done {
		t.Error("expected true")
	}
}

func TestGetHostAddrRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &compute.Instance{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

	im.GetHostAddr("us-central1-a", "foo")

	expected := "/projects/google.com:test-project/zones/us-central1-a/instances/foo"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%q>>, want: %q", pathSent, expected)
	}
}

func TestGetHostAddrMissingNetworkInterface(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &compute.Instance{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

	_, err := im.GetHostAddr("us-central1-a", "foo")

	var appErr *app.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("unexpected error <<\"%v\">>, want \"%T\"", err, appErr)
	}
}

func TestGetHostAddrSuccess(t *testing.T) {
	expectedIP := "10.128.0.63"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := &compute.Instance{
			NetworkInterfaces: []*compute.NetworkInterface{
				{
					NetworkIP: expectedIP,
				},
			},
		}
		replyJSON(w, i)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

	addr, _ := im.GetHostAddr("us-central1-a", "foo")

	if addr != expectedIP {
		t.Errorf("unexpected host address <<%q>>, want: %q", addr, expectedIP)
	}
}

func TestListHostsRequestQuery(t *testing.T) {
	var usedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedQuery = r.URL.Query().Encode()
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}
	req := &app.ListHostsRequest{
		MaxResults: 100,
		PageToken:  "foo",
	}

	im.ListHosts("us-central1-a", &TestUserInfo{}, req)

	expected := "alt=json&filter=labels.cf-created_by%3Ajohndoe&maxResults=100&pageToken=foo&prettyPrint=false"
	if usedQuery != expected {
		t.Errorf("expected <<%q>>, got %q", expected, usedQuery)
	}
}

func TestListHostsOverMaxResultsLimit(t *testing.T) {
	var usedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usedQuery = r.URL.Query().Encode()
		replyJSON(w, &compute.InstanceList{})
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}
	req := &app.ListHostsRequest{
		MaxResults: 501,
		PageToken:  "foo",
	}

	im.ListHosts("us-central1-a", &TestUserInfo{}, req)

	expected := "alt=json&filter=labels.cf-created_by%3Ajohndoe&maxResults=500&pageToken=foo&prettyPrint=false"
	if usedQuery != expected {
		t.Errorf("expected <<%q>>, got %q", expected, usedQuery)
	}
}

func TestListHostsSucceeds(t *testing.T) {
	i1 := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 10}},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}
	i2 := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 20}},
		Name:           "bar",
		MachineType:    "mtbaz",
		MinCpuPlatform: "mcpbaz",
	}
	nextPageToken := "test-token"
	items := []*compute.Instance{i1, i2}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		list := &compute.InstanceList{
			Items:         items,
			NextPageToken: "test-token",
		}
		replyJSON(w, list)
	}))
	defer ts.Close()
	testService := buildTestService(t, ts)
	im := InstanceManager{
		Config:                testConfig,
		Service:               testService,
		InstanceNameGenerator: testNameGenerator,
	}

	resp, err := im.ListHosts("us-central1-a", &TestUserInfo{}, &app.ListHostsRequest{})

	if err != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
	if resp.NextPageToken != nextPageToken {
		t.Errorf("expected <<%q>>, got %q", nextPageToken, resp.NextPageToken)
	}
	for i := range resp.Items {
		expected, _ := BuildHostInstance(items[i])
		if !reflect.DeepEqual(resp.Items[i], expected) {
			t.Errorf("unexpected host instance with diff: %s",
				diffPrettyText(prettyJSON(t, *expected), prettyJSON(t, *resp.Items[0])))
		}
	}
}

func TestBuildHostInstance(t *testing.T) {
	input := &compute.Instance{
		Disks:          []*compute.AttachedDisk{{DiskSizeGb: 10}},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}

	result, err := BuildHostInstance(input)

	expected := `{
  "name": "foo",
  "gcp": {
    "disk_size_gb": 10,
    "machine_type": "mt",
    "min_cpu_platform": "mcp"
  }
}`
	r := prettyJSON(t, result)
	if r != expected {
		t.Errorf("unexpected host instance, got diff: %s", diffPrettyText(r, expected))
	}
	if err != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
}

func TestBuildHostInstanceNoDisk(t *testing.T) {
	input := &compute.Instance{
		Disks:          []*compute.AttachedDisk{},
		Name:           "foo",
		MachineType:    "mt",
		MinCpuPlatform: "mcp",
	}

	result, err := BuildHostInstance(input)

	var appErr *app.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("error type <<\"%T\">> not found in error chain, got %v", appErr, err)
	}
	if result != nil {
		t.Errorf("expected <<nil>>, got %+v", err)
	}
}

func buildTestService(t *testing.T, s *httptest.Server) *compute.Service {
	srv, err := compute.NewService(
		context.TODO(),
		option.WithHTTPClient(s.Client()),
		option.WithEndpoint(s.URL),
	)
	if err != nil {
		t.Fatalf("failed to create compute service withe error: %+v", err)
	}
	return srv
}

func prettyJSON(t *testing.T, obj interface{}) string {
	s, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(s)
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

type testConstantNameGenerator struct {
	name string
}

func (g *testConstantNameGenerator) NewName() string {
	return g.name
}
