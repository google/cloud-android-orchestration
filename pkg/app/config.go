// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	toml "github.com/pelletier/go-toml/v2"
	"os"
)

const DefaultConfFile = "conf.toml"
const ConfFileEnvVar = "CONFIG_FILE"

type Config struct {
	AccountManager  AMConfig
	InstanceManager IMConfig
	Infra           InfraConfig
	Operations      OperationsConfig
}

type IMConfig struct {
	Type IMType
	GCP  *GCPIMConfig
}

type IMType string

const (
	UnixIMType IMType = "unix"
	GCEIMType  IMType = "GCP"
)

type GCPIMConfig struct {
	ProjectID string
	HostImage string
}

type AMConfig struct {
	Type AMType
}

type AMType string

const (
	UnixAMType AMType = "unix"
	GAEAMType  AMType = "GCP"
)

type InfraConfig struct {
	STUNServers []string
}

type OperationsConfig struct {
	CreateHostDisabled bool
}

func LoadConfig() (*Config, error) {
	confFile := os.Getenv(ConfFileEnvVar)
	if confFile == "" {
		confFile = "conf.toml"
	}
	file, err := os.Open(confFile)
	if err != nil {
		return nil, err
	}
	decoder := toml.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var cfg Config
	err = decoder.Decode(&cfg)
	return &cfg, err
}
