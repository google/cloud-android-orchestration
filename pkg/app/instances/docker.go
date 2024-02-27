// Copyright 2024 Google LLC
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
	"context"
	"fmt"
	"net/url"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
)

const DockerIMType IMType = "docker"

type DockerIMConfig struct {
	DockerImageName      string
	HostOrchestratorPort int
}

// Docker implementation of the instance manager.
type DockerInstanceManager struct {
	Config Config
	Client client.Client
}

func NewDockerInstanceManager(cfg Config, cli client.Client) *DockerInstanceManager {
	return &DockerInstanceManager{
		Config: cfg,
		Client: cli,
	}
}

func (m *DockerInstanceManager) ListZones() (*apiv1.ListZonesResponse, error) {
	return &apiv1.ListZonesResponse{
		Items: []*apiv1.Zone{{
			Name: "local",
		}},
	}, nil
}

func (m *DockerInstanceManager) CreateHost(zone string, _ *apiv1.CreateHostRequest, _ accounts.User) (*apiv1.Operation, error) {
	if zone != "local" {
		return nil, fmt.Errorf("Invalid zone. It should be 'local'.")
	}
	ctx := context.TODO()
	config := &container.Config{
		AttachStdin: true,
		Image:       m.Config.Docker.DockerImageName,
		Tty:         true,
	}
	hostConfig := &container.HostConfig{
		Privileged:      true,
		PublishAllPorts: true,
	}
	createRes, err := m.Client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("Failed to create docker container: %w", err)
	}
	err = m.Client.ContainerStart(ctx, createRes.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to start docker container: %w", err)
	}
	return &apiv1.Operation{
		Name: createRes.ID,
		Done: true,
	}, nil
}

func (m *DockerInstanceManager) ListHosts(zone string, _ accounts.User, _ *ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	if zone != "local" {
		return nil, fmt.Errorf("Invalid zone. It should be 'local'.")
	}
	ctx := context.TODO()
	listRes, err := m.Client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to list docker containers: %w", err)
	}
	var items []*apiv1.HostInstance
	for _, container := range listRes {
		items = append(items, &apiv1.HostInstance{
			Name: container.ID,
		})
	}
	return &apiv1.ListHostsResponse{
		Items: items,
	}, nil
}

func (m *DockerInstanceManager) DeleteHost(zone string, _ accounts.User, host string) (*apiv1.Operation, error) {
	if zone != "local" {
		return nil, fmt.Errorf("Invalid zone. It should be 'local'.")
	}
	ctx := context.TODO()
	err := m.Client.ContainerStop(ctx, host, container.StopOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to stop docker container: %w", err)
	}
	err = m.Client.ContainerRemove(ctx, host, types.ContainerRemoveOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to remove docker container: %w", err)
	}
	return &apiv1.Operation{
		Name: host,
		Done: true,
	}, nil
}

func (m *DockerInstanceManager) WaitOperation(zone string, _ accounts.User, _ string) (any, error) {
	if zone != "local" {
		return nil, fmt.Errorf("Invalid zone. It should be 'local'.")
	}
	return nil, fmt.Errorf("%T#WaitOperation is not implemented", *m)
}

func (m *DockerInstanceManager) GetHostAddr() (string, error) {
	return "127.0.0.1", nil
}

func (m *DockerInstanceManager) GetHostPort(host string) (int, error) {
	ctx := context.TODO()
	listRes, err := m.Client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return -1, fmt.Errorf("Failed to list docker containers: %w", err)
	}
	var hostContainer *types.Container
	for _, container := range listRes {
		if container.ID == host {
			hostContainer = &container
			break
		}
	}
	if hostContainer == nil {
		return -1, fmt.Errorf("Failed to find host: %s", host)
	}
	var exposedHostOrchestratorPort int
	exposedHostOrchestratorPort = -1
	for _, port := range hostContainer.Ports {
		if int(port.PrivatePort) == m.Config.Docker.HostOrchestratorPort {
			exposedHostOrchestratorPort = int(port.PublicPort)
			break
		}
	}
	if exposedHostOrchestratorPort == -1 {
		return -1, fmt.Errorf("Failed to find exposed Host Orchestrator port for given host: %s", host)
	}
	return exposedHostOrchestratorPort, nil
}

func (m *DockerInstanceManager) GetHostURL(host string) (*url.URL, error) {
	addr, err := m.GetHostAddr()
	if err != nil {
		return nil, err
	}
	port, err := m.GetHostPort(host)
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", m.Config.HostOrchestratorProtocol, addr, port))
}

func (m *DockerInstanceManager) GetHostClient(zone string, host string) (HostClient, error) {
	if zone != "local" {
		return nil, fmt.Errorf("Invalid zone. It should be 'local'.")
	}
	url, err := m.GetHostURL(host)
	if err != nil {
		return nil, err
	}
	return NewNetHostClient(url, m.Config.AllowSelfSignedHostSSLCertificate), nil
}
