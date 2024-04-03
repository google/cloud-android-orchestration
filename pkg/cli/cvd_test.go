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
