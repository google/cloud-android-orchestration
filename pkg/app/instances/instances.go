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

package instances

import (
	"net/http/httputil"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
)

type Manager interface {
	// List zones
	ListZones() (*apiv1.ListZonesResponse, error)
	// Creates a host instance.
	CreateHost(zone string, req *apiv1.CreateHostRequest, user accounts.User) (*apiv1.Operation, error)
	// List hosts
	ListHosts(zone string, user accounts.User, req *ListHostsRequest) (*apiv1.ListHostsResponse, error)
	// Deletes the given host instance.
	DeleteHost(zone string, user accounts.User, name string) (*apiv1.Operation, error)
	// Waits until operation is DONE or earlier. If DONE return the expected  response of the operation. If the
	// original method returns no data on success, such as `Delete`, response will be empty. If the original method
	// is standard `Get`/`Create`/`Update`, the response should be the relevant resource.
	WaitOperation(zone string, user accounts.User, name string) (any, error)
	// Creates a connector to the given host.
	GetHostClient(zone string, host string) (HostClient, error)
}

type HostClient interface {
	// Get and Post requests return the HTTP status code or an error.
	// The response body is parsed into the res output parameter if provided.
	Get(URLPath, URLQuery string, res *HostResponse) (int, error)
	Post(URLPath, URLQuery string, bodyJSON any, res *HostResponse) (int, error)
	GetReverseProxy() *httputil.ReverseProxy
}

type HostResponse struct {
	Result any
	Error  *apiv1.Error
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

type IMType string

type Config struct {
	Type IMType
	// The protocol the host orchestrator expects, either http or https
	HostOrchestratorProtocol          string
	AllowSelfSignedHostSSLCertificate bool
	GCP                               *GCPIMConfig
	UNIX                              *UNIXIMConfig
	Docker                            *DockerIMConfig
}
