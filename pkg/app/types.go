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
	"net/url"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
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
	// Returns base URL the orchestrator is listening on.
	GetHostURL(zone string, host string) (*url.URL, error)
	// Creates a host instance.
	CreateHost(zone string, req *apiv1.CreateHostRequest, user UserInfo) (*apiv1.Operation, error)
	// List hosts
	ListHosts(zone string, user UserInfo, req *ListHostsRequest) (*apiv1.ListHostsResponse, error)
	// Deletes the given host instance.
	DeleteHost(zone string, user UserInfo, name string) (*apiv1.Operation, error)
	// Waits until operation is DONE or earlier. If DONE return the expected  response of the operation. If the
	// original method returns no data on success, such as `Delete`, response will be empty. If the original method
	// is standard `Get`/`Create`/`Update`, the response should be the relevant resource.
	WaitOperation(zone string, user UserInfo, name string) (interface{}, error)
}

type ListHostsRequest struct {
	// The maximum number of results per page that should be returned. If the number of available results is larger
	// than MaxResults, a `NextPageToken` will be returned which can be used to get the next page of results
	// in subsequent List requests.
	MaxResults uint32
	// Specifies a page token to use.
	// Use the `NextPageToken` value returned by a previous List request.
	PageToken string
}

type HostURLResolver interface {
	// Returns base URL the orchestrator is listening on.
	GetHostURL(zone string, host string) (*url.URL, error)
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
