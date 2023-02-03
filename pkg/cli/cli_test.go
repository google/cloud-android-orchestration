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

package cli

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	"github.com/google/go-cmp/cmp"
)

func TestRequiredFlags(t *testing.T) {
	tests := []struct {
		Name      string
		FlagNames []string
		Args      []string
	}{
		{
			Name:      "host create",
			FlagNames: []string{serviceURLFlag},
			Args:      []string{"host", "create"},
		},
		{
			Name:      "cvd create",
			FlagNames: []string{hostFlag, serviceURLFlag},
			Args:      []string{"cvd", "create"},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			io, _, _ := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams: io,
				Args:      test.Args,
			}

			err := NewCVDRemoteCommand(opts).Execute()

			// Asserting against the error message itself as there's no specific error type for
			// required flags based failures.
			expErrMsg := fmt.Sprintf(`required flag(s) %s not set`, strings.Join(test.FlagNames, ", "))
			if diff := cmp.Diff(expErrMsg, strings.ReplaceAll(err.Error(), "\"", "")); diff != "" {
				t.Errorf("err mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type fakeService struct{}

func (fakeService) CreateHost(req *apiv1.CreateHostRequest) (*apiv1.HostInstance, error) {
	return &apiv1.HostInstance{Name: "foo"}, nil
}

func (fakeService) ListHosts() (*apiv1.ListHostsResponse, error) {
	return &apiv1.ListHostsResponse{
		Items: []*apiv1.HostInstance{{Name: "foo"}, {Name: "bar"}},
	}, nil
}

func (fakeService) DeleteHosts(name []string) error {
	return nil
}

func (fakeService) GetInfraConfig(host string) (*apiv1.InfraConfig, error) {
	return nil, nil
}

func (fakeService) ConnectWebRTC(host, device string, observer wclient.Observer) (*wclient.Connection, error) {
	return nil, nil
}

func (fakeService) CreateCVD(host string, req *hoapi.CreateCVDRequest) (*hoapi.CVD, error) {
	return &hoapi.CVD{Name: "cvd-1"}, nil
}

func (fakeService) ListCVDs(host string) ([]*hoapi.CVD, error) {
	return nil, nil
}

func (fakeService) CreateUpload(host string) (string, error) {
	return "", nil
}

func (fakeService) UploadFiles(host, uploadDir string, filenames []string) error {
	return nil
}

func TestCommandSucceeds(t *testing.T) {
	const serviceURL = "http://waldo.com"
	tests := []struct {
		Name   string
		Args   []string
		ExpOut string
	}{
		{
			Name:   "host create",
			Args:   []string{"host", "create"},
			ExpOut: "foo\n",
		},
		{
			Name:   "host list",
			Args:   []string{"host", "list"},
			ExpOut: "foo\nbar\n",
		},
		{
			Name:   "host delete",
			Args:   []string{"host", "delete", "foo", "bar"},
			ExpOut: "",
		},
		{
			Name:   "cvd create",
			Args:   []string{"cvd", "create", "--host=foo", "--build_id=123"},
			ExpOut: cvdOutput(serviceURL, "foo", "cvd-1"),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			io, _, out := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams: io,
				Args:      append(test.Args, "--service_url="+serviceURL),
				ServiceBuilder: func(opts *client.ServiceOptions) (client.Service, error) {
					return &fakeService{}, nil
				},
			}

			err := NewCVDRemoteCommand(opts).Execute()

			if err != nil {
				t.Fatal(err)
			}
			b, _ := ioutil.ReadAll(out)
			if diff := cmp.Diff(test.ExpOut, string(b)); diff != "" {
				t.Errorf("standard output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newTestIOStreams() (IOStreams, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := io.Discard

	return IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}, in, out
}

func cvdOutput(serviceURL, host, cvd string) string {
	output := CVDOutput{
		BaseURL: serviceURL + "/v1",
		Host:    "foo",
		CVD: &hoapi.CVD{
			Name: "cvd-1",
		},
	}
	return output.String() + "\n"
}
