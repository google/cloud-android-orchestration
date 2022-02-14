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

type ErrorMsg struct {
	Error string `json:"error"`
}

type NewConnMsg struct {
	DeviceId string `json:"device_id"`
}

type NewConnReply struct {
	ConnId     string      `json:"connection_id"`
	DeviceInfo interface{} `json:"device_info"`
}

type ForwardMsg struct {
	Payload interface{} `json:"payload"`
}

type SServerResponse struct {
	Response   interface{}
	StatusCode int
}

type SignalingServer interface {
	// All endpoints in the SignalingServer return the (possibly modified)
	// response from the Host Orchestrator and the status code if it was
	// able to communicate with it, otherwise it returns an error.
	NewConnection(msg NewConnMsg) (*SServerResponse, error)
	Forward(id string, msg ForwardMsg) (*SServerResponse, error)
	Messages(id string, start int, count int) (*SServerResponse, error)
}

type DeviceDesc struct {
	// The (internal) network address of the host where the cuttlefish device is
	// running. The address can either be an IPv4, IPv6 or a domain name.
	Addr string
	// The id under which the cuttlefish device is registered with the host
	// orchestrator (can be different from the id used in the cloud orchestrator)
	LocalId string
}

type InstanceManager interface {
	DeviceFromId(name string) (DeviceDesc, error)
}
