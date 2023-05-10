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

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

// A simple and probably insecure implementation of an encryption service to be used for testing and
// local development.
type SimpleEncryptionService struct {
	block cipher.Block
}

func NewSimpleEncryptionService(key []byte) (*SimpleEncryptionService, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &SimpleEncryptionService{block}, nil
}

func (es *SimpleEncryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	blkSize := es.block.BlockSize()
	iv := make([]byte, blkSize)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	ec := cipher.NewCFBEncrypter(es.block, iv)
	ciphertext := make([]byte, len(plaintext))
	ec.XORKeyStream(ciphertext, plaintext)
	// Put the initialization vector at the beginning of the ciphertext.
	return append(iv, ciphertext...), nil
}

func (es *SimpleEncryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	blkSize := es.block.BlockSize()
	// Panic if smaller than blkSize, that's ok
	iv := ciphertext[:blkSize]
	ciphertext = ciphertext[blkSize:]
	dc := cipher.NewCFBDecrypter(es.block, iv)
	plaintext := make([]byte, len(ciphertext))
	dc.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
