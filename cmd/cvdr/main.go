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

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/google/cloud-android-orchestration/pkg/cli"
)

type cmdRunner struct{}

func (*cmdRunner) StartBgCommand(args ...string) ([]byte, error) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}
	defer pipe.Close()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start command: %w", err)
	}
	defer cmd.Process.Release()
	output, err := io.ReadAll(pipe)
	if err != nil {
		return nil, fmt.Errorf("error reading command output: %v", err)
	}
	return output, nil
}

func main() {
	config, err := cli.LoadInitialConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	opts := &cli.CommandOptions{
		IOStreams:      cli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Args:           os.Args[1:],
		InitialConfig:  *config,
		CommandRunner:  &cmdRunner{},
		ADBServerProxy: &cli.ADBServerProxyImpl{},
	}

	if err := cli.NewCVDRemoteCommand(opts).Execute(); err != nil {
		os.Exit(1)
	}
}
