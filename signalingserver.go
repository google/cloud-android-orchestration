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

package main

// The ForwardingSignalingServer implements the SignalingServer interface by
// communicating with the host orchestrator in the appropriate instance and
// forwarding all requests to it.
type ForwardingSignalingServer struct{}

func NewForwardingSignalingServer() *ForwardingSignalingServer {
	return &ForwardingSignalingServer{}
}

func (s *ForwardingSignalingServer) NewConnection(msg NewConnMsg) NewConnReply {
	return NewConnReply{ConnId: "id", DeviceInfo: "device info"}
}

func (s *ForwardingSignalingServer) Forward(id string, msg ForwardMsg) error {
	return nil
}

func (s *ForwardingSignalingServer) Messages(id string, start int, count int) ([]interface{}, error) {
	return make([]interface{}, 0), nil
}
