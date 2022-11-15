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
	"io"
	"sync"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	client "github.com/google/cloud-android-orchestration/pkg/client"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

// Groups streams for standard IO.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type CommandOptions struct {
	IOStreams
	Args []string
}

type CVDRemoteCommand struct {
	command *cobra.Command
}

const (
	serviceURLFlag = "service_url"
	zoneFlag       = "zone"
	httpProxyFlag  = "http_proxy"
)

type configFlags struct {
	ServiceURL string
	Zone       string
	HTTPProxy  string
}

func NewCVDRemoteCommand(o *CommandOptions) *CVDRemoteCommand {
	configFlags := &configFlags{}
	rootCmd := &cobra.Command{
		Use:               "cvdremote",
		Short:             "Manages Cuttlefish Virtual Devices (CVDs) in the cloud.",
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	rootCmd.SetArgs(o.Args)
	rootCmd.SetOut(o.IOStreams.Out)
	rootCmd.SetErr(o.IOStreams.ErrOut)
	rootCmd.PersistentFlags().StringVar(&configFlags.ServiceURL, serviceURLFlag, "",
		"Cloud orchestration service url.")
	rootCmd.MarkPersistentFlagRequired(serviceURLFlag)
	rootCmd.PersistentFlags().StringVar(&configFlags.Zone, zoneFlag, "", "Cloud zone.")
	rootCmd.PersistentFlags().StringVar(&configFlags.HTTPProxy, httpProxyFlag, "",
		"Proxy used to route the http communication through.")
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.AddCommand(newHostCommand())
	return &CVDRemoteCommand{rootCmd}
}

func (c *CVDRemoteCommand) Execute() error {
	err := c.command.Execute()
	if err != nil {
		c.command.PrintErrln(err)
	}
	return err
}

const (
	verboseFlag = "verbose"

	gcpMachineTypeFlag    = "gcp_machine_type"
	gcpMinCPUPlatformFlag = "gcp_min_cpu_platform"
)

type createGCPHostFlags struct {
	MachineType    string
	MinCPUPlatform string
}

type subCommandFlags struct {
	createGCPHostFlags

	Verbose bool
}

func newHostCommand() *cobra.Command {
	flags := &subCommandFlags{}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE:  runCreateHostCommand,
	}
	create.LocalFlags().StringVar(&flags.MachineType, gcpMachineTypeFlag, "n1-standard-4",
		"Indicates the machine type")
	create.LocalFlags().StringVar(&flags.MinCPUPlatform, gcpMinCPUPlatformFlag, "Intel Haswell",
		"Specifies a minimum CPU platform for the VM instance")
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists hosts.",
		RunE:  runListHostsCommand,
	}
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE:  runDeleteHostsCommand,
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	host.PersistentFlags().BoolVarP(&flags.Verbose, verboseFlag, "v", false, "Be verbose.")
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func buildAPIClient(c *cobra.Command) (*client.APIClient, error) {
	proxyURL := c.InheritedFlags().Lookup(httpProxyFlag).Value.String()
	verbose := c.InheritedFlags().Lookup(verboseFlag).Changed
	var dumpOut io.Writer = io.Discard
	if verbose {
		dumpOut = c.ErrOrStderr()
	}
	return client.NewAPIClient(buildBaseURL(c), proxyURL, dumpOut, c.ErrOrStderr())
}

func notImplementedCommand(c *cobra.Command, _ []string) error {
	return fmt.Errorf("Command not implemented")
}

func runCreateHostCommand(c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(c)
	if err != nil {
		return err
	}
	req := apiv1.CreateHostRequest{
		HostInstance: &apiv1.HostInstance{
			GCP: &apiv1.GCPInstance{
				MachineType:    c.LocalFlags().Lookup(gcpMachineTypeFlag).Value.String(),
				MinCPUPlatform: c.LocalFlags().Lookup(gcpMinCPUPlatformFlag).Value.String(),
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

func runListHostsCommand(c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(c)
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

func runDeleteHostsCommand(c *cobra.Command, args []string) error {
	apiClient, err := buildAPIClient(c)
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

func buildBaseURL(c *cobra.Command) string {
	serviceURL := c.InheritedFlags().Lookup(serviceURLFlag).Value.String()
	zone := c.InheritedFlags().Lookup(zoneFlag).Value.String()
	baseURL := serviceURL + "/v1"
	if zone != "" {
		baseURL += "/zones/" + zone
	}
	return baseURL
}
