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
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml"
)

type GCPHostConfig struct {
	MachineType    string
	MinCPUPlatform string
}

type HostConfig struct {
	GCP GCPHostConfig
}

type Config struct {
	ServiceURL           string
	Zone                 string
	HTTPProxy            string
	ConnectionControlDir string
	KeepLogFilesDays     int
	Host                 HostConfig
}

func (c *Config) ConnectionControlDirExpanded() string {
	return ExpandPath(c.ConnectionControlDir)
}

func (c *Config) LogFilesDeleteThreshold() time.Duration {
	return time.Duration(c.KeepLogFilesDays*24) * time.Hour
}

func BaseConfig() *Config {
	return &Config{
		ConnectionControlDir: "~/.cvdr/connections",
		KeepLogFilesDays:     30, // A default is needed to not keep forever
	}
}

func LoadConfigFile(path string, c *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Error reading config file: %w", err)
	}
	decoder := toml.NewDecoder(bytes.NewReader(b))
	// Fail if there is some unknown configuration. This is better than silently
	// ignoring a (perhaps mispelled) config entry.
	decoder.Strict(true)
	return decoder.Decode(c)
}

func ExpandPath(path string) string {
	if !strings.Contains(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Unable to expand path %q: %v", path, err))
	}
	return strings.ReplaceAll(path, "~", home)
}

// Build a final configuration instance from different sources. Each config parameter will take
// the value from the last source where it's set.
func buildConfig(sources []io.Reader) (*Config, error) {
	c := BaseConfig()
	for _, s := range sources {
		decoder := toml.NewDecoder(s)
		decoder.Strict(true)
		if err := decoder.Decode(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
