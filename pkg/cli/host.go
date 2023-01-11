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

type hostFlags struct {
	*subCommandFlags
}

type createGCPHostFlags struct {
	*hostFlags
	MachineType    string
	MinCPUPlatform string
}

func newHostCommand(configFlags *configFlags, config *HostConfig, opts *subCommandOpts) *cobra.Command {
	subCommandFlags := &subCommandFlags{configFlags: configFlags}
	hostFlags := &hostFlags{subCommandFlags: subCommandFlags}
	createFlags := &createGCPHostFlags{hostFlags: hostFlags}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateHostCommand(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.MachineType, gcpMachineTypeFlag,
		config.GCP.DefaultMachineType, "Indicates the machine type")
	create.Flags().StringVar(&createFlags.MinCPUPlatform, gcpMinCPUPlatformFlag,
		config.GCP.DefaultMinCPUPlatform,
		"Specifies a minimum CPU platform for the VM instance")
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runListHostsCommand(c, hostFlags, opts)
		},
	}
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runDeleteHostsCommand(c, args, hostFlags, opts)
		},
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	addCommonSubcommandFlags(host, subCommandFlags)
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func runCreateHostCommand(c *cobra.Command, flags *createGCPHostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.subCommandFlags, c)
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

func runListHostsCommand(c *cobra.Command, flags *hostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.subCommandFlags, c)
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

func runDeleteHostsCommand(c *cobra.Command, args []string, flags *hostFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags.subCommandFlags, c)
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
