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

import (
	imtypes "cloud-android-orchestration/api/instancemanager/v1"
	"net/http"
)

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

type InfraConfig struct {
	IceServers []IceServer `json:"ice_servers"`
}
type IceServer struct {
	URLs []string `json:"urls"`
}

type DeviceFilesRequest struct {
	devId string
	path  string
	w     http.ResponseWriter
	r     *http.Request
}

type UserInfo interface {
	Username() string
}

type SignalingServer interface {
	// These endpoints in the SignalingServer return the (possibly modified)
	// response from the Host Orchestrator and the status code if it was
	// able to communicate with it, otherwise it returns an error.
	NewConnection(msg NewConnMsg, user UserInfo) (*SServerResponse, error)
	Forward(id string, msg ForwardMsg, user UserInfo) (*SServerResponse, error)
	Messages(id string, start int, count int, user UserInfo) (*SServerResponse, error)

	// Forwards the reques to the device's server unless it's a for a file that
	// the signaling server needs to serve itself.
	ServeDeviceFiles(params DeviceFilesRequest, user UserInfo) error
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
	DeviceFromId(name string, user UserInfo) (DeviceDesc, error)
	InsertHost(zone string, req *imtypes.InsertHostRequest, user UserInfo) (*imtypes.Operation, error)
}

type AuthHTTPHandler func(http.ResponseWriter, *http.Request, UserInfo) error
type HTTPHandler func(http.ResponseWriter, *http.Request) error

type AccountManager interface {
	// Returns the received http handler wrapped in another that extracts user
	// information from the request and passes it to to the original handler as
	// the last parameter.
	// The wrapper will only pass the request to the inner handler if a user is
	// authenticated, otherwise it may choose to return an error or respond with
	// an HTTP redirect to the login page.
	Authenticate(fn AuthHTTPHandler) HTTPHandler
}
