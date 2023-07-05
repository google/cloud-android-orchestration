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
	"fmt"
	"net/url"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
)

const UnixIMType IMType = "unix"

type UNIXIMConfig struct {
	HostOrchestratorPort int
}

// Implements the Manager interface providing access to the first
// device in the local host orchestrator.
// This implementation is useful for both development and testing
type LocalInstanceManager struct {
	config Config
}

func NewLocalInstanceManager(cfg Config) *LocalInstanceManager {
	return &LocalInstanceManager{
		config: cfg,
	}
}

func (m *LocalInstanceManager) GetHostAddr(_ string, _ string) (string, error) {
	return "127.0.0.1", nil
}

func (m *LocalInstanceManager) GetHostURL(zone string, host string) (*url.URL, error) {
	addr, err := m.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", m.config.HostOrchestratorProtocol, addr, m.config.UNIX.HostOrchestratorPort))
}

func (m *LocalInstanceManager) CreateHost(_ string, _ *apiv1.CreateHostRequest, _ accounts.User) (*apiv1.Operation, error) {
	return &apiv1.Operation{
		Name: "Create Host",
		Done: true,
	}, nil
}

func (m *LocalInstanceManager) ListHosts(zone string, user accounts.User, req *ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	return &apiv1.ListHostsResponse{
		Items: []*apiv1.HostInstance{{
			Name: "local",
		}},
	}, nil
}

func (m *LocalInstanceManager) DeleteHost(zone string, user accounts.User, name string) (*apiv1.Operation, error) {
	return nil, fmt.Errorf("%T#DeleteHost is not implemented", *m)
}

func (m *LocalInstanceManager) WaitOperation(zone string, user accounts.User, name string) (any, error) {
	return nil, fmt.Errorf("%T#WaitOperation is not implemented", *m)
}

func (m *LocalInstanceManager) GetHostClient(zone string, host string) (HostClient, error) {
	url, err := m.GetHostURL(zone, host)
	if err != nil {
		return nil, err
	}
	return NewNetHostClient(url, m.config.AllowSelfSignedHostSSLCertificate), nil
}
