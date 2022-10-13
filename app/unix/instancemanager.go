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

package unix

import (
	"fmt"
	"net/url"

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"
)

// Implements the InstanceManager interface providing access to the first
// device in the local host orchestrator.
// This implementation is useful for both development and testing
type InstanceManager struct{}

func (m *InstanceManager) GetHostAddr(_ string, _ string) (string, error) {
	return "127.0.0.1", nil
}

const (
	hostURLScheme = "http"
	hostURLPort   = 1080
)

func (m *InstanceManager) GetHostURL(zone string, host string) (*url.URL, error) {
	addr, err := m.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", hostURLScheme, addr, hostURLPort))
}

func (m *InstanceManager) CreateHost(_ string, _ *apiv1.CreateHostRequest, _ app.UserInfo) (*apiv1.Operation, error) {
	return nil, app.NewInternalError(fmt.Sprintf("%T#CreateHost is not implemented", *m), nil)
}

func (m *InstanceManager) ListHosts(zone string, user app.UserInfo, req *app.ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	return nil, app.NewInternalError(fmt.Sprintf("%T#ListHosts is not implemented", *m), nil)
}

func (m *InstanceManager) DeleteHost(zone string, user app.UserInfo, name string) (*apiv1.Operation, error) {
	return nil, app.NewInternalError(fmt.Sprintf("%T#DeleteHost is not implemented", *m), nil)
}

func (m *InstanceManager) WaitOperation(zone string, user app.UserInfo, name string) (*apiv1.Operation, error) {
	return nil, app.NewInternalError(fmt.Sprintf("%T#WaitOperation is not implemented", *m), nil)
}
