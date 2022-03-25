package main

import (
	"bytes"
	imtypes "cloud-android-orchestration/api/instancemanager/v1"
	compute "cloud.google.com/go/compute/apiv1"
	"context"
	"encoding/json"
	"errors"
	"github.com/sergi/go-diff/diffmatchpatch"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestUserInfo struct{}

func (i *TestUserInfo) Username() string {
	return "johndoe"
}

func TestInsertHostInvalidRequests(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replyJSON(w, &computepb.Operation{}, http.StatusOK)
	}))
	defer ts.Close()
	client := newTestInstancesRESTClient(t, ts)
	defer client.Close()
	var validRequest = func() *imtypes.InsertHostRequest {
		return &imtypes.InsertHostRequest{
			CVDInfo: &imtypes.CVDInfo{
				BuildID: "1234",
				Target:  "aosp_cf_x86_64_phone-userdebug",
			},
			HostInfo: &imtypes.HostInfo{
				GCP: &imtypes.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		}
	}
	// Make sure the valid request is indeed valid.
	_, err := NewGcpIM(client).InsertHost("us-central1-a", validRequest(), &TestUserInfo{})
	if err != nil {
		t.Fatalf("the valid request is not valid")
	}
	var tests = []struct {
		corruptRequest func(r *imtypes.InsertHostRequest)
	}{
		{func(r *imtypes.InsertHostRequest) { r.CVDInfo = nil }},
		{func(r *imtypes.InsertHostRequest) { r.CVDInfo.BuildID = "" }},
		{func(r *imtypes.InsertHostRequest) { r.CVDInfo.Target = "" }},
		{func(r *imtypes.InsertHostRequest) { r.HostInfo = nil }},
		{func(r *imtypes.InsertHostRequest) { r.HostInfo.GCP = nil }},
		{func(r *imtypes.InsertHostRequest) { r.HostInfo.GCP.DiskSizeGB = 0 }},
		{func(r *imtypes.InsertHostRequest) { r.HostInfo.GCP.MachineType = "" }},
	}

	for _, test := range tests {
		req := validRequest()
		test.corruptRequest(req)
		_, err := NewGcpIM(client).InsertHost("us-central1-a", req, &TestUserInfo{})
		if !errors.Is(err, ErrBadInsertHostRequest) {
			t.Errorf("unexpected error <<\"%v\">>, want \"%v\"", err, ErrBadInsertHostRequest)
		}
	}
}

func TestInsertHostRequestPath(t *testing.T) {
	var pathSent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSent = r.URL.Path
		replyJSON(w, &computepb.Operation{}, http.StatusOK)
	}))
	defer ts.Close()
	client := newTestInstancesRESTClient(t, ts)
	defer client.Close()

	NewGcpIM(client).InsertHost("us-central1-a",
		&imtypes.InsertHostRequest{
			CVDInfo: &imtypes.CVDInfo{
				BuildID: "1234",
				Target:  "aosp_cf_x86_64_phone-userdebug",
			},
			HostInfo: &imtypes.HostInfo{
				GCP: &imtypes.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUserInfo{})

	expected := "/compute/v1/projects/google.com:cloud-android-jemoreira/zones/us-central1-a/instances"
	if pathSent != expected {
		t.Errorf("unexpected url path <<%s>>, want: %s", pathSent, expected)
	}
}

func TestInsertHostRequestBody(t *testing.T) {
	// Save and restore original newUUIDString.
	savedNewUUIDString := newUUIDString
	defer func() { newUUIDString = savedNewUUIDString }()
	// Install the test's fake newUUIDString.
	newUUIDString = func() string { return "123e4567" }
	var bodySent []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err == nil {
			bodySent = b
		}
		replyJSON(w, &computepb.Operation{Name: proto.String("operation-16482")}, http.StatusOK)
	}))
	defer ts.Close()
	client := newTestInstancesRESTClient(t, ts)
	defer client.Close()

	NewGcpIM(client).InsertHost("us-central1-a",
		&imtypes.InsertHostRequest{
			CVDInfo: &imtypes.CVDInfo{
				BuildID: "1234",
				Target:  "aosp_cf_x86_64_phone-userdebug",
			},
			HostInfo: &imtypes.HostInfo{
				GCP: &imtypes.GCPInstance{
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
        "sourceImage": "projects/cloud-android-releases/global/images/cuttlefish-google-vsoc-0-9-21"
      }
    }
  ],
  "labels": {
    "cf-build_id": "1234",
    "cf-creator": "johndoe",
    "cf-target": "aosp_cf_x86_64_phone-userdebug"
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
      "name": "projects/cloud-android-jemoreira/global/networks/default"
    }
  ]
}`
	r := prettyJSON(t, bodySent)
	if r != expected {
		t.Errorf("unexpected body, diff: %s", diffPrettyText(r, expected))
	}
}

func TestInsertHostSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := &computepb.Operation{
			Name:   proto.String("operation-123"),
			Status: computepb.Operation_DONE.Enum(),
		}
		replyJSON(w, o, http.StatusOK)
	}))
	defer ts.Close()
	client := newTestInstancesRESTClient(t, ts)
	defer client.Close()

	op, _ := NewGcpIM(client).InsertHost("us-central1-a",
		&imtypes.InsertHostRequest{
			CVDInfo: &imtypes.CVDInfo{
				BuildID: "1234",
				Target:  "aosp_cf_x86_64_phone-userdebug",
			},
			HostInfo: &imtypes.HostInfo{
				GCP: &imtypes.GCPInstance{
					DiskSizeGB:     100,
					MachineType:    "zones/us-central1-f/machineTypes/n1-standard-1",
					MinCPUPlatform: "Intel Haswell",
				},
			},
		},
		&TestUserInfo{})

	expected := &imtypes.Operation{Name: "operation-123", Done: true}
	if *op != *expected {
		t.Errorf("unexpected operation: <<%v>>, want: %v", *op, *expected)
	}
}

func newTestInstancesRESTClient(t *testing.T, s *httptest.Server) *compute.InstancesClient {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx,
		option.WithEndpoint(s.URL),
		option.WithHTTPClient(s.Client()),
	)
	if err != nil {
		t.Fatalf("failed to create new instances rest client with error: %v", err)
	}
	return client
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
