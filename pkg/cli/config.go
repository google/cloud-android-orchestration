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
	MachineType        string   `json:"machine_type,omitempty"`
	MinCPUPlatform     string   `json:"min_cpu_platform,omitempty"`
	BootDiskSizeGB     int64    `json:"boot_disk_size_gb,omitempty"`
	AcceleratorConfigs []string `json:"accelerator_configs,omitempty"`
}

type HostConfig struct {
	GCP GCPHostConfig `json:"gcp,omitempty"`
}

type AuthnConfig struct {
	OIDCToken      *OIDCTokenConfig      `json:"oidc_token,omitempty"`
	HTTPBasicAuthn *HTTPBasicAuthnConfig `json:"http_basic_authn,omitempty"`
}

type OIDCTokenConfig struct {
	TokenFile string `json:"token_file,omitempty"`
}

type UsernameSrcType string

const UnixUsernameSrc UsernameSrcType = "unix"

type HTTPBasicAuthnConfig struct {
	UsernameSrc UsernameSrcType `json:"username_src,omitempty"`
}

type Config struct {
	// Default service, service to be used in case none other was selected.
	SystemDefaultService string `json:"system_default_service,omitempty"`
	// [OPTIONAL] If set, it overrides the `SystemDefaultService` parameter.
	UserDefaultService   string              `json:"user_default_service,omitempty"`
	Services             map[string]*Service `json:"services,omitempty"`
	ConnectionControlDir string              `json:"connection_control_dir,omitempty"`
	KeepLogFilesDays     int                 `json:"keep_log_files_days,omitempty"`
}

type Service struct {
	ServiceURL                string       `json:"service_url,omitempty"`
	Zone                      string       `json:"zone,omitempty"`
	Proxy                     string       `json:"proxy,omitempty"`
	BuildAPICredentialsSource string       `json:"build_api_credentials_source,omitempty"`
	Host                      HostConfig   `json:"host,omitempty"`
	Authn                     *AuthnConfig `json:"authn,omitempty"`
	ConnectAgent              string       `json:"connect_agent,omitempty"`
}

func (c *Config) DefaultService() *Service {
	if c.UserDefaultService != "" {
		return c.Services[c.UserDefaultService]
	} else if c.SystemDefaultService != "" {
		return c.Services[c.SystemDefaultService]
	}
	return &Service{}
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

func LoadConfig(sysSrc, userSrc string, c *Config) error {
	// Load system configuration
	if err := loadConfig(sysSrc, c); err != nil {
		return fmt.Errorf("error loading system configuration: %w", err)
	}
	if userSrc == "" {
		return nil
	}
	sysSrvcs := c.Services
	c.Services = make(map[string]*Service)
	// Load user configuration
	if err := loadConfig(userSrc, c); err != nil {
		return fmt.Errorf("error loading user configuration: %w", err)
	}
	// Combine services configurations.
	for k, v := range sysSrvcs {
		c.Services[k] = v
	}
	return nil
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
		"Services": map[string]any{
			"acloud": map[string]any{
				"Zone": ac.Zone,
				"Host": map[string]any{
					"GCP": map[string]any{
						"MachineType": ac.MachineType,
					},
				},
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

func loadConfig(src string, out *Config) error {
	if src == "" {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	decoder := toml.NewDecoder(bytes.NewReader(b))
	// Fail if there is some unknown configuration. This is better than silently
	// ignoring a (perhaps misspelled) config entry.
	decoder.Strict(true)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}
