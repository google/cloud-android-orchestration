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
	"reflect"
	"strings"
	"testing"
)

const (
	validConfig = `
ServiceURL = "service_url"
Zone = "zone"
HTTPProxy = "http_proxy"
[Host.GCP]
MachineType = "machine_type"
MinCPUPlatform = "cpu_platform"
`
	invalidConfig = "foo_bar_baz = \"unknown field\""
)

func TestParseConfigFile(t *testing.T) {
	// This test is little more than a change detector, however it's still
	// useful to detect config parsing errors early, such as those introduced by
	// a rename of the config properties.
	config := DefaultConfig()
	err := ParseConfigFile(&config, strings.NewReader(validConfig))
	if err != nil {
		t.Errorf("Failed to parse config: %v", err)
	}
	// It's not necessary to check the value of each property because a successful
	// parsing of the valid config above and a failed parsing of the config with
	// unknown properties below guarantee that all properties of the valid config
	// were parsed. The parsing library doesn't need to be tested here.
}

func TestParseAllConfig(t *testing.T) {
	config := DefaultConfig()
	err := ParseConfigFile(&config, strings.NewReader(validConfig))
	if err != nil {
		t.SkipNow()
	}
	// Ensure no field was left unparsed. This will fail everytime a new field is
	// added to cli.Config, just add it to the valid config above with a non zero
	// value to make it pass and ensure these tests apply to that field in the
	// future.
	if has, f := HasZeroes(config); has {
		t.Errorf("The Config's %s field was not parsed", f)
	}
}

func TestParseInvalidConfig(t *testing.T) {
	config := DefaultConfig()
	err := ParseConfigFile(&config, strings.NewReader(invalidConfig))
	if err == nil {
		t.Errorf("Expected unknown config property to produce an error")
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
