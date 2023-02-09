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
	"testing"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	"github.com/google/go-cmp/cmp"
)

func TestCVDOutput(t *testing.T) {
	output := CVDOutput{
		ServiceRootEndpoint: "http://foo.com",
		Host:                "bar",
		CVD: &hoapi.CVD{
			Name:     "cvd-1",
			Status:   "Running",
			Displays: []string{"720 x 1280 ( 320 )"},
		},
	}

	got := output.String()

	expected := `cvd-1 (bar)
  Status: Running
  Displays: [720 x 1280 ( 320 )]
  WebRTCStream: http://foo.com/hosts/bar/devices/cvd-1/files/client.html
  Logs: http://foo.com/hosts/bar/cvds/cvd-1/logs/`

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestGetAndroidEnvVarValuesMissingVariable(t *testing.T) {
	_, err := GetAndroidEnvVarValues()

	if _, ok := err.(MissingEnvVarErr); !ok {
		t.Errorf("expected %+v, got %+v", MissingEnvVarErr(""), err)
	}

}

func TestGetAndroidEnvVarValues(t *testing.T) {
	t.Setenv(AndroidBuildTopVarName, "foo")
	t.Setenv(AndroidHostOutVarName, "bar")
	t.Setenv(AndroidProductOutVarName, "baz")

	got, _ := GetAndroidEnvVarValues()

	expected := AndroidEnvVars{
		BuildTop:   "foo",
		HostOut:    "bar",
		ProductOut: "baz",
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestListLocalImageRequiredFiles(t *testing.T) {
	tmpDir := t.TempDir()
	reqImgsFileDir := tmpDir + "/" + path.Dir(RequiredImagesFilename)
	imagesFile := tmpDir + "/" + RequiredImagesFilename
	err := os.MkdirAll(reqImgsFileDir, 0750)
	if err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}
	err = os.WriteFile(imagesFile, []byte("foo\nbar\nbaz\n"), 0660)
	if err != nil {
		t.Fatal(err)
	}
	vars := AndroidEnvVars{
		BuildTop:   tmpDir,
		HostOut:    "/product/vsoc_x86_64",
		ProductOut: "/out/host/linux-x86",
	}

	got, err := ListLocalImageRequiredFiles(vars)

	expected := []string{
		"/out/host/linux-x86/foo",
		"/out/host/linux-x86/bar",
		"/out/host/linux-x86/baz",
		"/product/vsoc_x86_64/cvd-host_package.tar.gz",
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

}
