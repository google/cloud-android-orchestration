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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
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
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			io, _, _ := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams:      io,
				Args:           test.Args,
				CommandRunner:  &fakeCommandRunner{},
				ADBServerProxy: &fakeADBServerProxy{},
				InitialConfig: Config{
					ConnectionControlDir: t.TempDir(),
				},
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

type fakeCommandRunner struct{}

func (_ *fakeCommandRunner) StartBgCommand(...string) ([]byte, error) {
	// The only command started for now is the connection agent.
	return json.Marshal(&ConnStatus{ADB: ForwarderState{Port: 12345}})
}

type fakeADBServerProxy struct{}

func (*fakeADBServerProxy) Connect(int) error {
	return nil
}

func (*fakeADBServerProxy) Disconnect(int) error {
	return nil
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

func (fakeService) ConnectWebRTC(host, device string, observer wclient.Observer, logger io.Writer) (*wclient.Connection, error) {
	return nil, nil
}

func (fakeService) CreateCVD(host string, req *hoapi.CreateCVDRequest) (*hoapi.CreateCVDResponse, error) {
	if host == "" {
		panic("empty host")
	}
	return &hoapi.CreateCVDResponse{CVDs: []*hoapi.CVD{{Name: "cvd-1"}}}, nil
}

func (fakeService) ListCVDs(host string) ([]*hoapi.CVD, error) {
	return []*hoapi.CVD{{Name: "cvd-1"}}, nil
}

func (fakeService) CreateUpload(host string) (string, error) {
	return "", nil
}

func (fakeService) UploadFiles(host, uploadDir string, filenames []string) error {
	return nil
}

func (fakeService) DownloadRuntimeArtifacts(host string, dst io.Writer) error {
	return nil
}

const serviceURL = "http://waldo.com"

func (fakeService) RootURI() string {
	return serviceURL + "/v1"
}

func TestCommandSucceeds(t *testing.T) {
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
			Name:   "create",
			Args:   []string{"create", "--build_id=123"},
			ExpOut: cvdOutput(serviceURL, "foo", hoapi.CVD{Name: "cvd-1"}, 12345),
		},
		{
			Name:   "create with --host",
			Args:   []string{"create", "--host=bar", "--build_id=123"},
			ExpOut: cvdOutput(serviceURL, "bar", hoapi.CVD{Name: "cvd-1"}, 12345),
		},
		{
			Name: "list",
			Args: []string{"list"},
			ExpOut: cvdOutput(serviceURL, "foo", hoapi.CVD{Name: "cvd-1"}, 0) +
				cvdOutput(serviceURL, "bar", hoapi.CVD{Name: "cvd-1"}, 0),
		},
		{
			Name:   "list with --host",
			Args:   []string{"list", "--host=bar"},
			ExpOut: cvdOutput(serviceURL, "bar", hoapi.CVD{Name: "cvd-1"}, 0),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			io, _, out := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams: io,
				Args:      append(test.Args, "--service_url="+serviceURL),
				InitialConfig: Config{
					ConnectionControlDir: t.TempDir(),
				},
				ServiceBuilder: func(opts *client.ServiceOptions) (client.Service, error) {
					return &fakeService{}, nil
				},
				CommandRunner:  &fakeCommandRunner{},
				ADBServerProxy: &fakeADBServerProxy{},
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

func TestBuildAgentCmdline(t *testing.T) {
	/*****************************************************************
	If this test fails you most likely need to fix an AsArgs function!
	******************************************************************/
	// Don't name the fields to force a compiler error when the flag structures
	// are modified. This should help the developer realize they also need to
	// modify the corresponding AsArgs method.
	flags := ConnectFlags{
		&CVDRemoteFlags{
			"service url",
			"zone",
			"http proxy",
			true, // verbose
		},
		"host",
		false, // skipconfirmation
	}
	device := "device"
	args := buildAgentCmdArgs(&flags, device)
	var options CommandOptions
	cmd := NewCVDRemoteCommand(&options)
	subCmd, args, err := cmd.command.Traverse(args)
	// This at least ensures no required flags were left blank.
	if err != nil {
		t.Errorf("Failed to parse args: %v", err)
	}
	// Just a sanity check that all flags were parsed and only the device was
	// left as possitional argument.
	if reflect.DeepEqual(args, []string{device}) {
		t.Errorf("Expected resulting args to just have [%q], but found %v", device, args)
	}
	if subCmd.Name() != ConnectionAgentCommandName {
		t.Errorf("Expected it to parse %q command, found: %q", ConnectionAgentCommandName, subCmd.Name())
	}
	// TODO(jemoreira): Compare the parsed flags with used flags
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

func cvdOutput(serviceURL, host string, cvd hoapi.CVD, port int) string {
	out := &bytes.Buffer{}
	cvdOut := NewCVDInfo(fakeService{}.RootURI(), host, &cvd)
	cvdOut.ConnStatus = &ConnStatus{
		ADB: ForwarderState{
			State: "not connected",
			Port:  port,
		},
	}
	fmt.Fprintln(out, ToPrintableStr(cvdOut))
	b, _ := ioutil.ReadAll(out)
	return string(b)
}
