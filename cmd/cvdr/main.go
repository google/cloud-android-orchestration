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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/google/cloud-android-orchestration/pkg/cli"
	"github.com/google/cloud-android-orchestration/pkg/client"
)

const (
	envVarSystemConfigPath = "CVDR_SYSTEM_CONFIG_PATH"
	// User config values overrieds system config values.
	envVarUserConfigPath = "CVDR_USER_CONFIG_PATH"
)

func loadInitialConfig() (*cli.Config, error) {
	config := cli.BaseConfig()
	if path, ok := os.LookupEnv(envVarSystemConfigPath); ok {
		path = cli.ExpandPath(path)
		if err := cli.LoadConfigFile(path, config); err != nil {
			return nil, err
		}
	}
	if path, ok := os.LookupEnv(envVarUserConfigPath); ok {
		path = cli.ExpandPath(path)
		if _, err := os.Stat(path); err == nil {
			if err := cli.LoadConfigFile(path, config); err != nil {
				return nil, err
			}
		} else if errors.Is(err, os.ErrNotExist) {
			// TODO: Create a new one importing relevant acloud config parameters.
		} else {
			return nil, fmt.Errorf("Invalid user config file path: %w", err)
		}
	}
	return config, nil
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
	config, err := loadInitialConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	opts := &cli.CommandOptions{
		IOStreams:      cli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Args:           os.Args[1:],
		ServiceBuilder: client.NewService,
		InitialConfig:  *config,
		CommandRunner:  &cmdRunner{},
		ADBServerProxy: &cli.ADBServerProxyImpl{},
	}

	if err := cli.NewCVDRemoteCommand(opts).Execute(); err != nil {
		os.Exit(1)
	}
}
