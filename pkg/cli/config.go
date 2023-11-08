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
	"os"
	"path/filepath"
	"regexp"
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
		return fmt.Errorf("error reading config file: %w", err)
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

type AcloudConfig struct {
	Zone        string
	MachineType string
}

func ImportAcloudConfig(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading %q: %w", src, err)
	}
	ac, err := extractAcloudConfig(string(b))
	if err != nil {
		return fmt.Errorf("failed extracing acloud config: %w", err)
	}
	c := map[string]any{
		"Zone": ac.Zone,
		"Host": map[string]any{
			"GCP": map[string]any{
				"MachineType": ac.MachineType,
			},
		},
	}
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
		return fmt.Errorf("failed creating dir %q: %w", dstDir, err)
	}
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed creating file %q: %w", dst, err)
	}
	defer f.Close()
	encoder := toml.NewEncoder(f).Indentation("")
	if err = encoder.Encode(c); err != nil {
		return fmt.Errorf("failed encoding: %w", err)
	}
	return nil
}

// Extract relevant acloud config values.
// Acloud config is written in Proto Text Format https://protobuf.dev/reference/protobuf/textformat-spec/
func extractAcloudConfig(input string) (AcloudConfig, error) {
	extract := func(field, input string) (string, error) {
		re := regexp.MustCompile(`(^|\n)\s*` + field + `\s*:\s*\"([a-zA-Z0-9_-]+)\"`)
		matches := re.FindStringSubmatch(input)
		if len(matches) != 3 {
			return "", fmt.Errorf("value for field: %q not found", field)
		}
		return matches[2], nil
	}
	zone, err := extract("zone", input)
	if err != nil {
		return AcloudConfig{}, err
	}
	machineType, err := extract("machine_type", input)
	if err != nil {
		return AcloudConfig{}, err
	}
	result := AcloudConfig{
		Zone:        zone,
		MachineType: machineType,
	}
	return result, nil
}
