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
	"os"

	toml "github.com/pelletier/go-toml"
)

const DefaultConfFile = "conf.toml"
const ConfFileEnvVar = "CONFIG_FILE"

type Config struct {
	WebStaticFilesPath string
	AccountManager     AMConfig
	SecretManager      SMConfig
	InstanceManager    IMConfig
	EncryptionService  ESConfig
	DatabaseService    DBConfig
	Infra              InfraConfig
	Operations         OperationsConfig
}

type IMConfig struct {
	Type IMType
	// The protocol the host orchestrator expects, either http or https
	HostOrchestratorProtocol          string
	AllowSelfSignedHostSSLCertificate bool
	GCP                               *GCPIMConfig
	UNIX                              *UNIXIMConfig
}

type IMType string

const (
	UnixIMType IMType = "unix"
	GCEIMType  IMType = "GCP"
)

type GCPIMConfig struct {
	ProjectID            string
	HostImageFamily      string
	HostOrchestratorPort int
	// If true, instances created should be compatible with `acloud CLI`.
	AcloudCompatible bool
}

type UNIXIMConfig struct {
	HostOrchestratorPort int
}

type AMConfig struct {
	Type  AMType
	OAuth OAuthConfig
}

type OAuthConfig struct {
	Provider    string
	RedirectURL string
}

const (
	GoogleOAuthProvider = "Google"
)

type AMType string

const (
	UnixAMType AMType = "unix"
	GAEAMType  AMType = "GCP"
)

type SMConfig struct {
	Type SMType
	GCP  *GCPSMConfig
	UNIX *UnixSMConfig
}

type SMType string

const (
	UnixSMType = "unix"
	GCPSMType  = "GCP"
)

type UnixSMConfig struct {
	SecretFilePath string
}

type GCPSMConfig struct {
	OAuthClientResourceID string
}

type ESConfig struct {
	Type   string
	Simple *SimpleESConfig
	GCPKMS *GCPKMSConfig
}

const (
	SimpleESType = "Simple"
	GCPKMSESType = "GCP_KMS" // GCP's KMS service.
)

type SimpleESConfig struct {
	KeySizeBits int
}

type GCPKMSConfig struct {
	KeyName string
}

type DBConfig struct {
	Type    string
	Spanner *SpannerConfig
}

const (
	InMemoryDBType = "InMemory"
	SpannerDBType  = "Spanner"
)

type SpannerConfig struct {
	DatabaseName string
}

type InfraConfig struct {
	STUNServers []string
}

type OperationsConfig struct {
	CreateHostDisabled bool
}

func LoadConfig() (*Config, error) {
	confFile := os.Getenv(ConfFileEnvVar)
	if confFile == "" {
		confFile = DefaultConfFile
	}
	file, err := os.Open(confFile)
	if err != nil {
		return nil, err
	}
	decoder := toml.NewDecoder(file)
	var cfg Config
	err = decoder.Decode(&cfg)
	return &cfg, err
}
