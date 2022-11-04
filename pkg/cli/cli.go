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
)

type configFlags struct {
	ServiceURL string
}

func NewCVDRemoteCommandWithArgs(o *CommandOptions) *CVDRemoteCommand {
	var configFlags = new(configFlags)
	hostCmd := &cobra.Command{
		Use:   "host",
		Short: "Work with hosts",
	}
	hostCmd.AddCommand(newCreateHostCommand())
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
	// Do not show a `help` command, users have always the `-h` and `--help` flags for help purpose.
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.AddCommand(hostCmd)
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

func newCreateHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a host.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateHostCommand(c)
		},
	}
	return cmd
}

func runCreateHostCommand(c *cobra.Command) error {
	serviceURL := c.InheritedFlags().Lookup(serviceURLFlag).Value.String()
	baseURL := serviceURL + "/v1"
	req := &apiv1.CreateHostInstanceRequest{}
	body := &apiv1.CreateHostRequest{CreateHostInstanceRequest: req}
	var op apiv1.Operation
	client := &http.Client{}
	if err := doRequest(client, "POST", baseURL+"/hosts", body, &op); err != nil {
		return err
	}
	if err := doRequest(client, "POST", baseURL+"/operations/"+op.Name+"/wait", nil, &op); err != nil {
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

// It either populates the passed resPayload reference and returns nil error or returns an error.
// For responses with non-2xx status code an error will be returned.
func doRequest(
	client *http.Client, method, url string, reqPayload interface{}, resPayload interface{}) error {
	var body io.Reader
	if reqPayload != nil {
		json, err := json.Marshal(reqPayload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(json)
	}
	req, err := http.NewRequest(method, url, body)
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
