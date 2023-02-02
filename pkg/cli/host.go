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
	"sync"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

const (
	gcpMachineTypeFlag    = "gcp_machine_type"
	gcpMinCPUPlatformFlag = "gcp_min_cpu_platform"
)

type HostFlags struct {
	*CommonSubcmdFlags
}

type CreateGCPHostFlags struct {
	*HostFlags
	MachineType    string
	MinCPUPlatform string
}

func newHostCommand(opts *subCommandOpts) *cobra.Command {
	commonSCFlags := &CommonSubcmdFlags{CVDRemoteFlags: opts.RootFlags}
	hostFlags := &HostFlags{CommonSubcmdFlags: commonSCFlags}
	createFlags := &CreateGCPHostFlags{HostFlags: hostFlags}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return createHost(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.MachineType, gcpMachineTypeFlag,
		opts.InitialConfig.Host.GCP.MachineType, "Indicates the machine type")
	create.Flags().StringVar(&createFlags.MinCPUPlatform, gcpMinCPUPlatformFlag,
		opts.InitialConfig.Host.GCP.MinCPUPlatform,
		"Specifies a minimum CPU platform for the VM instance")
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
	addCommonSubcommandFlags(host, commonSCFlags)
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func createHost(c *cobra.Command, flags *CreateGCPHostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
	if err != nil {
		return err
	}
	req := apiv1.CreateHostRequest{
		HostInstance: &apiv1.HostInstance{
			GCP: &apiv1.GCPInstance{
				MachineType:    flags.MachineType,
				MinCPUPlatform: flags.MinCPUPlatform,
			},
		},
	}
	ins, err := apiClient.CreateHost(&req)
	if err != nil {
		return fmt.Errorf("Error creating host: %w", err)
	}
	c.Printf("%s\n", ins.Name)
	return nil
}

func listHosts(c *cobra.Command, flags *HostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
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

func deleteHosts(c *cobra.Command, args []string, flags *HostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var merr error
	for _, arg := range args {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := apiClient.DeleteHost(name); err != nil {
				mu.Lock()
				defer mu.Unlock()
				merr = multierror.Append(merr, fmt.Errorf("delete host %q failed: %w", name, err))
			}
		}(arg)
	}
	wg.Wait()
	return merr
}
