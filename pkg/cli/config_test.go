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
SystemDefaultService = "foo"

[Services."foo"]
ServiceURL = "service_url"
Proxy = "proxy"
Host = { GCP = { MinCPUPlatform = "cpu_platform" } }
`
	fname := tempFile(t, config)
	c := BaseConfig()
	// This test is little more than a change detector, however it's still
	// useful to detect config parsing errors early, such as those introduced by
	// a rename of the config properties.
	err := LoadConfig(fname, "", c)

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
SystemDefaultService = "foo"
UserDefaultService = "bar"

[Services."foo"]
ServiceURL = "service_url"
Zone = "zone"
Proxy = "proxy"
BuildAPICredentialsSource = "injected"
Host = {
  GCP = {
    MachineType = "machine_type",
    MinCPUPlatform = "cpu_platform",
    BootDiskSizeGB = 10
  }
}
Authn = {
  OIDCToken = {
    TokenFile = "/path/to/token"
  }
}
`
	fname := tempFile(t, fullConfig)
	c := BaseConfig()

	err := LoadConfig(fname, "", c)
	if err != nil {
		t.Fatal(err)
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

	err := LoadConfig(fname, "", c)

	if err == nil {
		t.Errorf("expected unknown config property to produce an error")
	}
}

func TestLoadConfigMultiFiles(t *testing.T) {
	const system = `
SystemDefaultService = "foo"

[Services."foo"]
ServiceURL = "foo.com"
`
	const user = `
UserDefaultService = "bar"

[Services."bar"]
ServiceURL = "bar.com"
`
	scf := tempFile(t, system)
	ucf := tempFile(t, user)
	expected := &Config{
		SystemDefaultService: "foo",
		UserDefaultService:   "bar",
		ConnectionControlDir: "~/.cvdr/connections",
		KeepLogFilesDays:     30,
		Services: map[string]*Service{
			"foo": {
				ServiceURL: "foo.com",
			},
			"bar": {
				ServiceURL: "bar.com",
			},
		},
	}
	c := BaseConfig()

	err := LoadConfig(scf, ucf, c)

	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, c); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadConfigMultiFilesNoUserService(t *testing.T) {
	const system = `
SystemDefaultService = "foo"

[Services."foo"]
ServiceURL = "foo.com"

[Services."bar"]
ServiceURL = "bar.com"
`
	const user = `
UserDefaultService = "bar"
`
	scf := tempFile(t, system)
	ucf := tempFile(t, user)
	expected := &Config{
		SystemDefaultService: "foo",
		UserDefaultService:   "bar",
		ConnectionControlDir: "~/.cvdr/connections",
		KeepLogFilesDays:     30,
		Services: map[string]*Service{
			"foo": {
				ServiceURL: "foo.com",
			},
			"bar": {
				ServiceURL: "bar.com",
			},
		},
	}
	c := BaseConfig()

	err := LoadConfig(scf, ucf, c)

	if err != nil {
		t.Fatal(err)
	}
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
		Services: map[string]*Service{
			"acloud": {
				Zone: "foo-central1-c",
				Host: HostConfig{
					GCP: GCPHostConfig{
						MachineType: "foo-standard-4",
					},
				},
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
		if err := LoadConfig(dst, "", c); err != nil {
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
	if value.Kind() != reflect.Struct && value.Kind() != reflect.Map {
		return value.IsZero(), ""
	}
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			if has, name := HasZeroes(value.Field(i).Interface()); has {
				fullName := value.Type().Field(i).Name
				if name != "" {
					fullName += "." + name
				}
				return true, fullName
			}
		}
	} else if value.Kind() == reflect.Map {
		return HasZeroes(reflect.Indirect(value.MapIndex(value.MapKeys()[0])).Interface())
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
