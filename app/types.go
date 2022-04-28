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
	"net/http"

	apiv1 "cloud-android-orchestration/api/v1"
)

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
	NewConnection(zone string, host string, msg apiv1.NewConnMsg, user UserInfo) (*apiv1.SServerResponse, error)
	Forward(zone string, host string, id string, msg apiv1.ForwardMsg, user UserInfo) (*apiv1.SServerResponse, error)
	Messages(zone string, host string, id string, start int, count int, user UserInfo) (*apiv1.SServerResponse, error)

	// Forwards the reques to the device's server unless it's a for a file that
	// the signaling server needs to serve itself.
	ServeDeviceFiles(zone string, host string, params DeviceFilesRequest, user UserInfo) error
}

type InstanceManager interface {
	// Returns the (internal) network address of the host where the cuttlefish device is
	// running. The address can either be an IPv4, IPv6 or a domain name.
	GetHostAddr(zone string, host string) (string, error)
	// Creates a host instance.
	CreateHost(zone string, req *apiv1.CreateHostRequest, user UserInfo) (*apiv1.Operation, error)
	// Closes the connection with the underlying API
	Close() error
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
