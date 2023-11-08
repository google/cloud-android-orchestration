// Copyright 2023 Google LLC
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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfigFile(t *testing.T) {
	const config = `
ServiceURL = "service_url"
HTTPProxy = "http_proxy"
[Host.GCP]
MinCPUPlatform = "cpu_platform"
`
	fname := tempFile(t, config)
	c := BaseConfig()
	// This test is little more than a change detector, however it's still
	// useful to detect config parsing errors early, such as those introduced by
	// a rename of the config properties.
	err := LoadConfigFile(fname, c)

	if err != nil {
		t.Errorf("failed to parse config: %v", err)
	}
	// It's not necessary to check the value of each property because a successful
	// parsing of the valid config above and a failed parsing of the config with
	// unknown properties below guarantee that all properties of the valid config
	// were parsed. The parsing library doesn't need to be tested here.
}

func TestLoadFullConfig(t *testing.T) {
	const fullConfig = `
ServiceURL = "service_url"
Zone = "zone"
HTTPProxy = "http_proxy"
[Host.GCP]
MachineType = "machine_type"
MinCPUPlatform = "cpu_platform"
`
	fname := tempFile(t, fullConfig)
	c := BaseConfig()

	err := LoadConfigFile(fname, c)
	if err != nil {
		t.SkipNow()
	}
	// Ensure no field was left unparsed. This will fail everytime a new field is
	// added to cli.Config, just add it to the valid config above with a non zero
	// value to make it pass and ensure these tests apply to that field in the
	// future.
	if has, f := HasZeroes(*c); has {
		t.Errorf("the Config's %s field was not parsed", f)
	}
}

func TestLoadConfigFileInvalidConfig(t *testing.T) {
	const config = "foo_bar_baz  \"unknown field\""
	fname := tempFile(t, config)
	c := BaseConfig()

	err := LoadConfigFile(fname, c)

	if err == nil {
		t.Errorf("expected unknown config property to produce an error")
	}
}

func TestLoadConfigFileTwice(t *testing.T) {
	const system = `
ServiceURL = "service-foo"
Zone = "zone-foo"
KeepLogFilesDays = 30
[Host.GCP]
MinCPUPlatform = "min-cpu-platform-foo"
`
	const user = `
Zone = "zone-bar"
KeepLogFilesDays = 0
[Host.GCP]
MachineType = "machine-type-bar"
`
	scf := tempFile(t, system)
	ucf := tempFile(t, user)
	expected := &Config{
		ServiceURL:           "service-foo",
		Zone:                 "zone-bar",
		KeepLogFilesDays:     0,
		ConnectionControlDir: "~/.cvdr/connections",
		Host: HostConfig{
			GCP: GCPHostConfig{
				MachineType:    "machine-type-bar",
				MinCPUPlatform: "min-cpu-platform-foo",
			},
		},
	}
	c := BaseConfig()

	_ = LoadConfigFile(scf, c)
	_ = LoadConfigFile(ucf, c)

	if diff := cmp.Diff(expected, c); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestImportAcloudConfig(t *testing.T) {
	tests := []struct {
		content string
	}{
		{
			content: `project: "foo" # project zero
# Zone comment
zone: "foo-central1-c" # zone: "invalid"
# machine_type: "comment"
machine_type: "foo-standard-4"
network: "default"

# Cuttlefish host image
# stable_host_image_family: "acloud-release"
stable_host_image_name: "cuttlefish-google-vsoc-0-9-27-160g"
machine_type: "foo-standard-4"
`,
		},
		{
			content: `zone: "foo-central1-c" # zone: "invalid"
machine_type: "foo-standard-4"
`,
		},
		{
			content: `#
zone	:		"foo-central1-c"
machine_type:"foo-standard-4"
`,
		},
		{
			content: `#
	zone	:		"foo-central1-c"
machine_type: "foo-standard-4"
`,
		},
	}
	expected := &Config{
		Zone: "foo-central1-c",
		Host: HostConfig{
			GCP: GCPHostConfig{
				MachineType: "foo-standard-4",
			},
		},
	}

	for _, tc := range tests {
		source := tempFile(t, tc.content)
		dstDir := t.TempDir()
		dst := path.Join(dstDir, "config")

		err := ImportAcloudConfig(source, dst)

		if err != nil {
			t.Fatal(err)
		}
		c := &Config{}
		if err := LoadConfigFile(dst, c); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expected, c); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}
	}

}

func TestImportAcloudConfigInvalidConfig(t *testing.T) {
	const acloudConfig = `project: "foo" # project zero
#zone: "foo-central1-c" # zone: "invalid"
# machine_type: "comment"
machine_type: "foo-standard-4"
`
	source := tempFile(t, acloudConfig)
	dstDir := t.TempDir()
	dst := path.Join(dstDir, "config")

	err := ImportAcloudConfig(source, dst)

	if err == nil {
		t.Errorf("expected error")
	}
}

// Returns true if the argument or any field at any nest level is zero-valued.
// It also returns the name of the first zero-valued field it finds.
func HasZeroes(o any) (bool, string) {
	value := reflect.ValueOf(o)
	if value.Kind() != reflect.Struct {
		return value.IsZero(), ""
	}
	for i := 0; i < value.NumField(); i++ {
		if has, name := HasZeroes(value.Field(i).Interface()); has {
			fullName := value.Type().Field(i).Name
			if name != "" {
				fullName += "." + name
			}
			return true, fullName
		}
	}
	return false, ""
}

func tempFile(t *testing.T, content string) string {
	dir := t.TempDir()
	fname := path.Join(dir, "cvdr.config")
	err := os.WriteFile(fname, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return fname
}
