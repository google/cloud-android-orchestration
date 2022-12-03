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
	"testing"

	client "github.com/google/cloud-android-orchestration/pkg/client"

	"github.com/google/go-cmp/cmp"
)

func TestCVDOutput(t *testing.T) {
	output := CVDOutput{
		BaseURL: "http://foo.com",
		Host:    "bar",
		CVD: &client.CVD{
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
