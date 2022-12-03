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

type createGCPHostFlags struct {
	*subCommandFlags
	MachineType    string
	MinCPUPlatform string
}

func newHostCommand(cfgFlags *configFlags) *cobra.Command {
	hostFlags := &subCommandFlags{
		cfgFlags,
		false,
	}
	createFlags := &createGCPHostFlags{hostFlags, "", ""}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateHostCommand(createFlags, c, args)
		},
	}
	create.Flags().StringVar(&createFlags.MachineType, gcpMachineTypeFlag, "n1-standard-4",
		"Indicates the machine type")
	create.Flags().StringVar(&createFlags.MinCPUPlatform, gcpMinCPUPlatformFlag, "Intel Haswell",
		"Specifies a minimum CPU platform for the VM instance")
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runListHostsCommand(hostFlags, c, args)
		},
	}
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runDeleteHostsCommand(hostFlags, c, args)
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

func runCreateHostCommand(flags *createGCPHostFlags, c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(flags.subCommandFlags, c)
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
		return err
	}
	c.Printf("%s\n", ins.Name)
	return nil
}

func runListHostsCommand(flags *subCommandFlags, c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(flags, c)
	if err != nil {
		return err
	}
	hosts, err := apiClient.ListHosts()
	if err != nil {
		return err
	}
	for _, ins := range hosts.Items {
		c.Printf("%s\n", ins.Name)
	}
	return nil
}

func runDeleteHostsCommand(flags *subCommandFlags, c *cobra.Command, args []string) error {
	apiClient, err := buildAPIClient(flags, c)
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
