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
	"time"

	client "github.com/google/cloud-android-orchestration/pkg/client"

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
	Args           []string
	FileConfig     FileConfig
	ServiceBuilder client.ServiceBuilder
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

type CVDRemoteFlags struct {
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
	flags := &CVDRemoteFlags{}
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
	rootCmd.PersistentFlags().StringVar(&flags.ServiceURL, serviceURLFlag,
		o.FileConfig.DefaultServiceURL, "Cloud orchestration service url.")
	if o.FileConfig.DefaultServiceURL == "" {
		// Make it required if not configured
		rootCmd.MarkPersistentFlagRequired(serviceURLFlag)
	}
	rootCmd.PersistentFlags().StringVar(&flags.Zone, zoneFlag, o.FileConfig.DefaultZone,
		"Cloud zone.")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPProxy, httpProxyFlag,
		o.FileConfig.DefaultHTTPProxy, "Proxy used to route the http communication through.")
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	subCmdOpts := &subCommandOpts{
		ServiceBuilder: buildServiceBuilder(o.ServiceBuilder),
		RootFlags:      flags,
		Config:         &o.FileConfig,
	}
	rootCmd.AddCommand(newHostCommand(subCmdOpts))
	rootCmd.AddCommand(newADBTunnelCommand(subCmdOpts))
	rootCmd.AddCommand(newCVDCommand(subCmdOpts))
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
)

type CommonSubcmdFlags struct {
	*CVDRemoteFlags
	Verbose bool
}

type serviceBuilder func(flags *CommonSubcmdFlags, c *cobra.Command) (client.Service, error)

type subCommandOpts struct {
	ServiceBuilder serviceBuilder
	RootFlags      *CVDRemoteFlags
	Config         *FileConfig
}

const chunkSizeBytes = 16 * 1024 * 1024

func buildServiceBuilder(builder client.ServiceBuilder) serviceBuilder {
	return func(flags *CommonSubcmdFlags, c *cobra.Command) (client.Service, error) {
		proxyURL := flags.HTTPProxy
		var dumpOut io.Writer = io.Discard
		if flags.Verbose {
			dumpOut = c.ErrOrStderr()
		}
		opts := &client.ServiceOptions{
			BaseURL:        buildBaseURL(flags.CVDRemoteFlags),
			ProxyURL:       proxyURL,
			DumpOut:        dumpOut,
			ErrOut:         c.ErrOrStderr(),
			RetryAttempts:  3,
			RetryDelay:     5 * time.Second,
			ChunkSizeBytes: chunkSizeBytes,
		}
		return builder(opts)
	}
}

func addCommonSubcommandFlags(c *cobra.Command, flags *CommonSubcmdFlags) {
	c.PersistentFlags().BoolVarP(&flags.Verbose, verboseFlag, "v", false, "Be verbose.")
}

func notImplementedCommand(c *cobra.Command, _ []string) error {
	return fmt.Errorf("Command not implemented")
}

func buildBaseURL(flags *CVDRemoteFlags) string {
	serviceURL := flags.ServiceURL
	zone := flags.Zone
	baseURL := serviceURL + "/v1"
	if zone != "" {
		baseURL += "/zones/" + zone
	}
	return baseURL
}
