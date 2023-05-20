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

package unix

// A simple and very insecure implementation of an encryption service to be used for testing and
// local development.
type SimpleEncryptionService struct{}

func NewSimpleEncryptionService(key []byte) (*SimpleEncryptionService, error) {
	return &SimpleEncryptionService{}, nil
}

func (es *SimpleEncryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	// Pretend to encrypt/decrypt messages by flipping the bits in the message. That ensures the
	// encrypted message is different than the original.
	const mask byte = 255
	res := make([]byte, len(plaintext))
	for i := 0; i < len(plaintext); i += 1 {
		res[i] = plaintext[i] ^ mask
	}
	return res, nil
}

func (es *SimpleEncryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	// Same procedure to decrypt
	return es.Encrypt(ciphertext)
}
