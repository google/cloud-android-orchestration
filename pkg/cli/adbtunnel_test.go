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
	"testing"
)

func TestBuildAgentCmdline(t *testing.T) {
	/*****************************************************************
	If this test fails you most likely need to fix an AsArgs function!
	******************************************************************/
	// Don't name the fields to force a compiler error when the flag structures
	// are modified. This should help the developer realize they also need to
	// modify the corresponding AsArgs method.
	flags := ADBTunnelFlags{
		&CommonSubcmdFlags{
			&CVDRemoteFlags{
				"service url",
				"zone",
				"http proxy",
			},
			true, // verbose
		},
		"host",
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
	if subCmd.Name() != ADBTunnelAgentCommandName {
		t.Errorf("Expected it to parse %q command, found: %q", ADBTunnelAgentCommandName, subCmd.Name())
	}
	// TODO(jemoreira): Compare the parsed flags with used flags
}
