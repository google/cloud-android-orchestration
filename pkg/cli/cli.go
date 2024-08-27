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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	client "github.com/google/cloud-android-orchestration/pkg/client"
	wclient "github.com/google/cloud-android-orchestration/pkg/webrtcclient"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
	"golang.org/x/net/proxy"
	"golang.org/x/term"
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
	StartBgCommand(...string) ([]byte, error)
	// When needed RunCommand can be added, returning the exit code, the output and
	// an error.
}

type CommandOptions struct {
	IOStreams
	Args           []string
	InitialConfig  Config
	ServiceBuilder client.ServiceBuilder
	CommandRunner  CommandRunner
	ADBServerProxy ADBServerProxy
}

type CVDRemoteCommand struct {
	command *cobra.Command
	options *CommandOptions
}

const (
	serviceFlag    = "service"
	hostFlag       = "host"
	groupFlag      = "group"
	serviceURLFlag = "service_url"
	zoneFlag       = "zone"
	proxyFlag      = "proxy"
	verboseFlag    = "verbose"
)

const (
	acceleratorFlag = "accelerator"

	gcpMachineTypeFlag    = "gcp_machine_type"
	gcpMinCPUPlatformFlag = "gcp_min_cpu_platform"
)

const (
	acceleratorFlagDesc       = "Configuration to attach accelerator cards, i.e: --accelerator type=nvidia-tesla-p100,count=1"
	gcpMachineTypeFlagDesc    = "Indicates the machine type"
	gcpMinCPUPlatformFlagDesc = "Specifies a minimum CPU platform for the VM instance"
)

const (
	branchFlag                 = "branch"
	buildIDFlag                = "build_id"
	buildTargetFlag            = "build_target"
	localImageFlag             = "local_image"
	kernelBranchFlag           = "kernel_branch"
	kernelBuildIDFlag          = "kernel_build_id"
	kernelBuildTargetFlag      = "kernel_build_target"
	bootloaderBranchFlag       = "bootloader_branch"
	bootloaderBuildIDFlag      = "bootloader_build_id"
	bootloaderBuildTargetFlag  = "bootloader_build_target"
	systemImgBranchFlag        = "system_branch"
	systemImgBuildIDFlag       = "system_build_id"
	systemImgBuildTargetFlag   = "system_build_target"
	numInstancesFlag           = "num_instances"
	autoConnectFlag            = "auto_connect"
	buildAPICredsSourceFlag    = "build_api_creds_source"
	buildAPICredsUserProjectID = "build_api_creds_user_project_id"
	localBootloaderSrcFlag     = "local_bootloader_src"
	localCVDHostPkgSrcFlag     = "local_cvd_host_pkg_src"
	localImagesSrcsFlag        = "local_images_srcs"
	localImagesZipSrcFlag      = "local_images_zip_src"
)

const (
	ConnectCommandName                  = "connect"
	DisconnectCommandName               = "disconnect"
	ConnectionWebRTCAgentCommandName    = "webrtc_agent"
	ConnectionProxyAgentCommandName     = "proxy_agent"
	ConnectionWebSocketAgentCommandName = "websocket_agent"
)

const (
	iceConfigFlag = "ice_config"
)

const (
	iceConfigFlagDesc = "Path to file containing the ICE configuration to be used in the underlaying WebRTC connection"
)

type AsArgs interface {
	AsArgs() []string
}

type RootFlags struct {
	Service string
	Verbose bool
}

type ServiceFlags struct {
	*RootFlags

	ServiceURL string
	Zone       string
	Proxy      string
}

func (f *ServiceFlags) Update(s *Service) {
	f.ServiceURL = s.ServiceURL
	f.Zone = s.Zone
	f.Proxy = s.Proxy
}

func (f *ServiceFlags) AsArgs() []string {
	args := []string{
		"--" + serviceURLFlag, f.ServiceURL,
		"--" + zoneFlag, f.Zone,
	}
	if f.Proxy != "" {
		args = append(args, "--"+proxyFlag, f.Proxy)
	}
	if f.Verbose {
		args = append(args, "-v")
	}
	return args
}

type CreateHostFlags struct {
	*ServiceFlags
	*CreateHostOpts
}

func (f *CreateHostFlags) Update(s *Service) {
	f.ServiceFlags.Update(s)
	f.CreateHostOpts.Update(s)
}

type CreateCVDFlags struct {
	*ServiceFlags
	*CreateCVDOpts
	*CreateHostOpts
}

func (f *CreateCVDFlags) Update(s *Service) {
	f.ServiceFlags.Update(s)
	f.CreateCVDOpts.Update(s)
	f.CreateHostOpts.Update(s)
}

type ListCVDsFlags struct {
	*ServiceFlags
	Host string
}

type BugreportFlags struct {
	*ServiceFlags
	Host  string
	Group string
}

type DeleteCVDFlags struct {
	*ServiceFlags
	Host string
}

type subCommandOpts struct {
	ServiceBuilder serviceBuilder
	ServiceFlags   *ServiceFlags
	InitialConfig  Config
	CommandRunner  CommandRunner
	ADBServerProxy ADBServerProxy
}

type ConnectFlags struct {
	*ServiceFlags
	host             string
	skipConfirmation bool
	// Path to file containing the ICE configuration to be used in the underlaying WebRTC connection.
	ice_config   string
	connectAgent string
}

func (f *ConnectFlags) AsArgs() []string {
	args := f.ServiceFlags.AsArgs()
	if f.host != "" {
		args = append(args, "--"+hostFlag, f.host)
	}
	if f.ice_config != "" {
		args = append(args, "--"+iceConfigFlag, f.ice_config)
	}
	return args
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

func WriteListCVDsOutput(w io.Writer, hosts []*RemoteHost) {
	for _, host := range hosts {
		fmt.Fprintln(w, hostOutput(host))
		if len(host.CVDs) == 0 {
			fmt.Fprintln(w, " ~~Empty~~ ")
			fmt.Fprintln(w, "")
			continue
		}
		for _, cvd := range host.CVDs {
			lines := cvdOutput(cvd)
			for _, l := range lines {
				fmt.Fprintln(w, "  "+l)
			}
			fmt.Fprintln(w)
		}
	}
}

func hostOutput(h *RemoteHost) string {
	return fmt.Sprintf("%s (%s)",
		h.Name,
		client.BuilHostIndexURL(h.ServiceRootEndpoint, h.Name))
}

func cvdOutput(c *RemoteCVD) []string {
	return []string{
		c.ID,
		"Status: " + c.Status,
		"ADB: " + adbStateStr(c),
		"Displays: " + fmt.Sprintf("%v", c.Displays),
		"Logs: " + client.BuildCVDLogsURL(c.ServiceRootEndpoint, c.Host, c.ID),
	}
}

func adbStateStr(c *RemoteCVD) string {
	if c.ConnStatus != nil {
		if c.ConnStatus.ADB.Port > 0 {
			return fmt.Sprintf("127.0.0.1:%d", c.ConnStatus.ADB.Port)
		} else {
			return c.ConnStatus.ADB.State
		}
	}
	return "not connected"
}

type SelectionOption int32

const (
	Single   SelectionOption = 0
	AllowAll SelectionOption = 1 << iota
)

// The PromptSelectionFromSlice<Type> functions iterate over given container and present
// users with a prompt like this:
// 0: String representation of first choice
// 2: String representation of second choice
// ...
// N: All
// Choose an option: <cursor>
// These should have been methods of command, but Go doesn't allow generic methods.
func PromptSelectionFromSlice[T any](c *command, choices []T, toStr func(T) string, selOpt SelectionOption) ([]T, error) {
	for i, v := range choices {
		c.PrintErrf("%d) %s\n", i, toStr(v))
	}
	maxChoice := len(choices) - 1
	if len(choices) > 1 && selOpt&AllowAll != 0 {
		c.PrintErrf("%d) All\n", len(choices))
		maxChoice += 1
	}
	c.PrintErrf("Choose an option: ")
	chosen := -1
	_, err := fmt.Fscanln(c.InOrStdin(), &chosen)
	if err != nil {
		return nil, fmt.Errorf("failed to read choice: %w", err)
	}
	if chosen < 0 || chosen > maxChoice {
		return nil, fmt.Errorf("choice out of range: %d", chosen)
	}
	if chosen < len(choices) {
		return []T{choices[chosen]}, nil
	}
	return choices, nil
}

func PromptSelectionFromSliceString(c *command, choices []string, selOpt SelectionOption) ([]string, error) {
	return PromptSelectionFromSlice(c, choices, func(s string) string { return s }, selOpt)
}

func PromptSelectionFromMap[K comparable, T any](c *command, choices map[K]T, toStr func(K, T) string, selOpt SelectionOption) (map[K]T, error) {
	i := 0
	keys := make([]K, len(choices))
	for k, v := range choices {
		c.PrintErrf("%d) %s\n", i, toStr(k, v))
		keys[i] = k
		i++
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
		return nil, fmt.Errorf("failed to read choice: %w", err)
	}
	if chosen < 0 || chosen > maxChoice {
		return nil, fmt.Errorf("choice out of range: %d", chosen)
	}
	if chosen < len(choices) {
		key := keys[chosen]
		return map[K]T{key: choices[key]}, nil
	}
	return choices, nil
}

func PromptYesOrNo(out *os.File, in *os.File, text string) (bool, error) {
	fmt.Fprint(out, text+" (Y/n): ")
	yn := "Y"
	_, err := fmt.Fscanln(os.Stdin, &yn)
	// Using the error text for comparison as there's no specific error type to compare against.
	if err != nil && err.Error() != "unexpected newline" {
		return false, fmt.Errorf("failed to read (Y/n) choice: %w", err)
	}
	ynLo := strings.ToLower(yn)
	if ynLo != "y" && ynLo != "yes" && ynLo != "n" && ynLo != "no" {
		return false, fmt.Errorf("entered invalid value: %q", yn)
	}
	return ynLo[0] == 'y', nil
}

func NewCVDRemoteCommand(o *CommandOptions) *CVDRemoteCommand {
	flags := &ServiceFlags{RootFlags: &RootFlags{}}
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
	rootCmd.PersistentFlags().StringVarP(&flags.Service, serviceFlag, "s", "", "Select service.")
	rootCmd.PersistentFlags().StringVar(&flags.ServiceURL, serviceURLFlag, o.InitialConfig.DefaultService().ServiceURL,
		"Cloud orchestration service url.")
	if o.InitialConfig.DefaultService().ServiceURL == "" {
		// Make it required if not configured
		rootCmd.MarkPersistentFlagRequired(serviceURLFlag)
	}
	rootCmd.PersistentFlags().StringVar(&flags.Zone, zoneFlag, o.InitialConfig.DefaultService().Zone, "Cloud zone.")
	rootCmd.PersistentFlags().StringVar(&flags.Proxy, proxyFlag, o.InitialConfig.DefaultService().Proxy,
		"Proxy used to route the http communication through.")
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.PersistentFlags().BoolVarP(&flags.Verbose, verboseFlag, "v", false, "Be verbose.")
	subCmdOpts := &subCommandOpts{
		ServiceBuilder: buildServiceBuilder(o.ServiceBuilder, &o.InitialConfig),
		ServiceFlags:   flags,
		InitialConfig:  o.InitialConfig,
		CommandRunner:  o.CommandRunner,
		ADBServerProxy: o.ADBServerProxy,
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
	for _, cmd := range connectionCommands(subCmdOpts) {
		cmd.GroupID = cvdGroup.ID
		rootCmd.AddCommand(cmd)
	}
	rootCmd.AddCommand(hostCommand(subCmdOpts))
	getConfigCommand := &cobra.Command{
		Use:    "get_config",
		Short:  "Get a specific configuration value.",
		Hidden: true,
		RunE: func(c *cobra.Command, args []string) error {
			return runGetConfigCommand(c, args, o.InitialConfig)
		},
	}
	rootCmd.AddCommand(getConfigCommand)
	return &CVDRemoteCommand{rootCmd, o}
}

func (c *CVDRemoteCommand) Execute() error {
	err := EnsureConnDirsExist(c.options.InitialConfig.ConnectionControlDirExpanded())
	if err == nil {
		err = c.command.Execute()
	}
	if err != nil {
		c.command.PrintErrln(err)
	}
	return err
}

func hostCommand(opts *subCommandOpts) *cobra.Command {
	acceleratorFlagValues := []string{}
	createFlags := &CreateHostFlags{ServiceFlags: opts.ServiceFlags, CreateHostOpts: &CreateHostOpts{}}
	create := &cobra.Command{
		Use:     "create",
		Short:   "Creates a host.",
		PreRunE: preRunE(createFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			configs, err := parseAcceleratorFlag(acceleratorFlagValues)
			if err != nil {
				return err
			}
			createFlags.GCP.AcceleratorConfigs = configs
			return runCreateHostCommand(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.GCP.MachineType, gcpMachineTypeFlag,
		opts.InitialConfig.DefaultService().Host.GCP.MachineType, gcpMachineTypeFlagDesc)
	create.Flags().StringVar(&createFlags.GCP.MinCPUPlatform, gcpMinCPUPlatformFlag,
		opts.InitialConfig.DefaultService().Host.GCP.MinCPUPlatform, gcpMinCPUPlatformFlagDesc)
	list := &cobra.Command{
		Use:     "list",
		Short:   "Lists hosts.",
		PreRunE: preRunE(opts.ServiceFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runListHostCommand(c, opts.ServiceFlags, opts)
		},
	}
	del := &cobra.Command{
		Use:     "delete <foo> <bar> <baz>",
		Short:   "Delete hosts.",
		PreRunE: preRunE(opts.ServiceFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runDeleteHostsCommand(c, args, opts.ServiceFlags, opts)
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
	// Create command
	createFlags := &CreateCVDFlags{
		ServiceFlags:   opts.ServiceFlags,
		CreateCVDOpts:  &CreateCVDOpts{},
		CreateHostOpts: &CreateHostOpts{},
	}
	create := &cobra.Command{
		Use:     "create [config.json]",
		Short:   "Creates a CVD",
		PreRunE: preRunE(createFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateCVDCommand(c, args, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.Host, hostFlag, "", "Specifies the host")
	// Main build flags.
	create.Flags().StringVar(&createFlags.MainBuild.Branch, branchFlag, "aosp-main", "The branch name")
	create.Flags().StringVar(&createFlags.MainBuild.BuildID, buildIDFlag, "", "Android build identifier")
	create.Flags().StringVar(&createFlags.MainBuild.Target, buildTargetFlag, "aosp_cf_x86_64_phone-trunk_staging-userdebug",
		"Android build target")
	create.MarkFlagsMutuallyExclusive(branchFlag, buildIDFlag)
	// Kernel build flags
	create.Flags().StringVar(&createFlags.KernelBuild.Branch, kernelBranchFlag, "", "Kernel branch name")
	create.Flags().StringVar(&createFlags.KernelBuild.BuildID, kernelBuildIDFlag, "", "Kernel build identifier")
	create.Flags().StringVar(&createFlags.KernelBuild.Target, kernelBuildTargetFlag, "", "Kernel build target")
	create.MarkFlagsMutuallyExclusive(kernelBranchFlag, kernelBuildIDFlag)
	// Bootloader build flags
	create.Flags().StringVar(&createFlags.BootloaderBuild.Branch, bootloaderBranchFlag, "", "Bootloader branch name")
	create.Flags().StringVar(&createFlags.BootloaderBuild.BuildID, bootloaderBuildIDFlag, "", "Bootloader build identifier")
	create.Flags().StringVar(&createFlags.BootloaderBuild.Target, bootloaderBuildTargetFlag, "", "Bootloader build target")
	create.MarkFlagsMutuallyExclusive(bootloaderBranchFlag, bootloaderBuildIDFlag)
	// System image build flags
	create.Flags().StringVar(&createFlags.SystemImgBuild.Branch, systemImgBranchFlag, "", "System image branch name")
	create.Flags().StringVar(&createFlags.SystemImgBuild.BuildID, systemImgBuildIDFlag, "", "System image build identifier")
	create.Flags().StringVar(&createFlags.SystemImgBuild.Target, systemImgBuildTargetFlag, "", "System image build target")
	create.MarkFlagsMutuallyExclusive(systemImgBranchFlag, systemImgBuildIDFlag)
	remoteBuildFlags := []string{
		branchFlag, buildIDFlag, buildTargetFlag,
		kernelBranchFlag, kernelBuildIDFlag, kernelBuildTargetFlag,
		bootloaderBranchFlag, bootloaderBuildIDFlag, bootloaderBuildTargetFlag,
		systemImgBranchFlag, systemImgBuildIDFlag, systemImgBuildTargetFlag,
	}
	// Local image
	create.Flags().BoolVar(&createFlags.LocalImage, localImageFlag, false,
		"Create instance from a local build, the required files are https://cs.android.com/android/platform/superproject/+/master:device/google/cuttlefish/required_images and cvd-host-packages.tar.gz")
	for _, remote := range remoteBuildFlags {
		create.MarkFlagsMutuallyExclusive(localImageFlag, remote)
	}
	create.Flags().IntVar(&createFlags.NumInstances, numInstancesFlag, 1,
		"Creates multiple instances with the same artifacts. Only relevant if given a single build source")
	create.Flags().BoolVar(&createFlags.AutoConnect, autoConnectFlag, true,
		"Automatically connect through ADB after device is created.")
	create.Flags().StringVar(
		&createFlags.BuildAPICredentialsSource,
		buildAPICredsSourceFlag,
		"none",
		"Source for the Build API OAuth2 credentials")
	create.Flags().StringVar(
		&createFlags.BuildAPIUserProjectID,
		buildAPICredsUserProjectID,
		"",
		"Project ID for user accessing the fetch APIs")
	// Local artifact sources
	create.Flags().StringVar(&createFlags.LocalBootloaderSrc, localBootloaderSrcFlag, "", "Local bootloader source")
	create.Flags().StringVar(&createFlags.LocalCVDHostPkgSrc, localCVDHostPkgSrcFlag, "", "Local cvd host package source")
	create.Flags().StringSliceVar(&createFlags.LocalImagesSrcs, localImagesSrcsFlag, []string{}, "Comma-separated list of local images sources")
	create.Flags().StringVar(&createFlags.LocalImagesZipSrc, localImagesZipSrcFlag, "",
		"Local *-img-*.zip source containing the images and bootloader files")
	create.MarkFlagsMutuallyExclusive(localImagesZipSrcFlag, localBootloaderSrcFlag)
	create.MarkFlagsMutuallyExclusive(localImagesZipSrcFlag, localImagesSrcsFlag)
	localSrcsFlag := []string{localBootloaderSrcFlag, localCVDHostPkgSrcFlag, localImagesSrcsFlag, localImagesZipSrcFlag}
	for _, local := range localSrcsFlag {
		create.MarkFlagsMutuallyExclusive(local, localImageFlag)
		for _, remote := range remoteBuildFlags {
			create.MarkFlagsMutuallyExclusive(local, remote)
		}
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
			Default:  opts.InitialConfig.DefaultService().Host.GCP.MachineType,
			Desc:     gcpMachineTypeFlagDesc,
		},
		{
			ValueRef: &createFlags.GCP.MinCPUPlatform,
			Name:     gcpMinCPUPlatformFlag,
			Default:  opts.InitialConfig.DefaultService().Host.GCP.MinCPUPlatform,
			Desc:     gcpMinCPUPlatformFlagDesc,
		},
	}
	for _, f := range createHostFlags {
		name := "host_" + f.Name
		create.Flags().StringVar(f.ValueRef, name, f.Default, f.Desc)
		create.MarkFlagsMutuallyExclusive(hostFlag, name)
	}
	// List command
	listFlags := &ListCVDsFlags{ServiceFlags: opts.ServiceFlags}
	list := &cobra.Command{
		Use:     "list",
		Short:   "List CVDs",
		PreRunE: preRunE(listFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runListCVDsCommand(c, listFlags, opts)
		},
	}
	list.Flags().StringVar(&listFlags.Host, hostFlag, "", "Specifies the host")
	// Bugreport command
	brFlags := &BugreportFlags{ServiceFlags: opts.ServiceFlags}
	br := &cobra.Command{
		Use:     "bugreport [--host HOST [--group GROUP]]",
		Short:   "Pull cvd runtime artifacts",
		PreRunE: preRunE(opts.ServiceFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runBugreportCommand(c, brFlags, opts)
		},
	}
	br.Flags().StringVar(&brFlags.Host, hostFlag, "", "Specifies the host")
	br.Flags().StringVar(&brFlags.Group, groupFlag, "", "Specifies the group")
	// Delete command
	delFlags := &DeleteCVDFlags{ServiceFlags: opts.ServiceFlags}
	del := &cobra.Command{
		Use:     "delete [--host=HOST] [id]",
		Short:   "Deletes cvd instance",
		PreRunE: preRunE(delFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runDeleteCVDCommand(c, args, delFlags, opts)
		},
	}
	del.Flags().StringVar(&delFlags.Host, hostFlag, "", "Specifies the host")
	del.MarkFlagRequired(hostFlag)
	return []*cobra.Command{create, list, br, del}
}

func connectionCommands(opts *subCommandOpts) []*cobra.Command {
	connFlags := &ConnectFlags{ServiceFlags: opts.ServiceFlags, host: "", skipConfirmation: false, connectAgent: ConnectionWebRTCAgentCommandName}
	connect := &cobra.Command{
		Use:     ConnectCommandName,
		Short:   "(Re)Connects to a CVD and tunnels ADB messages",
		PreRunE: preRunE(connFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runConnectCommand(connFlags, &command{c, &connFlags.Verbose}, args, opts)
		},
	}
	connect.Flags().StringVar(&connFlags.host, hostFlag, "", "Specifies the host")
	connect.Flags().BoolVarP(&connFlags.skipConfirmation, "yes", "y", false,
		"Don't ask for confirmation for closing multiple connections.")
	connect.Flags().StringVar(&connFlags.ice_config, iceConfigFlag, "", iceConfigFlagDesc)
	connect.Flags().StringVar(&connFlags.connectAgent, "connect_agent", ConnectionWebRTCAgentCommandName, "Connect agent type")
	disconnect := &cobra.Command{
		Use:     fmt.Sprintf("%s <foo> <bar> <baz>", DisconnectCommandName),
		Short:   "Disconnect (ADB) from CVD",
		PreRunE: preRunE(connFlags, &opts.ServiceFlags.Service, &opts.InitialConfig),
		RunE: func(c *cobra.Command, args []string) error {
			return runDisconnectCommand(connFlags, &command{c, &connFlags.Verbose}, args, opts)
		},
	}
	disconnect.Flags().StringVar(&connFlags.host, hostFlag, "", "Specifies the host")
	disconnect.Flags().BoolVarP(&connFlags.skipConfirmation, "yes", "y", false,
		"Don't ask for confirmation for closing multiple connections.")
	webrtcAgent := &cobra.Command{
		Hidden: true,
		Use:    ConnectionWebRTCAgentCommandName,
		RunE: func(c *cobra.Command, args []string) error {
			return runConnectionWebrtcAgentCommand(connFlags, &command{c, &connFlags.Verbose}, args, opts)
		},
	}
	webrtcAgent.Flags().StringVar(&connFlags.host, hostFlag, "", "Specifies the host")
	webrtcAgent.Flags().StringVar(&connFlags.ice_config, iceConfigFlag, "", iceConfigFlagDesc)
	webrtcAgent.MarkPersistentFlagRequired(hostFlag)
	proxyAgent := &cobra.Command{
		Hidden: true,
		Use:    ConnectionProxyAgentCommandName,
		RunE: func(c *cobra.Command, args []string) error {
			return runConnectionProxyAgentCommand(connFlags, &command{c, &connFlags.Verbose}, args, opts)
		},
	}
	proxyAgent.Flags().StringVar(&connFlags.host, hostFlag, "", "Specifies the host")
	proxyAgent.MarkPersistentFlagRequired(hostFlag)
	webSocketAgent := &cobra.Command{
		Hidden: true,
		Use:    ConnectionWebSocketAgentCommandName,
		RunE: func(c *cobra.Command, args []string) error {
			return runConnectionWebSocketAgentCommand(connFlags, &command{c, &connFlags.Verbose}, args, opts)
		},
	}
	webSocketAgent.Flags().StringVar(&connFlags.host, hostFlag, "", "Specifies the host")
	webSocketAgent.MarkPersistentFlagRequired(hostFlag)
	return []*cobra.Command{connect, disconnect, webrtcAgent, proxyAgent, webSocketAgent}
}

func runCreateHostCommand(c *cobra.Command, flags *CreateHostFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c)
	if err != nil {
		return fmt.Errorf("failed to build service instance: %w", err)
	}
	ins, err := createHost(service, *flags.CreateHostOpts)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	c.Printf("%s\n", ins.Name)
	return nil
}

func runListHostCommand(c *cobra.Command, flags *ServiceFlags, opts *subCommandOpts) error {
	apiClient, err := opts.ServiceBuilder(flags, c)
	if err != nil {
		return err
	}
	hosts, err := apiClient.ListHosts()
	if err != nil {
		return fmt.Errorf("error listing hosts: %w", err)
	}
	for _, ins := range hosts.Items {
		c.Printf("%s\n", ins.Name)
	}
	return nil
}

func runDeleteHostsCommand(c *cobra.Command, args []string, flags *ServiceFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags, c)
	if err != nil {
		return err
	}
	hosts := args
	if len(hosts) == 0 {
		if hosts, err = promptHostNameSelection(&command{c, &flags.Verbose}, service, AllowAll); err != nil {
			return err
		}
	}
	// Close connections first to avoid spurious error messages later.
	for _, host := range hosts {
		if err := disconnectDevicesByHost(host, opts); err != nil {
			// Warn only, the host can still be deleted
			c.PrintErrf("Warning: Failed to disconnect devices for host %s: %v\n", host, err)
		}
	}
	return service.DeleteHosts(hosts)
}

func disconnectDevicesByHost(host string, opts *subCommandOpts) error {
	controlDir := opts.InitialConfig.ConnectionControlDirExpanded()
	statuses, err := listCVDConnectionsByHost(controlDir, host)
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}
	var merr error
	for cvd, status := range statuses {
		if err := DisconnectCVD(controlDir, cvd, status); err != nil {
			merr = multierror.Append(merr, fmt.Errorf("failed to disconnect from %s: %w", cvd.WebRTCDeviceID, err))
		}
	}
	return merr
}

const (
	createHostStateMsg    = "Creating Host"
	connectCVDStateMsgFmt = "Connecting to %s"
)

func runCreateCVDCommand(c *cobra.Command, args []string, flags *CreateCVDFlags, opts *subCommandOpts) error {
	if len(args) > 0 {
		// Load and parse the passed environment specification.
		filename := args[0]
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("invalid environment specification file %q: %w", filename, err)
		}
		envConfig := make(map[string]interface{})
		if err := json.Unmarshal(data, &envConfig); err != nil {
			return fmt.Errorf("invalid environment specification: %w", err)
		}
		flags.CreateCVDOpts.EnvConfig = envConfig
	}
	if flags.NumInstances <= 0 {
		return fmt.Errorf("invalid --num_instances flag value: %d", flags.NumInstances)
	}
	statePrinter := newStatePrinter(c.ErrOrStderr(), flags.Verbose)
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c)
	if err != nil {
		return fmt.Errorf("failed to build service instance: %w", err)
	}
	if flags.CreateCVDOpts.Host == "" {
		statePrinter.Print(createHostStateMsg)
		ins, err := createHost(service, *flags.CreateHostOpts)
		statePrinter.PrintDone(createHostStateMsg, err)
		if err != nil {
			return fmt.Errorf("failed to create host: %w", err)
		}
		flags.CreateCVDOpts.Host = ins.Name
	}
	cvds, err := createCVD(service, *flags.CreateCVDOpts, statePrinter)
	if err != nil {
		var apiErr *client.ApiCallError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusUnauthorized {
			c.PrintErrf("Authorization required, please visit %s/auth\n", flags.ServiceURL)
		}
		return err
	}
	var merr error
	if flags.CreateCVDOpts.AutoConnect {
		for _, cvd := range cvds {
			statePrinter.Print(fmt.Sprintf(connectCVDStateMsgFmt, cvd.WebRTCDeviceID))
			cvd.ConnStatus, err = ConnectDevice(flags.CreateCVDOpts.Host, cvd.WebRTCDeviceID, "", ConnectionWebRTCAgentCommandName, &command{c, &flags.Verbose}, opts)
			statePrinter.PrintDone(fmt.Sprintf(connectCVDStateMsgFmt, cvd.WebRTCDeviceID), err)
			if err != nil {
				merr = multierror.Append(merr, fmt.Errorf("failed to connect to device: %w", err))
			}
		}
	}
	hosts := []*RemoteHost{
		{
			ServiceRootEndpoint: service.RootURI(),
			Name:                flags.CreateCVDOpts.Host,
			CVDs:                cvds,
		},
	}
	WriteListCVDsOutput(c.OutOrStdout(), hosts)
	return merr
}

func runListCVDsCommand(c *cobra.Command, flags *ListCVDsFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c)
	if err != nil {
		return err
	}
	var hosts []*RemoteHost
	if flags.Host != "" {
		hosts, err = listCVDsSingleHost(service, opts.InitialConfig.ConnectionControlDirExpanded(), flags.Host)
	} else {
		hosts, err = listCVDs(service, opts.InitialConfig.ConnectionControlDirExpanded())
	}
	WriteListCVDsOutput(c.OutOrStdout(), hosts)
	return err
}

func runBugreportCommand(c *cobra.Command, flags *BugreportFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c)
	if err != nil {
		return err
	}
	host := flags.Host
	if host == "" {
		sel, err := promptSingleHostNameSelection(&command{c, &flags.Verbose}, service)
		if err != nil {
			return err
		}
		if sel == "" {
			c.PrintErrln("no host selected")
			return nil
		}
		host = sel
	}
	group := flags.Group
	if group == "" {
		sel, err := promptSingleGroupNameSelection(
			&command{c, &flags.Verbose},
			service,
			opts.InitialConfig.ConnectionControlDirExpanded(),
			host)
		if err != nil {
			return err
		}
		if sel == "" {
			c.PrintErrln("no cvd group selected")
			return nil
		}
		group = sel
	}
	f, err := os.CreateTemp("", "cvdr_bugreport")
	if err != nil {
		return err
	}
	if err := service.HostService(host).CreateBugreport(group, f); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	c.Println("Bugreport created: " + f.Name())
	return nil
}

func runDeleteCVDCommand(c *cobra.Command, args []string, flags *DeleteCVDFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("missing id")
	}
	if len(args) > 1 {
		return errors.New("deleting multiple instances is not supported yet")
	}
	return service.HostService(flags.Host).DeleteCVD(args[0])
}

// Returns empty string if there was no host.
func promptSingleHostNameSelection(c *command, service client.Service) (string, error) {
	sel, err := promptHostNameSelection(c, service, Single)
	if err != nil {
		return "", err
	}
	if len(sel) > 1 {
		log.Fatalf("expected one item, got %+v", sel)
	}
	return sel[0], nil
}

// Returns empty list if there was no host.
func promptHostNameSelection(c *command, service client.Service, selOpt SelectionOption) ([]string, error) {
	names, err := hostnames(service)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}
	if len(names) == 0 {
		return []string{}, nil
	}
	return PromptSelectionFromSliceString(c, names, selOpt)
}

// Returns empty string if there was no group.
func promptSingleGroupNameSelection(c *command, service client.Service, controlDir, host string) (string, error) {
	hosts, err := listCVDsSingleHost(service, controlDir, host)
	if err != nil {
		return "", err
	}
	cvds := flattenCVDs(hosts)
	names := []string{}
	for _, e := range cvds {
		names = append(names, e.Group())
	}
	sel, err := PromptSelectionFromSliceString(c, names, Single)
	if err != nil {
		return "", err
	}
	if len(sel) > 1 {
		log.Fatalf("expected one item, got %+v", sel)
	}
	return sel[0], nil
}

// Starts a connection agent process and waits for it to report the connection was
// successfully created or an error occurred.
func ConnectDevice(host, device, ice_config, agent string, c *command, opts *subCommandOpts) (*ConnStatus, error) {
	// Clean old logs files as we are about to create new ones.
	go func() {
		minAge := opts.InitialConfig.LogFilesDeleteThreshold()
		if cnt, err := maybeCleanOldLogs(opts.InitialConfig.ConnectionControlDirExpanded(), minAge); err != nil {
			// This is not a fatal error, just inform the user
			c.PrintErrf("Error deleting old logs: %v\n", err)
		} else if cnt > 0 {
			c.PrintErrf("Deleted %d old log files\n", cnt)
		}
	}()

	flags := &ConnectFlags{
		ServiceFlags: opts.ServiceFlags,
		host:         host,
		ice_config:   ice_config,
	}
	cmdArgs := buildAgentCmdArgs(flags, device, agent)

	output, err := opts.CommandRunner.StartBgCommand(cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("unable to start connection agent: %w", err)
	}

	if len(output) == 0 {
		// The pipe was closed before any data was written because no connection was established.
		// No need to print error: the agent took care of that.
		// This is not equivalent to reading more than zero bytes from stderr since the agent
		// could write messages and warnings there without failing.
		return nil, fmt.Errorf("no response from agent")
	}

	var status ConnStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return nil, fmt.Errorf("failed to decode agent output(%s): %w", string(output), err)
	}

	return &status, nil
}

func runConnectCommand(flags *ConnectFlags, c *command, args []string, opts *subCommandOpts) error {
	if _, err := verifyICEConfigFlag(flags.ice_config); err != nil {
		return err
	}
	if len(args) > 0 && flags.host == "" {
		return fmt.Errorf("missing host for devices: %v", args)
	}
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c.Command)
	if err != nil {
		return err
	}
	var cvds []RemoteCVDLocator
	for _, d := range args {
		cvds = append(cvds, RemoteCVDLocator{
			ServiceRootEndpoint: service.RootURI(),
			Host:                flags.host,
			WebRTCDeviceID:      d,
		})
	}
	// Find the user's cvds if they didn't specify any.
	if len(cvds) == 0 {
		var hosts []*RemoteHost
		if flags.host == "" {
			hosts, err = listCVDs(service, opts.InitialConfig.ConnectionControlDirExpanded())
		} else {
			hosts, err = listCVDsSingleHost(
				service, opts.InitialConfig.ConnectionControlDirExpanded(), flags.host)
		}
		if err != nil {
			return err
		}
		// Only those that are not connected yet
		selectList := flattenCVDs(hosts)
		selectList = filterSlice(selectList, func(cvd *RemoteCVD) bool { return cvd.ConnStatus == nil })
		// Confirmation is only necessary when the user didn't specify devices.
		if len(selectList) > 1 && !flags.skipConfirmation {
			toStr := func(c *RemoteCVD) string {
				return fmt.Sprintf("%s/%s", c.Host, c.WebRTCDeviceID)
			}
			selectList, err = PromptSelectionFromSlice(c, selectList, toStr, AllowAll)
			if err != nil {
				// A failure to read user input cancels the entire command.
				return err
			}
		}
		cvds = make([]RemoteCVDLocator, len(selectList))
		for idx, e := range selectList {
			cvds[idx] = e.RemoteCVDLocator
		}
	}

	var merr error
	connChs := make([]chan ConnStatus, len(cvds))
	errChs := make([]chan error, len(cvds))
	for i, cvd := range cvds {
		// These channels have a buffer length of 0 to ensure the send operation blocks
		// until the message is received by the other side. This ensures the select
		// operation on the receiving side will not unblock accidentally with the
		// closing of the other channel.
		connChs[i] = make(chan ConnStatus)
		errChs[i] = make(chan error)
		go func(connCh chan ConnStatus, errCh chan error, cvd RemoteCVDLocator) {
			defer close(connCh)
			defer close(errCh)
			status, err := ConnectDevice(cvd.Host, cvd.WebRTCDeviceID, flags.ice_config, flags.connectAgent, c, opts)
			if err != nil {
				errCh <- fmt.Errorf("failed to connect to %q on %q: %w", cvd.WebRTCDeviceID, cvd.Host, err)
			} else {
				connCh <- *status
			}
		}(connChs[i], errChs[i], cvd)
	}

	for i, cvd := range cvds {
		select {
		case status := <-connChs[i]:
			printConnection(c, cvd, status)
		case err := <-errChs[i]:
			merr = multierror.Append(merr, err)
		}
	}
	return merr
}

func verifyICEConfigFlag(v string) (*wclient.ICEConfig, error) {
	if v == "" {
		return nil, nil
	}
	c, err := loadICEConfigFromFile(v)
	if err != nil {
		return nil, fmt.Errorf("invalid --%s flag value: %w", iceConfigFlag, err)
	}
	return c, nil
}

func printConnection(c *command, cvd RemoteCVDLocator, status ConnStatus) {
	state := status.ADB.State
	if status.ADB.Port > 0 {
		state = fmt.Sprintf("127.0.0.1:%d", status.ADB.Port)
	}
	c.Printf("%s/%s: %s\n", cvd.Host, cvd.WebRTCDeviceID, state)
}

func buildAgentCmdArgs(flags *ConnectFlags, device string, connectAgent string) []string {
	args := []string{connectAgent, device}
	return append(args, flags.AsArgs()...)
}

// Letting the process be a proxy server for establishing the connection.
func forwardProxy(socketPath, proxyAddr, adbAddr string, adbServerProxy ADBServerProxy) error {
	// Make connection towards [host_ip_address]:[cuttlefish_instance_adb_port]
	var dialer proxy.Dialer
	if proxyAddr != "" {
		proxyUrl, err := url.Parse(proxyAddr)
		if err != nil {
			return fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		if proxyUrl.Scheme != "socks5" {
			return fmt.Errorf("scheme of proxy URL is not socks5. actual: %s", proxyUrl.Scheme)
		}
		dialer, err = proxy.SOCKS5("tcp", proxyUrl.Host, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to create proxy dialer: %w", err)
		}
	} else {
		dialer = proxy.Direct
	}
	remoteConn, err := dialer.Dial("tcp", adbAddr)
	if err != nil {
		return fmt.Errorf("failed to dial remote port: %w", err)
	}
	defer remoteConn.Close()

	// Create a file socket to establish ADB connection
	localListener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create unix socket: %w", err)
	}
	defer localListener.Close()

	// Enroll signal channel to handle SIGINT and SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Initialize ADB connection
	errorCh := make(chan error)
	go func() {
		if err := adbServerProxy.ConnectWithLocalFileSystem(socketPath); err != nil {
			errorCh <- fmt.Errorf("failed to connect adb: %w", err)
		}
	}()
	defer adbServerProxy.DisconnectWithLocalFileSystem(socketPath)

	// Listener is accepted by ADB connection.
	startLocalToRemoteCh := make(chan net.Conn)
	startRemoteToLocalCh := make(chan net.Conn)
	go func() {
		localConn, err := localListener.Accept()
		if err != nil {
			errorCh <- fmt.Errorf("failed to accept connection: %w", err)

		} else {
			startLocalToRemoteCh <- localConn
			startRemoteToLocalCh <- localConn
		}
	}()

	// After acception, start data transfer to do the role proxying ADB connection.
	go func() {
		localConn := <-startLocalToRemoteCh
		if _, err := io.Copy(remoteConn, localConn); err != nil {
			errorCh <- fmt.Errorf("no longer able to copy data from local to remote: %w", err)
		}
	}()
	go func() {
		localConn := <-startRemoteToLocalCh
		if _, err := io.Copy(localConn, remoteConn); err != nil {
			errorCh <- fmt.Errorf("no longer able to copy data from remote to local: %w", err)
		}
	}()

	// Wait for channels notifying the termination.
	var chErr error
	select {
	case <-sigCh:
	case chErr = <-errorCh:
	}

	// Remove the socket file before the termination of this process.
	if err := os.Remove(socketPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			chErr = multierror.Append(chErr, fmt.Errorf("failed to cleanup existing proxy socket: %w", err))
		}
	}
	return chErr
}

// Handler for the webrtc agent command. This is not meant to be called by the
// user directly, but instead is started by the open command.
// The process runs as a background process of runConnectCommand, for proxying
// connection to the ADB port located remotely.
func runConnectionProxyAgentCommand(flags *ConnectFlags, c *command, args []string, opts *subCommandOpts) error {
	if len(args) > 1 {
		return fmt.Errorf("connection agent only supports a single device, received: %v", args)
	}
	if len(args) == 0 {
		return errors.New("missing device")
	}
	device := args[0]
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c.Command)
	if err != nil {
		return err
	}
	controlDir := opts.InitialConfig.ConnectionControlDirExpanded()

	// Retrieving IP address and port of ADB connection
	host, err := findHost(service, flags.host)
	if err != nil {
		return fmt.Errorf("failed to find host")
	}
	if host.Docker == nil {
		return errors.New("instance type should be Docker")
	}
	cvd, err := findCVD(service, controlDir, flags.host, device)
	if err != nil {
		return fmt.Errorf("failed to find cvd: %w", err)
	}
	_, port, err := net.SplitHostPort(cvd.ADBSerial)
	if err != nil {
		return fmt.Errorf("failed to parse port from ADB serial: %w", err)
	}
	if port == "" {
		return errors.New("failed to find port")
	}
	adbAddress := net.JoinHostPort(host.Docker.IPAddress, port)
	socketPath := GetProxySocketPath(controlDir, flags.host, device)

	return forwardProxy(socketPath, flags.Proxy, adbAddress, opts.ADBServerProxy)
}

// Handler for the webrtc agent command. This is not meant to be called by the
// user directly, but instead is started by the open command.
// The process starts executing in the foreground, with its stderr connected to
// the terminal. If an error occurs the process exits with a non-zero exit code
// and the error is printed to stderr. If the connection is successfully
// established, the process closes all its standard IO channels and continues
// execution in the background. Any errors detected when the process is in
// background are written to the log file instead.
func runConnectionWebrtcAgentCommand(flags *ConnectFlags, c *command, args []string, opts *subCommandOpts) error {
	localICEConfig, err := verifyICEConfigFlag(flags.ice_config)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return fmt.Errorf("connection agent only supports a single device, received: %v", args)
	}
	if len(args) == 0 {
		return fmt.Errorf("missing device")
	}
	device := args[0]
	service, err := opts.ServiceBuilder(flags.ServiceFlags, c.Command)
	if err != nil {
		return err
	}

	devSpec := RemoteCVDLocator{
		ServiceRootEndpoint: service.RootURI(),
		Host:                flags.host,
		WebRTCDeviceID:      device,
	}

	controlDir := opts.InitialConfig.ConnectionControlDirExpanded()
	ret, err := FindOrConnect(controlDir, devSpec, service, localICEConfig)
	if err != nil {
		return err
	}
	if ret.Error != nil {
		// A connection was found or created, but a non-fatal error occurred.
		c.PrintErrln(ret.Error)
	}

	// The agent's only output is the port
	output, err := json.Marshal(ret.Status)
	if err != nil {
		c.PrintErrf("Failed to encode connection status: %v\n", err)
	} else {
		c.Println(string(output))
	}

	// Ask ADB server to connect even if the connection to the device already exists.
	if err := opts.ADBServerProxy.Connect(ret.Status.ADB.Port); err != nil {
		c.PrintErrf("Failed to connect ADB to device %q: %v\n", device, err)
	}

	if ret.Controller == nil {
		// A connection already exists, this process is done.
		return nil
	}

	// Signal the caller that the agent is moving to the background by closing
	// the command's standard IO channels.
	if cin, ok := c.InOrStdin().(io.Closer); ok {
		cin.Close()
	}
	if cout, ok := c.OutOrStdout().(io.Closer); ok {
		cout.Close()
	}
	if cerr, ok := c.ErrOrStderr().(io.Closer); ok {
		cerr.Close()
	}

	ret.Controller.Run()

	if err := opts.ADBServerProxy.Disconnect(ret.Status.ADB.Port); err != nil {
		// The command's Err is already closed, use the controller's logger instead
		ret.Controller.logger.Printf("Failed to disconnect ADB: %v\n", err)
	}

	// There is no point in returning error codes from a background process, errors
	// will be written to the log file instead.
	return nil
}

func connectADBWebSocketDirect(serviceURL string, device string) (*websocket.Conn, error) {
	// Connect to ADB proxy WebSocket directly by using serviceURL as Operator URL.
	//   wss://127.0.0.1:1443/devices/cvd-1/adb
	url, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL %s: %w", serviceURL, err)
	}
	switch p := &url.Scheme; *p {
	case "https":
		*p = "wss"
	case "http":
		*p = "ws"
	default:
		return nil, fmt.Errorf("Unknown scheme %s", *p)
	}
	url = url.JoinPath("devices", device, "adb")
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	wsConn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect WebSocket %s: %w", url.String(), err)
	}
	return wsConn, nil
}

func runConnectionWebSocketAgentCommand(flags *ConnectFlags, c *command, args []string, opts *subCommandOpts) error {
	// Open local TCP port that ADB server connects to and make ADB server connect
	// to the port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		c.PrintErrf("Failed to listen ADB socket: %v", err)
		return err
	}
	adbPort := l.Addr().(*net.TCPAddr).Port
	go func() {
		if err := opts.ADBServerProxy.Connect(adbPort); err != nil {
			c.PrintErrf("Failed to connect ADB to device: %v\n", err)
		}
	}()
	tcpConn, err := l.Accept()
	if err != nil {
		c.PrintErrf("Failed to accept ADB socket: %v", err)
		return err
	}
	defer tcpConn.Close()

	device := args[0]
	var wsConn *websocket.Conn
	if flags.host == "none" {
		// Connect to the ADB WebSocket directly using ServiceURL and device.
		// This is for operator only scenario.
		wsConn, err = connectADBWebSocketDirect(flags.ServiceURL, device)
		if err != nil {
			return err
		}
	} else {
		// Connect to the ADB WebSocket using Host Orchestrator service client.
		// This is for normal scenario (HO behind CO).
		service, err := opts.ServiceBuilder(flags.ServiceFlags, c.Command)
		if err != nil {
			return err
		}
		wsConn, err = service.HostService(flags.host).ConnectADBWebSocket(device)
		if err != nil {
			return err
		}
	}

	// Send connect result to the parent
	result := ConnStatus{
		ADB: ForwarderState{
			Port:  adbPort,
			State: StateAsStr(FwdConnected),
		},
	}
	output, err := json.Marshal(result)
	if err != nil {
		return err
	} else {
		c.Println(string(output))
	}

	// Close all standard IOs to make agent as daemon
	if cin, ok := c.InOrStdin().(io.Closer); ok {
		cin.Close()
	}
	if cout, ok := c.OutOrStdout().(io.Closer); ok {
		cout.Close()
	}
	if cerr, ok := c.ErrOrStderr().(io.Closer); ok {
		cerr.Close()
	}

	// Redirect WebSocket to ADB TCP port
	wsWrapper := &wsIoWrapper{
		wsConn: wsConn,
		pos:    0,
		buf:    nil,
	}
	defer wsWrapper.Close()
	go func() {
		io.Copy(wsWrapper, tcpConn)
		wsWrapper.Close()
	}()
	io.Copy(tcpConn, wsWrapper)
	return nil
}

// Wrapper for implementing io.ReadWriteCloser of websocket.Conn
type wsIoWrapper struct {
	wsConn *websocket.Conn
	pos    int
	buf    []byte
}

var _ io.ReadWriteCloser = (*wsIoWrapper)(nil)

func (w *wsIoWrapper) Read(buf []byte) (int, error) {
	if w.buf == nil || w.pos >= len(w.buf) {
		_, readBuf, err := w.wsConn.ReadMessage()
		if err != nil {
			return 0, err
		}
		w.buf = readBuf
		w.pos = 0
	}
	nRead := copy(buf[:], w.buf[w.pos:])
	w.pos += nRead
	return nRead, nil
}

func (w *wsIoWrapper) Write(buf []byte) (int, error) {
	err := w.wsConn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return 0, err
	}
	return len(buf), nil
}

func (w *wsIoWrapper) Close() error {
	return w.wsConn.Close()
}

func runDisconnectCommand(flags *ConnectFlags, c *command, args []string, opts *subCommandOpts) error {
	if len(args) > 0 && flags.host == "" {
		return fmt.Errorf("missing host for devices: %v", args)
	}
	controlDir := opts.InitialConfig.ConnectionControlDirExpanded()
	var statuses map[RemoteCVDLocator]ConnStatus
	var merr error
	if flags.host != "" {
		statuses, merr = listCVDConnectionsByHost(controlDir, flags.host)
	} else {
		statuses, merr = listCVDConnections(controlDir)
	}
	if len(statuses) == 0 {
		return fmt.Errorf("no connections found")
	}
	// Restrict the list of connections to those specified as arguments
	if len(args) > 0 {
		devices := make(map[string]bool)
		for _, a := range args {
			devices[a] = true
		}
		statuses = filterMap(statuses, func(cvd RemoteCVDLocator, s ConnStatus) bool {
			if devices[cvd.WebRTCDeviceID] {
				delete(devices, cvd.WebRTCDeviceID)
				return true
			}
			return false
		})
		for device := range devices {
			merr = multierror.Append(merr, fmt.Errorf("connection not found for %q", device))
		}
	}
	if len(statuses) > 1 && !flags.skipConfirmation {
		var err error
		statuses, err = promptConnectionSelection(statuses, c)
		if err != nil {
			// A failure to read user input cancels the entire command.
			return err
		}
	}
	for cvd, dev := range statuses {
		if err := DisconnectCVD(controlDir, cvd, dev); err != nil {
			multierror.Append(merr, err)
			continue
		}
		c.Printf("%s/%s: disconnected\n", cvd.Host, cvd.WebRTCDeviceID)
	}
	return merr
}

func runGetConfigCommand(c *cobra.Command, args []string, cfg Config) error {
	if len(args) == 0 {
		return errors.New("missing config property name")
	}
	if len(args) > 1 {
		return errors.New("use one config property at a time")
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	m := interface{}(nil)
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	v, err := jsonpath.Get("$."+args[0], m)
	if err != nil {
		return fmt.Errorf("invalid config property name: %w", err)
	}
	c.Println(v)
	return nil
}

func promptConnectionSelection(devices map[RemoteCVDLocator]ConnStatus, c *command) (map[RemoteCVDLocator]ConnStatus, error) {
	c.PrintErrln("Multiple connections match:")
	toStr := func(cvd RemoteCVDLocator, d ConnStatus) string {
		return fmt.Sprintf("%s %s", cvd.Host, cvd.WebRTCDeviceID)
	}
	return PromptSelectionFromMap(c, devices, toStr, AllowAll)
}

func filterSlice[T any](s []T, pred func(T) bool) []T {
	var r []T
	for _, t := range s {
		if pred(t) {
			r = append(r, t)
		}
	}
	return r
}

func filterMap[K comparable, T any](m map[K]T, pred func(K, T) bool) map[K]T {
	r := make(map[K]T)
	for k, t := range m {
		if pred(k, t) {
			r[k] = t
		}
	}
	return r
}

type serviceBuilder func(flags *ServiceFlags, c *cobra.Command) (client.Service, error)

const chunkSizeBytes = 16 * 1024 * 1024

func buildServiceBuilder(builder client.ServiceBuilder, config *Config) serviceBuilder {
	return func(flags *ServiceFlags, c *cobra.Command) (client.Service, error) {
		proxyURL := flags.Proxy
		var dumpOut io.Writer = io.Discard
		if flags.Verbose {
			dumpOut = c.ErrOrStderr()
		}
		opts := &client.ServiceOptions{
			RootEndpoint:   buildServiceRootEndpoint(flags.ServiceURL, flags.Zone),
			ProxyURL:       proxyURL,
			DumpOut:        dumpOut,
			ErrOut:         c.ErrOrStderr(),
			ChunkSizeBytes: chunkSizeBytes,
		}
		srvConfig := config.DefaultService()
		if flags.Service != "" {
			srvConfig, _ = config.Services[flags.Service]
		}
		authnConfig := srvConfig.Authn
		if authnConfig != nil {
			if authnConfig.OIDCToken != nil && authnConfig.HTTPBasicAuthn != nil {
				return nil, fmt.Errorf("should only set one authentication option")
			}
			opts.Authn = &client.AuthnOpts{}
			if authnConfig.OIDCToken != nil {
				content, err := os.ReadFile(authnConfig.OIDCToken.TokenFile)
				if err != nil {
					return nil, fmt.Errorf("failed loading oidc token: %w", err)
				}
				value := strings.TrimSuffix(string(content), "\n")
				opts.Authn.OIDCToken = &client.OIDCToken{
					Value: value,
				}
			} else if authnConfig.HTTPBasicAuthn != nil {
				switch authnConfig.HTTPBasicAuthn.UsernameSrc {
				case UnixUsernameSrc:
					user, err := user.Current()
					if err != nil {
						return nil, fmt.Errorf("unable to get unix username for http basic authn: %w", err)
					}
					opts.Authn.HTTPBasic = &client.HTTPBasic{
						Username: user.Username,
					}
				default:
					return nil, fmt.Errorf("invalid http basic authn UsernameSrc type: %s", authnConfig.HTTPBasicAuthn.UsernameSrc)
				}
			}
		}
		opts.InjectBuildAPICreds = srvConfig.BuildAPICredentialsSource == "injected"
		return builder(opts)
	}
}

func buildServiceRootEndpoint(serviceURL, zone string) string {
	const version = "v1"
	return client.BuildRootEndpoint(serviceURL, version, zone)
}

// Prints out state changes.
//
// Only use this printer to print user friendly state changes, this is not a logger.
type statePrinter struct {
	// Writer to print the messages to. Visual features won't be displayed if not linked to an interactive terminal.
	Out io.Writer

	// If true, visual features like colors and animations won't be displayed.
	visualsOn bool
}

func newStatePrinter(out io.Writer, verbose bool) *statePrinter {
	visualsOn := false
	if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) && !verbose {
		visualsOn = true
	}
	return &statePrinter{Out: out, visualsOn: visualsOn}
}

func (p *statePrinter) Print(msg string) {
	p.print(msg, statePrinterState{Done: false})
}

func (p *statePrinter) PrintDone(msg string, err error) {
	p.print(msg, statePrinterState{Done: true, DoneErr: err})
}

type statePrinterState struct {
	Done bool
	// Only relevant if Done is true.
	DoneErr error
}

func (p *statePrinter) print(msg string, state statePrinterState) {
	prefix := ""
	if p.visualsOn {
		// Use cursor movement characters for an interactive experience when visuals are on.
		prefix = "\r\033[K"
	}
	result := prefix + toFixedLength(msg, 50, '.') + strings.Repeat(".", 3) + " "
	if state.Done {
		if state.DoneErr == nil {
			result += "OK"
		} else {
			result += "Failed"
		}
	}
	if !p.visualsOn || state.Done {
		result += "\n"
	}
	fmt.Fprint(p.Out, result)
}

// Return prefix or append a filling character building a string of fixed length.
func toFixedLength(v string, l int, filling rune) string {
	if len(v) > l {
		return v[:l]
	} else {
		return v + strings.Repeat(string(filling), l-len(v))
	}
}

type acceleratorConfig struct {
	Count int
	Type  string
}

// Values should follow the pattern: `type=[TYPE],count=[COUNT]`, i.e: `type=nvidia-tesla-p100,count=1`.
func parseAcceleratorFlag(values []string) ([]acceleratorConfig, error) {
	singleValueParser := func(value string) (*acceleratorConfig, error) {
		sanitized := strings.Join(strings.Fields(value), "")
		cStrs := strings.Split(sanitized, ",")
		if len(cStrs) != 2 {
			return nil, fmt.Errorf("invalid accelerator token: %q", value)
		}
		config := &acceleratorConfig{}
		for _, pair := range cStrs {
			keyValue := strings.Split(pair, "=")
			if len(keyValue) != 2 {
				return nil, fmt.Errorf("invalid accelerator `[key]=[value]` token: %q", keyValue)
			}
			switch key := keyValue[0]; key {
			case "count":
				i, err := strconv.Atoi(keyValue[1])
				if err != nil {
					return nil, fmt.Errorf("invalid accelerator count value: %w", err)
				}
				config.Count = i
			case "type":
				config.Type = keyValue[1]
			default:
				return nil, fmt.Errorf("unknown accelerator key: %q", key)
			}
		}
		if config.Count == 0 {
			return nil, fmt.Errorf("missing accelerator count")
		}
		if config.Type == "" {
			return nil, fmt.Errorf("missing accelerator type")
		}
		return config, nil
	}
	result := []acceleratorConfig{}
	for _, v := range values {
		c, err := singleValueParser(v)
		if err != nil {
			return nil, err
		}
		result = append(result, *c)
	}
	return result, nil
}

type Flags interface {
	Update(s *Service)
}

type preRunEFn func(cmd *cobra.Command, args []string) error

func preRunE(flags Flags, name *string, c *Config) preRunEFn {
	return func(cmd *cobra.Command, args []string) error {
		if name == nil || *name == "" {
			return nil
		}
		val, ok := c.Services[*name]
		if !ok {
			return fmt.Errorf("service %s not found", *name)
		}
		flags.Update(val)
		return nil
	}
}
