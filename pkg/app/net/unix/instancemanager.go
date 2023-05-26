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

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/net"
	"github.com/google/cloud-android-orchestration/pkg/app/types"
)

// Implements the InstanceManager interface providing access to the first
// device in the local host orchestrator.
// This implementation is useful for both development and testing
type InstanceManager struct {
	config types.IMConfig
}

func NewInstanceManager(cfg types.IMConfig) *InstanceManager {
	return &InstanceManager{
		config: cfg,
	}
}

func (m *InstanceManager) GetHostAddr(_ string, _ string) (string, error) {
	return "127.0.0.1", nil
}

func (m *InstanceManager) GetHostURL(zone string, host string) (*url.URL, error) {
	addr, err := m.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", m.config.HostOrchestratorProtocol, addr, m.config.UNIX.HostOrchestratorPort))
}

func (m *InstanceManager) CreateHost(_ string, _ *apiv1.CreateHostRequest, _ types.UserInfo) (*apiv1.Operation, error) {
	return nil, fmt.Errorf("%T#CreateHost is not implemented", *m)
}

func (m *InstanceManager) ListHosts(zone string, user types.UserInfo, req *types.ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	return nil, fmt.Errorf("%T#ListHosts is not implemented", *m)
}

func (m *InstanceManager) DeleteHost(zone string, user types.UserInfo, name string) (*apiv1.Operation, error) {
	return nil, fmt.Errorf("%T#DeleteHost is not implemented", *m)
}

func (m *InstanceManager) WaitOperation(zone string, user types.UserInfo, name string) (any, error) {
	return nil, fmt.Errorf("%T#WaitOperation is not implemented", *m)
}

func (m *InstanceManager) GetHostClient(zone string, host string) (types.HostClient, error) {
	url, err := m.GetHostURL(zone, host)
	if err != nil {
		return nil, err
	}
	return net.NewHostClient(url, m.config.AllowSelfSignedHostSSLCertificate), nil
}
