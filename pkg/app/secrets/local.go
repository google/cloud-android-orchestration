// Copyright 2023 Google LLC
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

package secrets

import (
	"encoding/json"
	"fmt"
	"os"
)

const UnixSMType = "unix"

type UnixSMConfig struct {
	SecretFilePath string
}

// A secret manager that reads secrets from a file in JSON format.
type FromFileSecretManager struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewFromFileSecretManager(path string) (*FromFileSecretManager, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}
	dec := json.NewDecoder(r)
	var sm FromFileSecretManager
	if err := dec.Decode(&sm); err != nil {
		return nil, err
	}
	return &sm, nil
}

func (sm FromFileSecretManager) OAuth2ClientID() string {
	return sm.ClientID
}

func (sm FromFileSecretManager) OAuth2ClientSecret() string {
	return sm.ClientSecret
}

type CredentialType struct {
	Type string `json:"type"`
}

// Check if the credential type is service accout,
// if so: return credential content
// else: return nil
func (sm FromFileSecretManager) GetServiceAccountCredential(path string) ([]byte, error) {
	credential, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, err
	}

	var credType CredentialType
	if err = json.Unmarshal(credential, &credType); err != nil {
		return []byte{}, err
	}
	if credType.Type != "service_account" {
		return []byte{}, nil
	}
	return credential, nil
}
