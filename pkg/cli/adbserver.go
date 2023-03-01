// Copyright 2022 Google LLC
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
	"fmt"
	"net"
)

type ADBServerProxy interface {
	Connect(port int) error
	Disconnect(port int) error
}

const ADBServerPort = 5037

type ADBServerProxyImpl struct{}

func (p *ADBServerProxyImpl) Connect(port int) error {
	adbSerial := fmt.Sprintf("127.0.0.1:%d", port)
	msg := fmt.Sprintf("host:connect:%s", adbSerial)
	return p.sendMsg(msg)
}

func (p *ADBServerProxyImpl) Disconnect(port int) error {
	adbSerial := fmt.Sprintf("127.0.0.1:%d", port)
	msg := fmt.Sprintf("host:disconnect:%s", adbSerial)
	return p.sendMsg(msg)
}

func (*ADBServerProxyImpl) sendMsg(msg string) error {
	msg = fmt.Sprintf("%.4x%s", len(msg), msg)
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ADBServerPort))
	if err != nil {
		return fmt.Errorf("Unable to contact ADB server: %w", err)
	}
	defer conn.Close()
	written := 0
	for written < len(msg) {
		n, err := conn.Write([]byte(msg[written:]))
		if err != nil {
			return fmt.Errorf("Error sending message to ADB server: %w", err)
		}
		written += n
	}
	return nil
}
