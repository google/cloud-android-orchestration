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

package config

import (
	"os"

	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/database"
	"github.com/google/cloud-android-orchestration/pkg/app/encryption"
	"github.com/google/cloud-android-orchestration/pkg/app/instances"
	"github.com/google/cloud-android-orchestration/pkg/app/secrets"

	toml "github.com/pelletier/go-toml"
)

type WebRTCConfig struct {
	STUNServers        []string
	CorsAllowedOrigins []string
}

type Config struct {
	WebStaticFilesPath string
	CORSAllowedOrigins []string
	AccountManager     accounts.Config
	SecretManager      secrets.Config
	InstanceManager    instances.Config
	EncryptionService  encryption.Config
	DatabaseService    database.Config
	WebRTC             WebRTCConfig
}

const DefaultConfFile = "conf.toml"
const ConfFileEnvVar = "CONFIG_FILE"

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
