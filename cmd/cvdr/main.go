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
	"path/filepath"
	"strings"

	"github.com/google/cloud-android-orchestration/pkg/cli"
	"github.com/google/cloud-android-orchestration/pkg/client"
	"github.com/google/cloud-android-orchestration/pkg/metrics"

	"golang.org/x/term"
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
		_, statErr := os.Stat(path)
		if errors.Is(statErr, os.ErrNotExist) {
			imported, err := importAcloudConfig(path)
			if err != nil {
				return nil, err
			}
			if !imported {
				// Create empty user configuration file.
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0750); err != nil {
					return nil, fmt.Errorf("failed creating user config directory: %w", err)
				}
				f, err := os.Create(path)
				if err != nil {
					return nil, fmt.Errorf("failed creating user config file: %w", err)
				}
				f.Close()
			}
			statErr = nil
		}
		if statErr == nil {
			if err := cli.LoadConfigFile(path, config); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid user config file path: %w", statErr)
		}
	}
	return config, nil
}

func importAcloudConfig(dst string) (bool, error) {
	// Do not prompt acloud importing if not in a terminal.
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false, nil
	}
	// Create a new user configuration file importing existing acloud configuration.
	acPath := cli.ExpandPath("~/.config/acloud/acloud.config")
	if _, err := os.Stat(acPath); err == nil {
		const p = "No user configuration found, would you like to generate it by importing " +
			"your acloud configuration?"
		yes, err := cli.PromptYesOrNo(os.Stdout, os.Stdin, p)
		if err != nil {
			return false, err
		}
		if yes {
			if err := cli.ImportAcloudConfig(acPath, dst); err != nil {
				return false, fmt.Errorf("failed importing acloud config file: %w", err)
			}
			return true, nil
		}
	}
	return false, nil
}

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
	config, err := loadInitialConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	commandLine := strings.Join(os.Args, " ")
	if err := metrics.SendLaunchCommand(commandLine); err != nil {
		fmt.Fprintln(os.Stderr, commandLine)
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
