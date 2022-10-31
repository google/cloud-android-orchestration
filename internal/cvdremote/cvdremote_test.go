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

package cvdremote

import (
	"reflect"
	"testing"
)

type testCommand struct {
	args []string
}

func (c *testCommand) Run(args []string) error {
	c.args = args
	return nil
}

func TestCVDRemoteCommandSubCommandsRouting(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		subCommands CVDRemoteSubCommands
	}{
		{
			name:        "test `cvd` is the default resource",
			args:        []string{"list"},
			subCommands: CVDRemoteSubCommands{ListCVDs: &testCommand{}},
		},
		{
			name:        "test `cvd list` subcommand",
			args:        []string{"cvd", "list"},
			subCommands: CVDRemoteSubCommands{ListCVDs: &testCommand{}},
		},
		{
			name:        "test `cvd create` subcommand",
			args:        []string{"cvd", "create"},
			subCommands: CVDRemoteSubCommands{CreateCVD: &testCommand{}},
		},
		{
			name:        "test `host list` subcommand",
			args:        []string{"host", "list"},
			subCommands: CVDRemoteSubCommands{ListHosts: &testCommand{}},
		},
		{
			name:        "test `host create` subcommand",
			args:        []string{"host", "create"},
			subCommands: CVDRemoteSubCommands{CreateHost: &testCommand{}},
		},
		{
			name:        "test `host delete` subcommand",
			args:        []string{"host", "delete"},
			subCommands: CVDRemoteSubCommands{DeleteHosts: &testCommand{}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := NewCVDRemoteCommand(test.subCommands)

			err := command.Run(test.args)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCVDRemoteCommandSubCommandsExpectedArgs(t *testing.T) {
	listCVDsCommand := &testCommand{}
	command := NewCVDRemoteCommand(CVDRemoteSubCommands{ListCVDs: listCVDsCommand})

	err := command.Run([]string{"list", "foo", "bar"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"foo", "bar"}
	if !reflect.DeepEqual(listCVDsCommand.args, expected) {
		t.Fatalf("expected %+v, got %+v", expected, listCVDsCommand.args)
	}
}
