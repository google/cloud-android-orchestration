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
	"net/http/httputil"

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
	// Creates a host instance.
	CreateHost(zone string, req *apiv1.CreateHostRequest, user UserInfo) (*apiv1.Operation, error)
	// List hosts
	ListHosts(zone string, user UserInfo, req *ListHostsRequest) (*apiv1.ListHostsResponse, error)
	// Deletes the given host instance.
	DeleteHost(zone string, user UserInfo, name string) (*apiv1.Operation, error)
	// Waits until operation is DONE or earlier. If DONE return the expected  response of the operation. If the
	// original method returns no data on success, such as `Delete`, response will be empty. If the original method
	// is standard `Get`/`Create`/`Update`, the response should be the relevant resource.
	WaitOperation(zone string, user UserInfo, name string) (any, error)
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

type AuthHTTPHandler func(http.ResponseWriter, *http.Request, UserInfo) error
type HTTPHandler func(http.ResponseWriter, *http.Request) error

// ID tokens (from OpenID connect) are presented in JWT format, with the relevant fields in the Claims section.
type IDTokenClaims map[string]interface{}

type AccountManager interface {
	// Returns the received http handler wrapped in another that extracts user
	// information from the request and passes it to to the original handler as
	// the last parameter.
	// The wrapper will only pass the request to the inner handler if a user is
	// authenticated, otherwise it may choose to return an error or respond with
	// an HTTP redirect to the login page.
	Authenticate(fn AuthHTTPHandler) HTTPHandler
	// Gives the account manager the chance to extract login information from the token (id token
	// for example), validate it, add cookies to the request, etc.
	OnOAuthExchange(w http.ResponseWriter, r *http.Request, idToken IDTokenClaims) (UserInfo, error)
}

type SecretManager interface {
	OAuthClientID() string
	OAuthClientSecret() string
}

type EncryptionService interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type Session struct {
	Key         string
	OAuth2State string
}

type DatabaseService interface {
	// Credentials are usually stored encrypted hence the []byte type.
	// If no credentials are available for the given user Fetch returns nil, nil.
	FetchBuildAPICredentials(username string) ([]byte, error)
	// Store new credentials or overwrite existing ones for the given user.
	StoreBuildAPICredentials(username string, credentials []byte) error
	// Create or update a user session.
	CreateOrUpdateSession(s Session) error
	// Fetch a session. Returns nil, nil if the session doesn't exist.
	FetchSession(key string) (*Session, error)
	// Delete a session. Won't return error if the session doesn't exist.
	DeleteSession(key string) error
}
