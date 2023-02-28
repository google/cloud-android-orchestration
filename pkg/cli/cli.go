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
	"os"
	"time"

	"github.com/google/cloud-android-orchestration/pkg/client"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

// Groups streams for standard IO.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type CommandRunner interface {
	// Start a command and doesn't wait for it to exit. Instead it reads its entire
	// standard output and returns that or an error. The commands stdin and stderr
	// should be connected to sensible IO channels.
	StartBgCommand(...string) (string, error)
	// When needed RunCommand can be added, returning the exit code, the output and
	// an error.
}

type CommandOptions struct {
	IOStreams
	Args           []string
	InitialConfig  Config
	ServiceBuilder client.ServiceBuilder
	CommandRunner  CommandRunner
}

type CVDRemoteCommand struct {
	command *cobra.Command
}

const (
	hostFlag       = "host"
	serviceURLFlag = "service_url"
	zoneFlag       = "zone"
	httpProxyFlag  = "http_proxy"
	verboseFlag    = "verbose"
)

const (
	gcpMachineTypeFlag    = "gcp_machine_type"
	gcpMinCPUPlatformFlag = "gcp_min_cpu_platform"
)

const (
	gcpMachineTypeFlagDesc    = "Indicates the machine type"
	gcpMinCPUPlatformFlagDesc = "Specifies a minimum CPU platform for the VM instance"
)

const (
	branchFlag     = "branch"
	buildIDFlag    = "build_id"
	targetFlag     = "target"
	localImageFlag = "local_image"
)

const (
	ADBTunnelCommandName      = "adbtunnel"
	ADBTunnelOpenCommandName  = "open"
	ADBTunnelCloseCommandName = "close"
	ADBTunnelListCommandName  = "list"
	ADBTunnelAgentCommandName = "agent"
)

type AsArgs interface {
	AsArgs() []string
}

type CVDRemoteFlags struct {
	ServiceURL string
	Zone       string
	HTTPProxy  string
	Verbose    bool
}

func (f *CVDRemoteFlags) AsArgs() []string {
	args := []string{
		"--" + serviceURLFlag, f.ServiceURL,
		"--" + zoneFlag, f.Zone,
	}
	if f.HTTPProxy != "" {
		args = append(args, "--"+httpProxyFlag, f.HTTPProxy)
	}
	if f.Verbose {
		args = append(args, "-v")
	}
	return args
}

type CreateHostFlags struct {
	*CVDRemoteFlags
	*CreateHostOpts
}

type CreateCVDFlags struct {
	*CVDRemoteFlags
	*CreateCVDOpts
	*CreateHostOpts
}

type ListCVDsFlags struct {
	*CVDRemoteFlags
	Host string
}

type subCommandOpts struct {
	ServiceBuilder serviceBuilder
	RootFlags      *CVDRemoteFlags
	InitialConfig  Config
	CommandRunner  CommandRunner
}

type ADBTunnelFlags struct {
	*CVDRemoteFlags
	host string
}

func (f *ADBTunnelFlags) AsArgs() []string {
	args := f.CVDRemoteFlags.AsArgs()
	if f.host != "" {
		args = append(args, "--"+hostFlag, f.host)
	}
	return args
}

type listADBTunnelFlags struct {
	*ADBTunnelFlags
	longFormat bool
}

type closeADBTunnelFlags struct {
	*ADBTunnelFlags
	skipConfirmation bool
}

// Extends a cobra.Command object with cvdr specific operations like
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

func (c *command) Parent() *command {
	p := c.Command.Parent()
	if p == nil {
		return nil
	}
	return &command{p, c.verbose}
}

type CVDOutput struct {
	*CVDInfo
	ADBPort int
}

func (o *CVDOutput) String() string {
	res := fmt.Sprintf("%s (%s)", o.Name, o.Host)
	res += "\n  " + "Status: " + o.Status
	adbState := ""
	if o.ADBPort > 0 {
		adbState = fmt.Sprintf("127.0.0.1:%d", o.ADBPort)
	} else {
		adbState = "disconnected"
	}
	res += "\n  " + "ADB: " + adbState
	res += "\n  " + "Displays: " + fmt.Sprintf("%v", o.Displays)
	res += "\n  " + "WebRTCStream: " + client.BuildWebRTCStreamURL(o.ServiceRootEndpoint, o.Host, o.Name)
	res += "\n  " + "Logs: " + client.BuildCVDLogsURL(o.ServiceRootEndpoint, o.Host, o.Name)
	return res
}

type SelectionOption int32

const (
	AllowAll SelectionOption = 1 << iota
)

func (c *command) PromptSelection(choices []string, selOpt SelectionOption) ([]int, error) {
	for i, v := range choices {
		c.PrintErrf("%d) %s\n", i, v)
	}
	maxChoice := len(choices) - 1
	if selOpt&AllowAll != 0 {
		c.PrintErrf("%d) All\n", len(choices))
		maxChoice = len(choices)
	}
	c.PrintErrf("Choose an option: ")
	chosen := -1
	_, err := fmt.Fscanln(c.InOrStdin(), &chosen)
	if err != nil {
		return nil, fmt.Errorf("Failed to read choice: %w", err)
	}
	if chosen < 0 || chosen > maxChoice {
		return nil, fmt.Errorf("Choice out of range: %d", chosen)
	}
	if chosen < len(choices) {
		return []int{chosen}, nil
	}
	ret := make([]int, len(choices))
	for i := range choices {
		ret[i] = i
	}
	return ret, nil
}

func NewCVDRemoteCommand(o *CommandOptions) *CVDRemoteCommand {
	flags := &CVDRemoteFlags{}
	rootCmd := &cobra.Command{
		Use:               "cvdr",
		Short:             "Manages Cuttlefish Virtual Devices (CVDs) in the cloud.",
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	rootCmd.SetArgs(o.Args)
	rootCmd.SetOut(o.IOStreams.Out)
	rootCmd.SetErr(o.IOStreams.ErrOut)
	rootCmd.PersistentFlags().StringVar(&flags.ServiceURL, serviceURLFlag, o.InitialConfig.ServiceURL,
		"Cloud orchestration service url.")
	if o.InitialConfig.ServiceURL == "" {
		// Make it required if not configured
		rootCmd.MarkPersistentFlagRequired(serviceURLFlag)
	}
	rootCmd.PersistentFlags().StringVar(&flags.Zone, zoneFlag, o.InitialConfig.Zone, "Cloud zone.")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPProxy, httpProxyFlag, o.InitialConfig.HTTPProxy,
		"Proxy used to route the http communication through.")
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, verboseFlag, "v", false, "Be verbose.")
	subCmdOpts := &subCommandOpts{
		ServiceBuilder: buildServiceBuilder(o.ServiceBuilder),
		RootFlags:      flags,
		InitialConfig:  o.InitialConfig,
		CommandRunner:  o.CommandRunner,
	}
	cvdGroup := &cobra.Group{
		ID:    "cvd",
		Title: "Commands:",
	}
	rootCmd.AddGroup(cvdGroup)
	for _, c := range cvdCommands(subCmdOpts) {
		c.GroupID = cvdGroup.ID
		rootCmd.AddCommand(c)
	}
	rootCmd.AddCommand(hostCommand(subCmdOpts))
	rootCmd.AddCommand(adbTunnelCommand(subCmdOpts))
	return &CVDRemoteCommand{rootCmd}
}

func (c *CVDRemoteCommand) Execute() error {
	err := c.command.Execute()
	if err != nil {
		c.command.PrintErrln(err)
	}
	return err
}

func hostCommand(opts *subCommandOpts) *cobra.Command {
	createFlags := &CreateHostFlags{CVDRemoteFlags: opts.RootFlags, CreateHostOpts: &CreateHostOpts{}}
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
			return runListHostCommand(c, opts.RootFlags, opts)
		},
	}
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runDeleteHostsCommand(c, args, opts.RootFlags, opts)
		},
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func cvdCommands(opts *subCommandOpts) []*cobra.Command {
	createFlags := &CreateCVDFlags{
		CVDRemoteFlags: opts.RootFlags,
		CreateCVDOpts:  &CreateCVDOpts{},
		CreateHostOpts: &CreateHostOpts{},
	}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a CVD",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateCVDCommand(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.Host, hostFlag, "", "Specifies the host")
	create.Flags().StringVar(&createFlags.Branch, branchFlag, "aosp-master", "The branch name")
	create.Flags().StringVar(&createFlags.BuildID, buildIDFlag, "", "Android build identifier")
	create.MarkFlagsMutuallyExclusive(branchFlag, buildIDFlag)
	create.Flags().StringVar(&createFlags.Target, targetFlag, "aosp_cf_x86_64_phone-userdebug",
		"Android build target")
	create.Flags().BoolVar(&createFlags.LocalImage, localImageFlag, false,
		"Builds a CVD with image files built locally, the required files are https://cs.android.com/android/platform/superproject/+/master:device/google/cuttlefish/required_images and cvd-host-packages.tar.gz")
	localImgMutuallyExFlags := []string{branchFlag, buildIDFlag, targetFlag}
	for _, f := range localImgMutuallyExFlags {
		create.MarkFlagsMutuallyExclusive(f, localImageFlag)
	}
	// Host flags
	createHostFlags := []struct {
		ValueRef *string
		Name     string
		Default  string
		Desc     string
	}{
		{
			ValueRef: &createFlags.GCP.MachineType,
			Name:     gcpMachineTypeFlag,
			Default:  opts.InitialConfig.Host.GCP.MachineType,
			Desc:     gcpMachineTypeFlagDesc,
		},
		{
			ValueRef: &createFlags.GCP.MinCPUPlatform,
			Name:     gcpMinCPUPlatformFlag,
			Default:  opts.InitialConfig.Host.GCP.MinCPUPlatform,
			Desc:     gcpMinCPUPlatformFlagDesc,
		},
	}
	for _, f := range createHostFlags {
		name := "host_" + f.Name
		create.Flags().StringVar(f.ValueRef, name, f.Default, f.Desc)
		create.MarkFlagsMutuallyExclusive(hostFlag, name)
	}
	listFlags := &ListCVDsFlags{CVDRemoteFlags: opts.RootFlags}
	list := &cobra.Command{
		Use:   "list",
		Short: "List CVDs",
		RunE: func(c *cobra.Command, args []string) error {
			return runListCVDsCommand(c, listFlags, opts)
		},
	}
	list.Flags().StringVar(&listFlags.Host, hostFlag, "", "Specifies the host")
	return []*cobra.Command{create, list}
}

func adbTunnelCommand(opts *subCommandOpts) *cobra.Command {
	adbTunnelFlags := &ADBTunnelFlags{opts.RootFlags, ""}
	open := &cobra.Command{
		Use:   ADBTunnelOpenCommandName,
		Short: "Opens an ADB tunnel.",
		RunE: func(c *cobra.Command, args []string) error {
			return runConnectCommand(adbTunnelFlags, &command{c, &adbTunnelFlags.Verbose}, args, opts)
		},
	}
	open.PersistentFlags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	open.MarkPersistentFlagRequired(hostFlag)
	listFlags := &listADBTunnelFlags{
		ADBTunnelFlags: adbTunnelFlags,
		longFormat:     false,
	}
	list := &cobra.Command{
		Use:   ADBTunnelListCommandName,
		Short: "Lists open ADB tunnels.",
		RunE: func(c *cobra.Command, args []string) error {
			return runListADBTunnelsCommand(listFlags, &command{c, &adbTunnelFlags.Verbose}, opts)
		},
	}
	list.Flags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	list.Flags().BoolVarP(&listFlags.longFormat, "long", "l", false, "Print with long format.")
	closeFlags := &closeADBTunnelFlags{adbTunnelFlags, false /*skipConfirmation*/}
	close := &cobra.Command{
		Use:   fmt.Sprintf("%s <foo> <bar> <baz>", ADBTunnelCloseCommandName),
		Short: "Close ADB tunnels.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCloseADBTunnelsCommand(closeFlags, &command{c, &adbTunnelFlags.Verbose}, args, opts)
		},
	}
	close.Flags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	close.Flags().BoolVarP(&closeFlags.skipConfirmation, "yes", "y", false,
		"Don't ask for confirmation for closing multiple tunnels.")
	agent := &cobra.Command{
		Hidden: true,
		Use:    ADBTunnelAgentCommandName,
		RunE: func(c *cobra.Command, args []string) error {
			return runADBTunnelAgentCommand(adbTunnelFlags, &command{c, &adbTunnelFlags.Verbose}, args, opts)
		},
	}
	agent.Flags().StringVar(&adbTunnelFlags.host, hostFlag, "", "Specifies the host")
	agent.MarkPersistentFlagRequired(hostFlag)
	adbTunnel := &cobra.Command{
		Use:   ADBTunnelCommandName,
		Short: "Work with ADB tunnels",
	}
	adbTunnel.AddCommand(open)
	adbTunnel.AddCommand(list)
	adbTunnel.AddCommand(close)
	adbTunnel.AddCommand(agent)
	return adbTunnel
}

func runCreateHostCommand(c *cobra.Command, flags *CreateHostFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CVDRemoteFlags, c)
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

func runListHostCommand(c *cobra.Command, flags *CVDRemoteFlags, opts *subCommandOpts) error {
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

func runDeleteHostsCommand(c *cobra.Command, args []string, flags *CVDRemoteFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags, c)
	if err != nil {
		return err
	}
	return service.DeleteHosts(args)
}

func runCreateCVDCommand(c *cobra.Command, flags *CreateCVDFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CVDRemoteFlags, c)
	if err != nil {
		return fmt.Errorf("Failed to build service instance: %w", err)
	}
	if flags.CreateCVDOpts.Host == "" {
		ins, err := createHost(service, *flags.CreateHostOpts)
		if err != nil {
			return fmt.Errorf("Failed to create host: %w", err)
		}
		flags.CreateCVDOpts.Host = ins.Name
	}
	cvd, err := createCVD(service, *flags.CreateCVDOpts)
	if err != nil {
		return err
	}
	connectFlags := &ADBTunnelFlags{
		CVDRemoteFlags: flags.CVDRemoteFlags,
		host:           flags.CreateCVDOpts.Host,
	}
	cvdO := &CVDOutput{cvd, -1}
	onConn := func(_ string, p int) {
		cvdO.ADBPort = p
	}
	err = ConnectDevices(connectFlags, &command{c, &flags.Verbose}, []string{cvd.Name}, opts, onConn)
	c.Println(cvdO)
	return err
}

func runListCVDsCommand(c *cobra.Command, flags *ListCVDsFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CVDRemoteFlags, c)
	if err != nil {
		return err
	}
	var cvds []*CVDInfo
	if flags.Host != "" {
		cvds, err = listHostCVDs(service, flags.Host)
	} else {
		cvds, err = listAllCVDs(service)
	}
	fwds, err := listADBTunnels(opts.InitialConfig.ADBControlDirExpanded())
	if err != nil {
		// Non fata error, the list of CVDs is still valid
		c.PrintErrf("Error found listing CVD connections: %v\n", err)
	}
	fwMap := make(map[CVD]*ADBForwarderStatus)
	for _, f := range fwds {
		fwMap[f.CVD] = &f
	}

	for _, cvd := range cvds {
		o := &CVDOutput{cvd, -1}
		if f := fwMap[cvd.CVD]; f != nil {
			o.ADBPort = f.Port
		}
		c.Println(o)
	}
	return err
}

// Starts an ADB agent process for each device. Waits for all started subprocesses
// to report the tunnel was successfully created or an error occurred. Returns a
// summary of errors reported by the ADB agents or nil if all succeeded. Some
// tunnels may have been established even if this function returns an error. Those
// are discoverable through listADBTunnels() after this function returns.
func ConnectDevices(flags *ADBTunnelFlags, c *command, args []string, opts *subCommandOpts, connReporter func(string, int)) error {
	// Clean old logs files as we are about to create new ones.
	defer func() {
		minAge := opts.InitialConfig.LogFilesDeleteThreshold()
		if cnt, err := maybeCleanOldLogs(opts.InitialConfig.ADBControlDirExpanded(), minAge); err != nil {
			// This is not a fatal error, just inform the user
			c.PrintErrf("Error deleting old logs: %v\n", err)
		} else if cnt > 0 {
			c.PrintErrf("Deleted %d old log files\n", cnt)
		}
	}()

	agentLauncher := func(device string) int {
		cmdArgs := buildAgentCmdArgs(flags, device)

		output, err := opts.CommandRunner.StartBgCommand(cmdArgs...)
		if err != nil {
			c.PrintErrf("Unable to start connection agent: %v\n", err)
			return -1
		}

		if len(output) == 0 {
			// The pipe was closed before any data was written because no connection was established.
			// No need to print error: the agent took care of that.
			// This is not equivalent to reading more than zero bytes from stderr since the agent
			// could write messages and warnings there without failing.
			return -1
		}

		var port int
		if _, err := fmt.Sscan(string(output), &port); err != nil {
			c.PrintErrf("Failed to parse port from agent output: %v\n", err)
			return -1
		}

		return port
	}

	return openADBTunnels(args, agentLauncher, connReporter)
}
func runConnectCommand(flags *ADBTunnelFlags, c *command, args []string, opts *subCommandOpts) error {
	connReporter := func(device string, port int) {
		c.Printf("%s/%s: %d\n", flags.host, device, port)
	}
	return ConnectDevices(flags, c, args, opts, connReporter)
}

func buildAgentCmdArgs(flags *ADBTunnelFlags, device string) []string {
	args := []string{
		ADBTunnelCommandName,
		ADBTunnelAgentCommandName,
		device,
	}
	return append(args, flags.AsArgs()...)
}

// Handler for the agent command. This is not meant to be called by the user
// directly, but instead is started by the open command.
// The process starts executing in the foreground, with its stderr connected to
// the terminal. If an error occurs the process exits with a non-zero exit code
// and the error is printed to stderr. If the connection is successfully
// established, the process closes all its standard IO channels and continues
// execution in the background. Any errors detected when the process is in
// background are written to the log file instead.
func runADBTunnelAgentCommand(flags *ADBTunnelFlags, c *command, args []string, opts *subCommandOpts) error {
	if len(args) > 1 {
		return fmt.Errorf("ADB tunnel agent only supports a single device, received: %v", args)
	}
	if len(args) == 0 {
		return fmt.Errorf("Missing device")
	}
	device := args[0]
	service, err := opts.ServiceBuilder(flags.CVDRemoteFlags, c.Command)
	if err != nil {
		return err
	}

	devSpec := CVD{
		ServiceRootEndpoint: service.RootURI(),
		Host:                flags.host,
		Name:                device,
	}

	controlDir := opts.InitialConfig.ADBControlDirExpanded()
	ret, err := findOrCreateTunnel(controlDir, devSpec, service)
	if err != nil {
		return err
	}
	if ret.Error != nil {
		// A tunnel was found or created, but a non-fatal error occurred.
		c.PrintErrln(ret.Error)
	}

	// The agent's only output is the port
	c.Println(ret.Port)

	if ret.Controller == nil {
		// A tunnel already exist, this process is done.
		return nil
	}

	// Signal the caller that the agent is moving to the background.
	os.Stdin.Close()
	os.Stdout.Close()
	os.Stderr.Close()

	ret.Controller.Run()

	// There is no point in returning error codes from a background process, errors
	// will be written to the log file instead.
	return nil
}

func printADBTunnels(forwarders []ADBForwarderStatus, c *command, longFormat bool) {
	for _, fwdr := range forwarders {
		line := fmt.Sprintf("%s/%s 127.0.0.1:%d %s",
			fwdr.Host, fwdr.Name, fwdr.Port, fwdr.State)
		if longFormat {
			line = fmt.Sprintf("%s: %s", fwdr.ServiceRootEndpoint, line)
		}
		c.Println(line)
	}
}

func runListADBTunnelsCommand(flags *listADBTunnelFlags, c *command, opts *subCommandOpts) error {
	controlDir := opts.InitialConfig.ADBControlDirExpanded()
	var forwarders []ADBForwarderStatus
	var err error
	if flags.host != "" {
		forwarders, err = listADBTunnelsByHost(controlDir, flags.host)
	} else {
		forwarders, err = listADBTunnels(controlDir)
	}
	// Print the tunnels we may have found before an error was encountered if any.
	printADBTunnels(forwarders, c, flags.longFormat)
	return err
}

func runCloseADBTunnelsCommand(flags *closeADBTunnelFlags, c *command, args []string, opts *subCommandOpts) error {
	devices := make(map[string]any)
	for _, a := range args {
		devices[a] = nil
	}
	controlDir := opts.InitialConfig.ADBControlDirExpanded()
	var forwarders []ADBForwarderStatus
	var merr error
	if flags.host != "" {
		forwarders, merr = listADBTunnelsByHost(controlDir, flags.host)
	} else {
		forwarders, merr = listADBTunnels(controlDir)
	}
	if len(forwarders) == 0 {
		return fmt.Errorf("No ADB tunnels found")
	}
	// Restrict the list of tunnels to those specified as arguments
	if len(args) > 0 {
		var inArgs []ADBForwarderStatus
		for _, fwdr := range forwarders {
			if _, in := devices[fwdr.Name]; in {
				inArgs = append(inArgs, fwdr)
				delete(devices, fwdr.Name)
			}
		}
		forwarders = inArgs
		for device := range devices {
			merr = multierror.Append(merr, fmt.Errorf("ADB Tunnel not found for %q\n", device))
		}
	}
	if len(forwarders) > 1 && !flags.skipConfirmation {
		var err error
		forwarders, err = promptTunnelSelection(forwarders, c)
		if err != nil {
			// A failure to read user input cancels the entire command.
			return err
		}
	}
	for _, dev := range forwarders {
		if err := closeTunnel(controlDir, dev); err != nil {
			multierror.Append(merr, err)
			continue
		}
		c.Printf("%s/%s: closed\n", dev.Host, dev.Name)
	}
	return merr
}

func promptTunnelSelection(devices []ADBForwarderStatus, c *command) ([]ADBForwarderStatus, error) {
	c.PrintErrln("Multiple ADB tunnels match:")
	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = fmt.Sprintf("%s %s", d.Host, d.Name)
	}
	choices, err := c.PromptSelection(names, AllowAll)
	if err != nil {
		return nil, err
	}
	ret := make([]ADBForwarderStatus, len(choices))
	for i, v := range choices {
		ret[i] = devices[v]
	}
	return ret, nil
}

type serviceBuilder func(flags *CVDRemoteFlags, c *cobra.Command) (client.Service, error)

const chunkSizeBytes = 16 * 1024 * 1024

func buildServiceBuilder(builder client.ServiceBuilder) serviceBuilder {
	return func(flags *CVDRemoteFlags, c *cobra.Command) (client.Service, error) {
		proxyURL := flags.HTTPProxy
		var dumpOut io.Writer = io.Discard
		if flags.Verbose {
			dumpOut = c.ErrOrStderr()
		}
		opts := &client.ServiceOptions{
			RootEndpoint:           buildServiceRootEndpoint(flags.ServiceURL, flags.Zone),
			ProxyURL:               proxyURL,
			DumpOut:                dumpOut,
			ErrOut:                 c.ErrOrStderr(),
			RetryAttempts:          3,
			RetryDelay:             5 * time.Second,
			ChunkSizeBytes:         chunkSizeBytes,
			ChunkUploadBackOffOpts: client.DefaultChunkUploadBackOffOpts(),
		}
		return builder(opts)
	}
}

func notImplementedCommand(c *cobra.Command, _ []string) error {
	return fmt.Errorf("Command not implemented")
}

func buildServiceRootEndpoint(serviceURL, zone string) string {
	const version = "v1"
	return client.BuildRootEndpoint(serviceURL, version, zone)
}
