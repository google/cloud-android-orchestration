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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

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

func NewCVDRemoteCommand() *CVDRemoteCommand {
	opts := &CommandOptions{
		IOStreams: IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Args:      os.Args[1:],
	}
	return NewCVDRemoteCommandWithArgs(opts)
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

func NewCVDRemoteCommandWithArgs(o *CommandOptions) *CVDRemoteCommand {
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
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help
	// purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.AddCommand(newHostCommand())
	return &CVDRemoteCommand{rootCmd}
}

// ExecuteNoErrOutput is a version of Execute which returns the command error instead of
// printing it and do not exit in case of error.
func (c *CVDRemoteCommand) ExecuteNoErrOutput() error {
	return c.command.Execute()
}

func (c *CVDRemoteCommand) Execute() {
	if err := c.ExecuteNoErrOutput(); err != nil {
		c.command.PrintErrln(err)
		os.Exit(1)
	}
}

type opTimeoutError string

func (s opTimeoutError) Error() string {
	return fmt.Sprintf("waiting for operation %q timed out", string(s))
}

type apiCallError struct {
	Err *apiv1.Error
}

func (e *apiCallError) Error() string {
	return fmt.Sprintf("api call error %s: %s", e.Err.Code, e.Err.Message)
}

type subCommandOptions struct {
	BaseURL    string
	HTTPClient *http.Client
}

type subCommandRunner func(c *cobra.Command, args []string, opts *subCommandOptions) error

func newHostCommand() *cobra.Command {
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return runSubCommand(c, args, runCreateHostCommand)
		},
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "Lists hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runSubCommand(c, args, runListHostsCommand)
		},
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	host.AddCommand(create)
	host.AddCommand(list)
	return host
}

func runSubCommand(c *cobra.Command, args []string, runner subCommandRunner) error {
	proxyURL := c.InheritedFlags().Lookup(httpProxyFlag).Value.String()
	// Handles http proxy
	if proxyURL != "" {
		if _, err := url.Parse(proxyURL); err != nil {
			return err
		}
		os.Setenv("HTTP_PROXY", proxyURL)
	}
	opts := &subCommandOptions{
		BaseURL: buildBaseURL(c),
	}
	return runner(c, args, opts)

}

func runCreateHostCommand(c *cobra.Command, _ []string, opts *subCommandOptions) error {
	client := &http.Client{}
	req := &apiv1.CreateHostInstanceRequest{}
	body := &apiv1.CreateHostRequest{CreateHostInstanceRequest: req}
	var op apiv1.Operation
	if err := doRequest(client, "POST", opts.BaseURL+"/hosts", body, &op); err != nil {
		return err
	}
	url := opts.BaseURL + "/operations/" + op.Name + "/wait"
	if err := doRequest(client, "POST", url, nil, &op); err != nil {
		return err
	}
	if op.Result != nil && op.Result.Error != nil {
		err := &apiCallError{op.Result.Error}
		return err
	}
	if !op.Done {
		return opTimeoutError(op.Name)
	}
	var ins apiv1.HostInstance
	if err := json.Unmarshal([]byte(op.Result.Response), &ins); err != nil {
		return err
	}
	c.Printf("%s\n", ins.Name)
	return nil
}

func runListHostsCommand(c *cobra.Command, _ []string, opts *subCommandOptions) error {
	var res apiv1.ListHostsResponse
	if err := doRequest(&http.Client{}, "GET", opts.BaseURL+"/hosts", nil, &res); err != nil {
		return err
	}
	for _, ins := range res.Items {
		c.Printf("%s\n", ins.Name)
	}
	return nil
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

// It either populates the passed resPayload reference and returns nil error or returns an error.
// For responses with non-2xx status code an error will be returned.
func doRequest(client *http.Client, method, url string, reqPayload interface{}, resPayload interface{}) error {
	var body io.Reader
	if reqPayload != nil {
		json, err := json.Marshal(reqPayload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(json)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		errPayload := new(apiv1.Error)
		if err := dec.Decode(errPayload); err != nil {
			return err
		}
		return &apiCallError{errPayload}
	}
	if resPayload != nil {
		if err := dec.Decode(resPayload); err != nil {
			return err
		}
	}
	return nil
}
