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
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

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
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
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

const (
	verboseFlag = "verbose"
)

type subCommandFlags struct {
	Verbose bool
}

type subCommandOptions struct {
	HTTPClient *http.Client
	BaseURL    string
	Verbose    bool
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
	del := &cobra.Command{
		Use:   "delete <foo> <bar> <baz>",
		Short: "Delete hosts.",
		RunE: func(c *cobra.Command, args []string) error {
			return runSubCommand(c, args, runDeleteHostsCommand)
		},
	}
	host := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	subCommandFlags := &subCommandFlags{}
	host.PersistentFlags().BoolVarP(&subCommandFlags.Verbose, verboseFlag, "v", false, "Be verbose.")
	host.AddCommand(create)
	host.AddCommand(list)
	host.AddCommand(del)
	return host
}

func runSubCommand(c *cobra.Command, args []string, runner subCommandRunner) error {
	httpClient := &http.Client{}
	proxyURL := c.InheritedFlags().Lookup(httpProxyFlag).Value.String()
	// Handles http proxy
	if proxyURL != "" {
		proxyUrl, err := url.Parse(proxyURL)
		if err != nil {
			return err
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	}
	opts := &subCommandOptions{
		HTTPClient: httpClient,
		BaseURL:    buildBaseURL(c),
		Verbose:    c.InheritedFlags().Lookup(verboseFlag).Changed,
	}
	return runner(c, args, opts)

}

func runCreateHostCommand(c *cobra.Command, _ []string, opts *subCommandOptions) error {
	var op apiv1.Operation
	reqOpts := doRequestOpts{
		Client:  opts.HTTPClient,
		Verbose: opts.Verbose,
		ErrOut:  c.ErrOrStderr(),
	}
	body := apiv1.CreateHostRequest{HostInstance: &apiv1.HostInstance{}}
	if err := doRequest("POST", opts.BaseURL+"/hosts", &body, &op, &reqOpts); err != nil {
		return err
	}
	url := opts.BaseURL + "/operations/" + op.Name + "/wait"
	if err := doRequest("POST", url, nil, &op, &reqOpts); err != nil {
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
	reqOpts := doRequestOpts{
		Client:  opts.HTTPClient,
		Verbose: opts.Verbose,
		ErrOut:  c.ErrOrStderr(),
	}
	if err := doRequest("GET", opts.BaseURL+"/hosts", nil, &res, &reqOpts); err != nil {
		return err
	}
	for _, ins := range res.Items {
		c.Printf("%s\n", ins.Name)
	}
	return nil
}

func runDeleteHostsCommand(c *cobra.Command, args []string, opts *subCommandOptions) error {
	reqOpts := doRequestOpts{
		Client:  opts.HTTPClient,
		Verbose: opts.Verbose,
		ErrOut:  c.ErrOrStderr(),
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var merr error
	for _, arg := range args {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			url := opts.BaseURL + "/hosts/" + name
			if err := doRequest("DELETE", url, nil, nil, &reqOpts); err != nil {
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

type doRequestOpts struct {
	Client  *http.Client
	Verbose bool
	ErrOut  io.Writer
}

// It either populates the passed response payload reference and returns nil error or returns an error.
// For responses with non-2xx status code an error will be returned.
func doRequest(method, url string, reqpl, respl interface{}, opts *doRequestOpts) error {
	var body io.Reader
	if reqpl != nil {
		json, err := json.Marshal(reqpl)
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
	if opts.Verbose {
		if err := dumpRequest(req, opts.ErrOut); err != nil {
			return err
		}
	}
	res, err := opts.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if opts.Verbose {
		if err := dumpResponse(res, opts.ErrOut); err != nil {
			return err
		}
	}
	dec := json.NewDecoder(res.Body)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		// DELETE responses do not have a body.
		if method == "DELETE" {
			return &apiCallError{&apiv1.Error{Message: res.Status}}
		}
		errpl := new(apiv1.Error)
		if err := dec.Decode(errpl); err != nil {
			return err
		}
		return &apiCallError{errpl}
	}
	if respl != nil {
		if err := dec.Decode(respl); err != nil {
			return err
		}
	}
	return nil
}

func dumpRequest(r *http.Request, w io.Writer) error {
	dump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", dump)
	return nil
}

func dumpResponse(r *http.Response, w io.Writer) error {
	dump, err := httputil.DumpResponse(r, true)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", dump)
	return nil
}
