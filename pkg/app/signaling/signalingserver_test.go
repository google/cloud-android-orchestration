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

package signaling

import (
	"strings"
	"testing"
)

var encodingTests = []struct {
	name   string
	connID string
	devId  string
}{
	{"Simple Encoding and Decoding", "exampleConnId", "exampleDevId"},
	{
		"Encoding and Decoding with separators",
		strings.ReplaceAll("connID:with:some:separators and spaces", ":", CONN_ID_SEPARATOR),
		strings.ReplaceAll("devId:with:separators and spaces too", ":", CONN_ID_SEPARATOR),
	},
}

func TestIdEncodingAndDecoding(t *testing.T) {
	for _, tt := range encodingTests {
		t.Run(tt.name, func(t *testing.T) {
			aux(tt.connID, tt.devId, t)
		})
	}
}

func aux(connID string, devId string, t *testing.T) {
	enc := encodeConnId(connID, devId)
	dec, err := decodeConnId(enc)
	if err != nil {
		t.Errorf("Failed to decode encoded id: %s", err.Error())
		return
	}
	if dec.ConnId != connID {
		t.Errorf("Decoded connection id doesn't match original: %s vs %s", dec.ConnId, connID)
	}
	if dec.DevId != devId {
		t.Errorf("Decoded device id doesn't match original: %s vs %s", dec.DevId, devId)
	}
}
