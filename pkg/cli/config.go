// Copyright 2023 Google LLC
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
	"io"

	toml "github.com/pelletier/go-toml"
)

type GCPHostConfig struct {
	DefaultMachineType    string
	DefaultMinCPUPlatform string
}

type HostConfig struct {
	GCP GCPHostConfig
}

type Config struct {
	DefaultServiceURL string
	DefaultZone       string
	DefaultHTTPProxy  string
	ADBControlDir     string
	Host              HostConfig
}

func ParseConfig(config *Config, confFile io.Reader) error {
	decoder := toml.NewDecoder(confFile)
	// Fail if there is some unknown configuration. This is better than silently
	// ignoring a (perhaps mispelled) config entry.
	decoder.Strict(true)
	return decoder.Decode(config)
}
