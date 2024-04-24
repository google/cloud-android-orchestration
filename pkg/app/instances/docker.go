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
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/errors"
)

const DockerIMType IMType = "docker"

type DockerIMConfig struct {
	DockerImageName      string
	HostOrchestratorPort int
}

const dockerLabelCreatedBy = "created_by"

// Docker implementation of the instance manager.
type DockerInstanceManager struct {
	Config Config
	Client client.Client
}

type OPType string

const (
	CreateHostOPType OPType = "createhost"
	DeleteHostOPType OPType = "deletehost"
)

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

func (m *DockerInstanceManager) CreateHost(zone string, _ *apiv1.CreateHostRequest, user accounts.User) (*apiv1.Operation, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	ctx := context.TODO()
	config := &container.Config{
		AttachStdin: true,
		Image:       m.Config.Docker.DockerImageName,
		Tty:         true,
		Labels: map[string]string{
			dockerLabelCreatedBy: user.Username(),
		},
	}
	hostConfig := &container.HostConfig{
		Privileged: true,
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
		Name: EncodeOperationName(CreateHostOPType, createRes.ID),
		Done: true,
	}, nil
}

func (m *DockerInstanceManager) ListHosts(zone string, user accounts.User, _ *ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	ctx := context.TODO()
	ownerFilterExpr := fmt.Sprintf("%s=%s", dockerLabelCreatedBy, user.Username())
	listFilters := filters.NewArgs(
		filters.KeyValuePair{
			Key:   "label",
			Value: ownerFilterExpr,
		},
	)
	listRes, err := m.Client.ContainerList(ctx, types.ContainerListOptions{
		Filters: listFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to list docker containers: %w", err)
	}
	var items []*apiv1.HostInstance
	for _, container := range listRes {
		ipAddr, err := m.getIpAddr(&container)
		if err != nil {
			return nil, fmt.Errorf("Failed to get IP address of docker instance: %w", err)
		}
		items = append(items, &apiv1.HostInstance{
			Name: container.ID,
			Docker: &apiv1.DockerInstance{
				ImageName: container.Image,
				IPAddress: ipAddr,
			},
		})
	}
	return &apiv1.ListHostsResponse{
		Items: items,
	}, nil
}

func (m *DockerInstanceManager) DeleteHost(zone string, user accounts.User, host string) (*apiv1.Operation, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	ctx := context.TODO()
	owner, _ := m.getContainerLabel(host, dockerLabelCreatedBy)
	if owner != user.Username() {
		return nil, fmt.Errorf("User %s cannot delete docker host owned by %s", user.Username(), owner)
	}
	err := m.Client.ContainerStop(ctx, host, container.StopOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to stop docker container: %w", err)
	}
	err = m.Client.ContainerRemove(ctx, host, types.ContainerRemoveOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to remove docker container: %w", err)
	}
	return &apiv1.Operation{
		Name: EncodeOperationName(DeleteHostOPType, host),
		Done: true,
	}, nil
}

func EncodeOperationName(opType OPType, host string) string {
	return string(opType) + "_" + host
}

func DecodeOperationName(name string) (OPType, string, error) {
	words := strings.SplitN(name, "_", 2)
	if len(words) == 2 {
		return OPType(words[0]), words[1], nil
	} else {
		return "", "", errors.NewBadRequestError(fmt.Sprintf("cannot parse operation from %q", name), nil)
	}
}

func (m *DockerInstanceManager) waitCreateHostOperation(host string) (*apiv1.HostInstance, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil, errors.NewServiceUnavailableError("Wait for operation timed out", nil)
		default:
			res, err := m.Client.ContainerInspect(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("Failed to inspect docker container: %w", err)
			}
			if res.State.Running {
				return &apiv1.HostInstance{
					Name: host,
				}, nil
			}
			time.Sleep(time.Second)
		}
	}
}

func (m *DockerInstanceManager) waitDeleteHostOperation(host string) (*apiv1.HostInstance, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
	defer cancel()
	resCh, errCh := m.Client.ContainerWait(ctx, host, "")
	select {
	case <-ctx.Done():
		return nil, errors.NewServiceUnavailableError("Wait for operation timed out", nil)
	case err := <-errCh:
		return nil, fmt.Errorf("Error is thrown while waiting for deleting host: %w", err)
	case <-resCh:
		return &apiv1.HostInstance{
			Name: host,
		}, nil
	}
}

func (m *DockerInstanceManager) WaitOperation(zone string, _ accounts.User, name string) (any, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	opType, host, err := DecodeOperationName(name)
	if err != nil {
		return nil, err
	}
	switch opType {
	case CreateHostOPType:
		return m.waitCreateHostOperation(host)
	case DeleteHostOPType:
		return m.waitDeleteHostOperation(host)
	default:
		return nil, errors.NewBadRequestError(fmt.Sprintf("operation type %s not found.", opType), nil)
	}
}

func (m *DockerInstanceManager) getIpAddr(container *types.Container) (string, error) {
	bridgeNetwork := container.NetworkSettings.Networks["bridge"]
	if bridgeNetwork == nil {
		return "", fmt.Errorf("Failed to find network information of docker instance")
	}
	return bridgeNetwork.IPAddress, nil
}

func (m *DockerInstanceManager) getHostAddr(host string) (string, error) {
	ctx := context.TODO()
	listRes, err := m.Client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to list docker containers: %w", err)
	}
	var hostContainer *types.Container
	for _, container := range listRes {
		if container.ID == host {
			hostContainer = &container
			break
		}
	}
	if hostContainer == nil {
		return "", fmt.Errorf("Failed to find host: %s", host)
	}
	return m.getIpAddr(hostContainer)
}

func (m *DockerInstanceManager) getHostPort() (int, error) {
	return m.Config.Docker.HostOrchestratorPort, nil
}

func (m *DockerInstanceManager) getHostURL(host string) (*url.URL, error) {
	addr, err := m.getHostAddr(host)
	if err != nil {
		return nil, err
	}
	port, err := m.getHostPort()
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", m.Config.HostOrchestratorProtocol, addr, port))
}

func (m *DockerInstanceManager) GetHostClient(zone string, host string) (HostClient, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	url, err := m.getHostURL(host)
	if err != nil {
		return nil, err
	}
	return NewNetHostClient(url, m.Config.AllowSelfSignedHostSSLCertificate), nil
}

func (m *DockerInstanceManager) getContainerLabel(host string, key string) (string, error) {
	ctx := context.TODO()
	inspect, err := m.Client.ContainerInspect(ctx, host)
	if err != nil {
		return "", fmt.Errorf("Failed to inspect container: %w", err)
	}
	value, exist := inspect.Config.Labels[key]
	if !exist {
		return "", fmt.Errorf("Failed to find docker label: %s", key)
	}
	return value, nil
}
