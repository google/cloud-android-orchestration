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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestListLocalImageRequiredFiles(t *testing.T) {
	tmpDir := t.TempDir()
	reqImgsFileDir := filepath.Join(tmpDir, path.Dir(RequiredImagesFilename))
	imagesFile := filepath.Join(tmpDir, RequiredImagesFilename)
	err := os.MkdirAll(reqImgsFileDir, 0750)
	if err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}
	err = os.WriteFile(imagesFile, []byte("foo\nbar\nbaz\n"), 0660)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ListLocalImageRequiredFiles(tmpDir, "/out/host/linux-x86")

	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"/out/host/linux-x86/foo",
		"/out/host/linux-x86/bar",
		"/out/host/linux-x86/baz",
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestAdditionalInstancesNum(t *testing.T) {
	tests := []struct {
		in int
		ex uint32
	}{
		{in: -1, ex: 0},
		{in: 0, ex: 0},
		{in: 1, ex: 0},
		{in: 100, ex: 99},
	}
	for _, tc := range tests {
		opts := &CreateCVDOpts{NumInstances: tc.in}

		got := opts.AdditionalInstancesNum()

		if diff := cmp.Diff(tc.ex, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestGetHostOutRelativePathSucceeds(t *testing.T) {
	targetArch := "x86_64"

	got, err := GetHostOutRelativePath(targetArch)

	if err != nil {
		t.Fatal(err)
	}
	expected := "out/host/linux-x86"
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetHostOutRelativePathFailsUnknownTargetArch(t *testing.T) {
	targetArch := "unknown"

	_, err := GetHostOutRelativePath(targetArch)

	if err == nil {
		t.Errorf("expected error")
	}
}

func TestEnvConfigValidity(t *testing.T) {
	cf, err := credentialsFactoryFromSource("none", "")
	if err != nil {
		t.Fatalf("couldn't make credentials factory")
	}

	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("couldn't create temp file")
	}
	fname := f.Name()
	f.Close()

	cc := cvdCreator{
		client:             fakeClient{},
		statePrinter:       newStatePrinter(io.Discard, false),
		credentialsFactory: cf,
		opts: CreateCVDOpts{
			Host: "foo",
		},
	}

	tests := []struct {
		name      string
		cfg       string
		expectErr bool
	}{
		{
			name:      "valid",
			cfg:       `{"common": {"host_package": "%s"}, "instances": [{"vm": {"cpus": 8}}]}`,
			expectErr: false,
		},
		{
			name:      "invalid proto field",
			cfg:       `{"common": {"host_package": "%s"}, "instances": [{"vm": {"cpus": "foobar"}}]}`,
			expectErr: true,
		},
		{
			name:      "additional fields",
			cfg:       `{"common": {"host_package": "%s"}, "instances": [{"vm": {"cpus": "foobar", "xx": 42}}]}`,
			expectErr: true,
		},
		{
			name:      "completely different",
			cfg:       `{"a": 1, "b": "%s"}`,
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := fmt.Sprintf(test.cfg, fname)
			ucfg := make(map[string]interface{})
			if err := json.Unmarshal([]byte(cfg), &ucfg); err != nil {
				t.Fatalf("config '%v' is not valid JSON: %s", cfg, err)
			}

			cc.opts.EnvConfig = ucfg
			_, err := cc.createWithCanonicalConfig()
			if err != nil && !test.expectErr {
				t.Errorf("unexpected error: got: %s", err)
			}
			if err != nil && test.expectErr {
				t.Logf("success: expected error, got: %s", err)
			}
			if err == nil && !test.expectErr {
				t.Logf("success: expected success")
			}
			if err == nil && test.expectErr {
				t.Errorf("unexpected success: config: %s", cfg)
			}
		})
	}
}
