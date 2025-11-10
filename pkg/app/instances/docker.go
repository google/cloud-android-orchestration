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
	"io"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

const DockerIMType IMType = "docker"

type DockerIMConfig struct {
	DockerImageName      string
	HostOrchestratorPort int
}

const (
	dockerLabelCreatedBy      = "created_by"
	dockerLabelKeyManagedBy   = "managed_by"
	dockerLabelValueManagedBy = "cloud_orchestrator"
)

const uaMountTarget = "/var/lib/cuttlefish-common/userartifacts"

// Docker implementation of the instance manager.
type DockerInstanceManager struct {
	Config     Config
	Client     *client.Client
	mutexes    sync.Map
	operations sync.Map
}

func NewDockerInstanceManager(cfg Config, cli *client.Client) *DockerInstanceManager {
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
	op := m.newOperation()
	go func() {
		val, err := m.createHost(user)
		if opErr := m.completeOperation(op.Name, &operationResult{Error: err, Value: val}); opErr != nil {
			log.Printf("error completing operation %q: %v\n", op.Name, opErr)
		}
	}()
	return &op, nil
}

func (m *DockerInstanceManager) ListHosts(zone string, user accounts.User, _ *ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	ctx := context.TODO()
	listRes, err := m.Client.ContainerList(ctx, container.ListOptions{
		Filters: dockerFilter(user),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}
	var items []*apiv1.HostInstance
	for _, container := range listRes {
		ipAddr, err := m.getIpAddr(&container)
		if err != nil {
			return nil, fmt.Errorf("failed to get IP address of docker instance: %w", err)
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
	op := m.newOperation()
	go func() {
		val, err := m.deleteHost(user, host)
		if opErr := m.completeOperation(op.Name, &operationResult{Error: err, Value: val}); opErr != nil {
			log.Printf("error completing operation %q: %v\n", op.Name, opErr)
		}
	}()
	return &op, nil
}

func (m *DockerInstanceManager) WaitOperation(zone string, _ accounts.User, name string) (any, error) {
	if zone != "local" {
		return nil, errors.NewBadRequestError("Invalid zone. It should be 'local'.", nil)
	}
	val, err := m.waitOperation(name, 3*time.Minute)
	if err != nil {
		return nil, err
	}
	return val.Value, val.Error
}

func (m *DockerInstanceManager) getIpAddr(container *types.Container) (string, error) {
	bridgeNetwork := container.NetworkSettings.Networks["bridge"]
	if bridgeNetwork == nil {
		return "", fmt.Errorf("failed to find network information of docker instance")
	}
	return bridgeNetwork.IPAddress, nil
}

func (m *DockerInstanceManager) getHostAddr(host string) (string, error) {
	ctx := context.TODO()
	listRes, err := m.Client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list docker containers: %w", err)
	}
	var hostContainer *types.Container
	for _, container := range listRes {
		if container.ID == host {
			hostContainer = &container
			break
		}
	}
	if hostContainer == nil {
		return "", fmt.Errorf("failed to find host: %s", host)
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

func uaVolumeName(user accounts.User) string {
	return fmt.Sprintf("user_artifacts_%s", user.Username())
}

func dockerFilter(user accounts.User) filters.Args {
	ownerFilterExpr := fmt.Sprintf("%s=%s", dockerLabelCreatedBy, user.Username())
	managerFilterExpr := fmt.Sprintf("%s=%s", dockerLabelKeyManagedBy, dockerLabelValueManagedBy)
	return filters.NewArgs(
		filters.KeyValuePair{
			Key:   "label",
			Value: ownerFilterExpr,
		},
		filters.KeyValuePair{
			Key:   "label",
			Value: managerFilterExpr,
		},
	)
}

func dockerLabelsDict(user accounts.User) map[string]string {
	return map[string]string{
		dockerLabelCreatedBy:    user.Username(),
		dockerLabelKeyManagedBy: dockerLabelValueManagedBy,
	}
}

func (m *DockerInstanceManager) getContainerLabel(host string, key string) (string, error) {
	ctx := context.TODO()
	inspect, err := m.Client.ContainerInspect(ctx, host)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	value, exist := inspect.Config.Labels[key]
	if !exist {
		return "", fmt.Errorf("failed to find docker label: %s", key)
	}
	return value, nil
}

func (m *DockerInstanceManager) createHost(user accounts.User) (*apiv1.HostInstance, error) {
	mu := m.getRWMutex(user)
	mu.RLock()
	defer mu.RUnlock()
	ctx := context.TODO()
	if err := m.downloadDockerImageIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("failed to retrieve docker image name: %w", err)
	}
	// A docker volume is shared across all hosts under each user. If no volume
	// exists for given user, create it.
	if err := m.createDockerVolumeIfNeeded(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to prepare docker volume: %w", err)
	}
	host, err := m.createDockerContainer(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare docker container: %w", err)
	}
	return &apiv1.HostInstance{
		Name: host,
	}, nil
}

func (m *DockerInstanceManager) downloadDockerImageIfNeeded(ctx context.Context) error {
	listRes, err := m.Client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list docker images: %w", err)
	}
	for _, image := range listRes {
		for _, tag := range image.RepoTags {
			if tag == m.Config.Docker.DockerImageName {
				return nil
			}
		}
	}

	reader, err := m.Client.ImagePull(ctx, m.Config.Docker.DockerImageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to request pulling docker image %q: %w", m.Config.Docker.DockerImageName, err)
	}
	defer reader.Close()
	// Caller of ImagePull should handle its output to complete actual ImagePull operation.
	// Details in https://pkg.go.dev/github.com/docker/docker/client#Client.ImagePull.
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("failed to pull docker image %q: %w", m.Config.Docker.DockerImageName, err)
	}
	log.Println("Downloaded docker image: " + m.Config.Docker.DockerImageName)
	return nil
}

func (m *DockerInstanceManager) createDockerVolumeIfNeeded(ctx context.Context, user accounts.User) error {
	listOpts := volume.ListOptions{Filters: dockerFilter(user)}
	volumeListRes, err := m.Client.VolumeList(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list docker volume: %w", err)
	}
	if len(volumeListRes.Volumes) > 0 {
		return nil
	}
	createOpts := volume.CreateOptions{
		Name:   uaVolumeName(user),
		Labels: dockerLabelsDict(user),
	}
	if _, err := m.Client.VolumeCreate(ctx, createOpts); err != nil {
		return fmt.Errorf("failed to create docker volume: %w", err)
	}
	return nil
}

const containerInspectRetryLimit = 5

func (m *DockerInstanceManager) createDockerContainer(ctx context.Context, user accounts.User) (string, error) {
	config := &container.Config{
		AttachStdin: true,
		Image:       m.Config.Docker.DockerImageName,
		Tty:         true,
		Labels:      dockerLabelsDict(user),
	}
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: uaVolumeName(user),
				Target: uaMountTarget,
			},
		},
		Privileged: true,
	}
	createRes, err := m.Client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create docker container: %w", err)
	}
	if err := m.Client.ContainerStart(ctx, createRes.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start docker container: %w", err)
	}
	execConfig := container.ExecOptions{
		Cmd:          []string{"chown", "httpcvd:httpcvd", uaMountTarget},
		AttachStdout: false,
		AttachStderr: false,
		Tty:          false,
	}
	execRes, err := m.Client.ContainerExecCreate(ctx, createRes.ID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create container execution %q: %w", strings.Join(execConfig.Cmd, " "), err)
	}
	if err := m.Client.ContainerExecStart(ctx, execRes.ID, container.ExecStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container execution %q: %w", strings.Join(execConfig.Cmd, " "), err)
	}
	for i := 0; ; i++ {
		res, err := m.Client.ContainerInspect(ctx, createRes.ID)
		if err == nil && res.State.Running {
			return createRes.ID, nil
		}
		if i >= containerInspectRetryLimit {
			return "", fmt.Errorf("failed to inspect docker container: %w", err)
		}
		time.Sleep(time.Second)
	}
}

func (m *DockerInstanceManager) deleteHost(user accounts.User, host string) (*apiv1.HostInstance, error) {
	ctx := context.TODO()
	if err := m.deleteDockerContainer(ctx, user, host); err != nil {
		return nil, fmt.Errorf("failed to delete docker container: %w", err)
	}
	// A docker volume is shared across all hosts under each user. If no host
	// exists for given user, delete volume afterwards to cleanup.
	if err := m.deleteDockerVolumeIfNeeded(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to cleanup docker volume: %w", err)
	}
	return &apiv1.HostInstance{
		Name: host,
	}, nil
}

func (m *DockerInstanceManager) deleteDockerContainer(ctx context.Context, user accounts.User, host string) error {
	if owner, err := m.getContainerLabel(host, dockerLabelCreatedBy); err != nil {
		return fmt.Errorf("failed to get container label: %w", err)
	} else if owner != user.Username() {
		return fmt.Errorf("user %s cannot delete docker host owned by %s", user.Username(), owner)
	}
	if err := m.Client.ContainerStop(ctx, host, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop docker container: %w", err)
	}
	if err := m.Client.ContainerRemove(ctx, host, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove docker container: %w", err)
	}
	return nil
}

const volumeInspectRetryCount = 3

func (m *DockerInstanceManager) deleteDockerVolumeIfNeeded(ctx context.Context, user accounts.User) error {
	mu := m.getRWMutex(user)
	if locked := mu.TryLock(); !locked {
		// If it can't acquire lock on this mutex, there's ongoing host
		// creation with this volume or deletion of this volume. For these
		// cases, it doesn't need to delete docker volume here.
		return nil
	}
	defer mu.Unlock()

	containerListOpts := container.ListOptions{Filters: dockerFilter(user)}
	listRes, err := m.Client.ContainerList(ctx, containerListOpts)
	if err != nil {
		return fmt.Errorf("failed to list docker containers: %w", err)
	}
	if len(listRes) > 0 {
		return nil
	}
	listOpts := volume.ListOptions{Filters: dockerFilter(user)}
	volumeListRes, err := m.Client.VolumeList(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list docker volume: %w", err)
	}
	for _, volume := range volumeListRes.Volumes {
		if err := m.Client.VolumeRemove(ctx, volume.Name, true); err != nil {
			return fmt.Errorf("failed to remove docker volume: %w", err)
		}
	}
	// Ensure the deletion of docker volumes
	for i := 0; i < volumeInspectRetryCount; i++ {
		volumeListRes, err := m.Client.VolumeList(ctx, listOpts)
		if err == nil && len(volumeListRes.Volumes) == 0 {
			return nil
		}
	}
	return fmt.Errorf("removed docker volume but still exists")
}

func (m *DockerInstanceManager) getRWMutex(user accounts.User) *sync.RWMutex {
	mu, _ := m.mutexes.LoadOrStore(user.Username(), &sync.RWMutex{})
	return mu.(*sync.RWMutex)
}

type operationResult struct {
	Error error
	Value interface{}
}

type operationEntry struct {
	op     apiv1.Operation
	result *operationResult
	mutex  sync.RWMutex
	done   chan struct{}
}

const newOperationRetryLimit = 100

func (m *DockerInstanceManager) newOperation() apiv1.Operation {
	for i := 0; i < newOperationRetryLimit; i++ {
		name := uuid.New().String()
		newEntry := &operationEntry{
			op: apiv1.Operation{
				Name: name,
				Done: false,
			},
			mutex: sync.RWMutex{},
			done:  make(chan struct{}),
		}
		entry, loaded := m.operations.LoadOrStore(name, newEntry)
		if !loaded {
			// It succeeded to store a new operation entry.
			return entry.(*operationEntry).op
		}
	}
	panic("Reached newOperationRetryLimit")
}

func (m *DockerInstanceManager) completeOperation(name string, result *operationResult) error {
	val, loaded := m.operations.Load(name)
	if !loaded {
		return fmt.Errorf("operation not found for %q", name)
	}
	entry := val.(*operationEntry)

	entry.mutex.Lock()
	defer entry.mutex.Unlock()
	entry.op.Done = true
	entry.result = result
	close(entry.done)
	return nil
}

func (m *DockerInstanceManager) waitOperation(name string, dt time.Duration) (*operationResult, error) {
	val, loaded := m.operations.Load(name)
	if !loaded {
		return nil, fmt.Errorf("operation not found for %q", name)
	}
	entry := val.(*operationEntry)
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Minute)
	defer cancel()
	select {
	case <-entry.done:
		entry.mutex.RLock()
		result := entry.result
		entry.mutex.RUnlock()
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("reached timeout for %q", name)
	}
}
