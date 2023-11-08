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

package encryption

import (
	"context"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

const GCPKMSESType = "GCP_KMS" // GCP's KMS service.

type GCPKMSConfig struct {
	KeyName string
}

type GCPKMSEncryptionService struct {
	keyName string
}

func NewGCPKMSEncryptionService(keyName string) *GCPKMSEncryptionService {
	return &GCPKMSEncryptionService{keyName}
}

func (s *GCPKMSEncryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	ctx := context.TODO()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate KMS client: %w", err)
	}
	defer client.Close()
	req := &kmspb.EncryptRequest{
		Name:      s.keyName,
		Plaintext: plaintext,
	}
	result, err := client.Encrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed encryption request: %w", err)
	}
	return result.Ciphertext, nil
}

func (s *GCPKMSEncryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	ctx := context.TODO()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate KMS client: %w", err)
	}
	defer client.Close()
	req := &kmspb.DecryptRequest{
		Name:       s.keyName,
		Ciphertext: ciphertext,
	}
	result, err := client.Decrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed encryption request: %w", err)
	}
	return result.Plaintext, nil
}
