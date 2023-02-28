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
	"github.com/google/cloud-android-orchestration/pkg/client"
)

const configPathVar = "CVDR_CONFIG_PATH"

func readConfig(config *cli.Config) error {
	configPath := os.Getenv(configPathVar)
	if configPath == "" {
		// No config file provided
		return nil
	}
	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("Error opening config file: %w", err)
	}
	defer configFile.Close()

	if err := cli.ParseConfigFile(config, configFile); err != nil {
		return fmt.Errorf("Error parsing config file: %w", err)
	}
	return nil
}

type cmdRunner struct{}

func (_ *cmdRunner) StartBgCommand(args ...string) ([]byte, error) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create pipe: %w", err)
	}
	defer pipe.Close()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Unable to start command: %w", err)
	}
	defer cmd.Process.Release()
	output, err := io.ReadAll(pipe)
	if err != nil {
		return nil, fmt.Errorf("Error reading command output: %v", err)
	}
	return output, nil
}

func main() {
	config := cli.DefaultConfig()
	// Overrides relevant defaults with values set in config file.
	if err := readConfig(&config); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	opts := &cli.CommandOptions{
		IOStreams:      cli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Args:           os.Args[1:],
		ServiceBuilder: client.NewService,
		InitialConfig:  config,
		CommandRunner:  &cmdRunner{},
	}

	if err := cli.NewCVDRemoteCommand(opts).Execute(); err != nil {
		os.Exit(1)
	}
}
