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

	"github.com/google/cloud-android-orchestration/pkg/cli"
	"golang.org/x/term"
)

const (
	envVarSystemConfigPath = "CVDR_SYSTEM_CONFIG_PATH"
	// User config values overrieds system config values.
	envVarUserConfigPath = "CVDR_USER_CONFIG_PATH"
	// It may ask for importing acloud config if cvdr user config is empty.
	acloudConfigPath = "~/.config/acloud/acloud.config"
)

func loadInitialConfig() (*cli.Config, error) {
	config := cli.BaseConfig()
	sysConfigSrc, userConfigSrc := "", ""
	if path, ok := os.LookupEnv(envVarSystemConfigPath); ok {
		sysConfigSrc = cli.ExpandPath(path)
	}
	if path, ok := os.LookupEnv(envVarUserConfigPath); ok {
		userConfigSrc = cli.ExpandPath(path)
		if err := createUserConfigIfNeeded(userConfigSrc); err != nil {
			return nil, err
		}
	}
	if err := cli.LoadConfig(sysConfigSrc, userConfigSrc, config); err != nil {
		return nil, err
	}
	return config, nil
}

func createUserConfigIfNeeded(dst string) error {
	if _, err := os.Stat(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("invalid user config file path: %w", err)
	} else if err == nil {
		return nil
	}
	imported, err := importAcloudConfig(dst)
	if err != nil {
		return fmt.Errorf("failed creating user config file with acloud config:%w", err)
	}
	if imported {
		return nil
	}
	// Create empty user configuration file.
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return fmt.Errorf("failed creating user config directory: %w", err)
	}
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed creating user config file: %w", err)
	}
	f.Close()
	return nil
}

func importAcloudConfig(ucPath string) (bool, error) {
	// Create a new user configuration file importing existing acloud configuration.
	acPath := cli.ExpandPath(acloudConfigPath)
	if _, err := os.Stat(acPath); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to stat acloud config file: %w", err)
	}
	// It doesn't import acloud config when user says no via prompt.
	if term.IsTerminal(int(os.Stdout.Fd())) {
		const p = "No user configuration found, would you like to generate it by importing " +
			"your acloud configuration?"
		yes, err := cli.PromptYesOrNo(os.Stdout, os.Stdin, p)
		if err != nil {
			return false, err
		}
		if !yes {
			return false, nil
		}
	}
	if err := cli.ImportAcloudConfig(acPath, ucPath); err != nil {
		return false, fmt.Errorf("failed importing acloud config file: %w", err)
	}
	return true, nil
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
