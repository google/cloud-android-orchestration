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
	"flag"
	"fmt"
)

type Command interface {
	Run([]string) error
}

// Groups the subcommands handled in cvdremote.
type CVDRemoteSubCommands struct {
	CreateCVD   Command
	ListCVDs    Command
	ListHosts   Command
	CreateHost  Command
	DeleteHosts Command
}

const cvdremoteUsage = `NAME
    cvdremote - manage Cuttlefish Virtual Devices (CVDs) in the cloud.

SYNOPSIS
    cvdremote [-h | -help] [<resource>] <command> [<args>]

RESOURCES
    cvd (default)
        Cuttlefish Virtual Devices.

    host
        Host machines where CVDs live.

COMMANDS
    create
        Create a resource.

    list
        List the resources.`

type CVDRemoteCommand struct {
	fs             *flag.FlagSet
	subCommandsMap map[string]map[string]Command
}

func NewCVDRemoteCommand() Command {
	subCommands := CVDRemoteSubCommands{
		CreateCVD:   &notImplementedCommand{},
		ListCVDs:    &notImplementedCommand{},
		CreateHost:  &notImplementedCommand{},
		ListHosts:   &notImplementedCommand{},
		DeleteHosts: &notImplementedCommand{},
	}
	return NewCVDRemoteCommandWithArgs(subCommands)
}

const (
	resourceHost = "host"
	resourceCVD  = "cvd"

	commandList   = "list"
	commandCreate = "create"
	commandDelete = "delete"
)

func NewCVDRemoteCommandWithArgs(subCommands CVDRemoteSubCommands) Command {
	c := &CVDRemoteCommand{
		fs: flag.NewFlagSet("cvdremote", flag.ContinueOnError),
		subCommandsMap: map[string]map[string]Command{
			resourceCVD: {
				commandList:   subCommands.ListCVDs,
				commandCreate: subCommands.CreateCVD,
			},
			resourceHost: {
				commandList:   subCommands.ListHosts,
				commandCreate: subCommands.CreateHost,
				commandDelete: subCommands.DeleteHosts,
			},
		},
	}
	c.fs.Usage = func() {
		fmt.Println(cvdremoteUsage)
	}
	return c
}

func (c *CVDRemoteCommand) Run(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}
	args = c.fs.Args()
	if len(args) == 0 {
		return fmt.Errorf("missing resource")
	}
	// If no resource is given it defaults to "cvd".
	// There is no chance of a collision between a cvd subcommand and a resource name because
	// resources are nouns and subcommands are verbs.
	resource := resourceCVD
	if _, ok := c.subCommandsMap[args[0]]; ok {
		resource = args[0]
		args = args[1:]
	}
	if len(args) == 0 {
		return fmt.Errorf("missing resource's command")
	}
	command, ok := c.subCommandsMap[resource][args[0]]
	if !ok {
		return fmt.Errorf("invalid command %q for resource %q", args[0], resource)
	}
	return command.Run(args[1:])
}

type notImplementedCommand struct{}

func (c *notImplementedCommand) Run(args []string) error {
	return fmt.Errorf("not implemented")
}
