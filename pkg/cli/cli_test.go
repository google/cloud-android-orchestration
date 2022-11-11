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
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-multierror"
)

func TestCVDRemoteRequiredFlags(t *testing.T) {
	tests := []struct {
		FlagName string
		Args     []string
	}{
		{
			FlagName: serviceURLFlag,
			Args:     []string{"host", "create"},
		},
	}

	for _, test := range tests {
		t.Run(test.FlagName, func(t *testing.T) {
			io, _, _ := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams: io,
				Args:      test.Args,
			}

			err := NewCVDRemoteCommand(opts).Execute()

			// Asserting against the error message itself as there's no specific error type for
			// required flags based failures.
			expErrMsg := fmt.Sprintf(`required flag(s) "%s" not set`, test.FlagName)
			if diff := cmp.Diff(expErrMsg, err.Error()); diff != "" {
				t.Errorf("err mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type createHostReqFailsHandler struct{ WithErrCode int }

func (h *createHostReqFailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "POST /v1/hosts":
		writeErr(w, h.WithErrCode)
	default:
		panic("unexpected request")
	}
}

type createHostWaitOpReqFailsHandler struct{ WithErrCode int }

func (h *createHostWaitOpReqFailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const opName = "op-foo"
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "POST /v1/hosts":
		writeOK(w, &apiv1.Operation{Name: opName})
	case "POST /v1/operations/" + opName + "/wait":
		writeErr(w, h.WithErrCode)
	default:
		panic("unexpected request")
	}
}

type createHostWaitOpNotDoneHandler struct{ WithOpName string }

func (h *createHostWaitOpNotDoneHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "POST /v1/hosts":
		writeOK(w, &apiv1.Operation{Name: h.WithOpName})
	case "POST /v1/operations/" + h.WithOpName + "/wait":
		writeOK(w, &apiv1.Operation{Name: h.WithOpName})
	default:
		panic("unexpected request")
	}
}

type createHostOpFailedHandler struct{ WithErrCode int }

func (h *createHostOpFailedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const opName = "op-foo"
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "POST /v1/hosts":
		writeOK(w, &apiv1.Operation{Name: opName})
	case "POST /v1/operations/" + opName + "/wait":
		op := &apiv1.Operation{
			Done:   true,
			Result: &apiv1.OperationResult{Error: &apiv1.Error{Code: strconv.Itoa(h.WithErrCode)}},
		}
		writeOK(w, op)
	default:
		panic("unexpected request")
	}
}

type listsHostReqFailsHandler struct{ WithErrCode int }

func (h *listsHostReqFailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "GET /v1/hosts":
		writeErr(w, h.WithErrCode)
	default:
		panic("unexpected request")
	}
}

func TestCommandFails(t *testing.T) {
	tests := []struct {
		Name       string
		Args       []string
		SrvHandler http.Handler
		ExpOut     string
		ExpErr     error
	}{
		{
			Name:       "create host api call fails",
			Args:       []string{"host", "create"},
			SrvHandler: &createHostReqFailsHandler{WithErrCode: 500},
			ExpErr:     &apiCallError{&apiv1.Error{Code: "500"}},
		},
		{
			Name:       "wait operation api call fails",
			Args:       []string{"host", "create"},
			SrvHandler: &createHostReqFailsHandler{WithErrCode: 503},
			ExpErr:     &apiCallError{&apiv1.Error{Code: "503"}},
		},
		{
			Name:       "wait operation, operation not done",
			Args:       []string{"host", "create"},
			SrvHandler: &createHostWaitOpNotDoneHandler{WithOpName: "op-foo"},
			ExpErr:     opTimeoutError("op-foo"),
		},
		{
			Name:       "failed operation",
			Args:       []string{"host", "create"},
			SrvHandler: &createHostOpFailedHandler{WithErrCode: 507},
			ExpErr:     &apiCallError{&apiv1.Error{Code: "507"}},
		},
		{
			Name:       "list hosts api call fails",
			Args:       []string{"host", "list"},
			SrvHandler: &listsHostReqFailsHandler{WithErrCode: 500},
			ExpErr:     &apiCallError{&apiv1.Error{Code: "500"}},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ts := httptest.NewServer(test.SrvHandler)
			defer ts.Close()
			io, _, out := newTestIOStreams()
			opts := &CommandOptions{
				IOStreams: io,
				Args:      append([]string{"--service_url=" + ts.URL}, test.Args[:]...),
			}

			err := NewCVDRemoteCommand(opts).Execute()

			b, _ := ioutil.ReadAll(out)
			if diff := cmp.Diff(test.ExpOut, string(b)); diff != "" {
				t.Errorf("standard output mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(test.ExpErr, err); diff != "" {
				t.Errorf("err mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

const testZone = "us-west1-c"

type alwaysSucceedsHandler struct {
	WithHostName      string
	WithHostInstances []*apiv1.HostInstance
}

func (h *alwaysSucceedsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const opName = "op-foo"
	switch ep := r.Method + " " + r.URL.Path; ep {
	case "POST /v1/hosts", "POST /v1/zones/" + testZone + "/hosts":
		writeOK(w, &apiv1.Operation{Name: opName})
	case "POST /v1/operations/" + opName + "/wait",
		"POST /v1/zones/" + testZone + "/operations/" + opName + "/wait":
		res, _ := json.Marshal(&apiv1.HostInstance{Name: h.WithHostName})
		op := &apiv1.Operation{Done: true, Result: &apiv1.OperationResult{Response: string(res)}}
		writeOK(w, op)
	case "GET /v1/hosts", "GET /v1/zones/" + testZone + "/hosts":
		writeOK(w, &apiv1.ListHostsResponse{Items: h.WithHostInstances})
	case "DELETE /v1/hosts/" + h.getHost(r.URL.Path),
		"DELETE /v1/zones/" + testZone + "/hosts/" + h.getHost(r.URL.Path):
		writeOK(w, "")
	default:
		panic("unexpected endpoint: " + ep)
	}
}

func (h *alwaysSucceedsHandler) getHost(path string) string {
	re := regexp.MustCompile(`^/v1/(zones/.*/)?hosts/(.*)$`)
	return re.FindStringSubmatch(path)[2]
}

func TestCommandSucceeds(t *testing.T) {
	tests := []struct {
		Args       []string
		SrvHandler http.Handler
		ExpOut     string
	}{
		{
			Args:       []string{"host", "create"},
			SrvHandler: &alwaysSucceedsHandler{WithHostName: "foo"},
			ExpOut:     "foo\n",
		},
		{
			Args: []string{"host", "list"},
			SrvHandler: &alwaysSucceedsHandler{
				WithHostInstances: []*apiv1.HostInstance{{Name: "foo"}, {Name: "bar"}},
			},
			ExpOut: "foo\nbar\n",
		},
		{
			Args:       []string{"host", "delete", "foo", "bar"},
			SrvHandler: &alwaysSucceedsHandler{},
			ExpOut:     "",
		},
	}
	for _, test := range tests {
		t.Run(strings.Join(test.Args, " "), func(t *testing.T) {
			ts := httptest.NewServer(test.SrvHandler)
			defer ts.Close()
			configs := []struct {
				Name string
				Args []string
			}{
				{Name: "default", Args: []string{"--service_url=" + ts.URL}},
				{Name: "having zone", Args: []string{"--service_url=" + ts.URL, "--zone=" + testZone}},
				{
					Name: "having proxy",
					Args: []string{
						"--service_url=http://foo.com",
						"--zone=" + testZone,
						"--http_proxy=" + ts.URL,
					},
				},
			}
			for _, cfg := range configs {
				t.Run("with config "+cfg.Name, func(t *testing.T) {
					io, _, out := newTestIOStreams()
					opts := &CommandOptions{
						IOStreams: io,
						Args:      append(cfg.Args, test.Args[:]...),
					}

					err := NewCVDRemoteCommand(opts).Execute()

					b, _ := ioutil.ReadAll(out)
					if diff := cmp.Diff(test.ExpOut, string(b)); diff != "" {
						t.Errorf("standard output mismatch (-want +got):\n%s", diff)
					}
					if err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}

type delHostReqHandler struct {
	WithHostNames map[string]struct{}
}

func (h *delHostReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		panic("unexpected method: " + r.Method)
	}
	re := regexp.MustCompile(`^/v1/hosts/(.*)$`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) != 2 {
		panic("unexpected path: " + r.URL.Path)
	}
	if _, ok := h.WithHostNames[matches[1]]; ok {
		writeOK(w, "")
	} else {
		writeErr(w, 404)
	}
}

func TestDeleteHostsCommandFails(t *testing.T) {
	hostNames := map[string]struct{}{"bar": {}, "baz": {}}
	srvHandler := &delHostReqHandler{hostNames}
	ts := httptest.NewServer(srvHandler)
	defer ts.Close()
	io, _, _ := newTestIOStreams()
	opts := &CommandOptions{
		IOStreams: io,
		Args:      append([]string{"--service_url=" + ts.URL}, "host", "delete", "foo", "bar", "baz", "quz"),
	}

	err := NewCVDRemoteCommand(opts).Execute()

	merr, _ := err.(*multierror.Error)
	errs := merr.WrappedErrors()
	if len(errs) != 2 {
		t.Errorf("expected 2, got: %d", len(errs))
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

func writeErr(w http.ResponseWriter, statusCode int) {
	write(w, &apiv1.Error{Code: strconv.Itoa(statusCode)}, statusCode)
}

func writeOK(w http.ResponseWriter, data interface{}) {
	write(w, data, http.StatusOK)
}

func write(w http.ResponseWriter, data interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(data)
}
