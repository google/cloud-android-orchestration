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
	"fmt"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/client"

	"github.com/spf13/cobra"
)

const (
	gcpMachineTypeFlag    = "gcp_machine_type"
	gcpMinCPUPlatformFlag = "gcp_min_cpu_platform"
)

const (
	gcpMachineTypeFlagDesc    = "Indicates the machine type"
	gcpMinCPUPlatformFlagDesc = "Specifies a minimum CPU platform for the VM instance"
)

type CreateHostOpts struct {
	GCP CreateGCPHostOpts
}

type CreateGCPHostOpts struct {
	MachineType    string
	MinCPUPlatform string
}

type CreateHostFlags struct {
	*CommonSubcmdFlags
	*CreateHostOpts
}

func newHostCommand(opts *subCommandOpts) *cobra.Command {
	hostFlags := &CommonSubcmdFlags{CVDRemoteFlags: opts.RootFlags}
	createFlags := &CreateHostFlags{CommonSubcmdFlags: hostFlags, CreateHostOpts: &CreateHostOpts{}}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateHostCommand(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.GCP.MachineType, gcpMachineTypeFlag,
		opts.InitialConfig.Host.GCP.MachineType, gcpMachineTypeFlagDesc)
	create.Flags().StringVar(&createFlags.GCP.MinCPUPlatform, gcpMinCPUPlatformFlag,
		opts.InitialConfig.Host.GCP.MinCPUPlatform, gcpMinCPUPlatformFlagDesc)
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return listHosts(c, hostFlags, opts)
		},
	}
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return deleteHosts(c, args, hostFlags, opts)
		},
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	addCommonSubcommandFlags(host, hostFlags)
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func runCreateHostCommand(c *cobra.Command, flags *CreateHostFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
	if err != nil {
		return fmt.Errorf("Failed to build service instance: %w", err)

	}
	ins, err := createHost(service, *flags.CreateHostOpts)
	if err != nil {
		return fmt.Errorf("Failed to create host: %w", err)
	}
	c.Printf("%s\n", ins.Name)
	return nil
}

func createHost(service client.Service, opts CreateHostOpts) (*apiv1.HostInstance, error) {
	req := apiv1.CreateHostRequest{
		HostInstance: &apiv1.HostInstance{
			GCP: &apiv1.GCPInstance{
				MachineType:    opts.GCP.MachineType,
				MinCPUPlatform: opts.GCP.MinCPUPlatform,
			},
		},
	}
	return service.CreateHost(&req)
}

func listHosts(c *cobra.Command, flags *CommonSubcmdFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags, c)
	if err != nil {
		return err
	}
	hosts, err := apiClient.ListHosts()
	if err != nil {
		return fmt.Errorf("Error listing hosts: %w", err)
	}
	for _, ins := range hosts.Items {
		c.Printf("%s\n", ins.Name)
	}
	return nil
}

func deleteHosts(c *cobra.Command, args []string, flags *CommonSubcmdFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags, c)
	if err != nil {
		return err
	}
	return service.DeleteHosts(args)
}
