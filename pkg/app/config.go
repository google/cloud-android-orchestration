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

	"github.com/google/cloud-android-orchestration/pkg/app/types"

	toml "github.com/pelletier/go-toml"
)

const DefaultConfFile = "conf.toml"
const ConfFileEnvVar = "CONFIG_FILE"

func LoadConfig() (*types.Config, error) {
	confFile := os.Getenv(ConfFileEnvVar)
	if confFile == "" {
		confFile = DefaultConfFile
	}
	file, err := os.Open(confFile)
	if err != nil {
		return nil, err
	}
	decoder := toml.NewDecoder(file)
	var cfg types.Config
	err = decoder.Decode(&cfg)
	return &cfg, err
}
