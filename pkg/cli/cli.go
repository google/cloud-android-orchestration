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
	hostFlag       = "host"
	serviceURLFlag = "service_url"
	zoneFlag       = "zone"
	httpProxyFlag  = "http_proxy"
)

type configFlags struct {
	ServiceURL string
	Zone       string
	HTTPProxy  string
}

// Extends a cobra.Command object with cvdremote specific operations like
// printing verbose logs
type command struct {
	*cobra.Command
	verbose *bool
}

func (c *command) PrintVerboseln(arg ...any) {
	if *c.verbose {
		c.PrintErrln(arg...)
	}
}

func (c *command) PrintVerbosef(format string, arg ...any) {
	if *c.verbose {
		c.PrintErrf(format, arg...)
	}
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
	rootCmd.AddCommand(newHostCommand(configFlags))
	rootCmd.AddCommand(newADBTunnelCommand(configFlags))
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

type subCommandFlags struct {
	*configFlags
	Verbose bool
}

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
	create.LocalFlags().StringVar(&createFlags.MachineType, gcpMachineTypeFlag, "n1-standard-4",
		"Indicates the machine type")
	create.LocalFlags().StringVar(&createFlags.MinCPUPlatform, gcpMinCPUPlatformFlag, "Intel Haswell",
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

func buildAPIClient(flags *subCommandFlags, c *cobra.Command) (*client.APIClient, error) {
	proxyURL := flags.HTTPProxy
	var dumpOut io.Writer = io.Discard
	if flags.Verbose {
		dumpOut = c.ErrOrStderr()
	}
	return client.NewAPIClient(buildBaseURL(flags.configFlags), proxyURL, dumpOut, c.ErrOrStderr())
}

func addCommonSubcommandFlags(c *cobra.Command, flags *subCommandFlags) {
	c.PersistentFlags().BoolVarP(&flags.Verbose, verboseFlag, "v", false, "Be verbose.")
}

func notImplementedCommand(c *cobra.Command, _ []string) error {
	return fmt.Errorf("Command not implemented")
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

func buildBaseURL(flags *configFlags) string {
	serviceURL := flags.ServiceURL
	zone := flags.Zone
	baseURL := serviceURL + "/v1"
	if zone != "" {
		baseURL += "/zones/" + zone
	}
	return baseURL
}
