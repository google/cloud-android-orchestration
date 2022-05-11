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

package gcp

import (
	"context"
	"fmt"
	"log"

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"

	compute "cloud.google.com/go/compute/apiv1"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

const (
	namePrefix           = "cf-"
	labelPrefix          = "cf-"
	labelAcloudCreatedBy = "created_by" // required for acloud backwards compatibility
	labelCreatedBy       = labelPrefix + "created_by"
)

// GCP implementation of the instance manager.
type InstanceManager struct {
	config      *app.IMConfig
	client      *compute.InstancesClient
	uuidFactory func() string
}

func NewInstanceManager(config *app.IMConfig, ctx context.Context, opts ...option.ClientOption) (*InstanceManager, error) {
	client, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &InstanceManager{
		config:      config,
		client:      client,
		uuidFactory: func() string { return uuid.New().String() },
	}, nil
}

func (m *InstanceManager) GetHostAddr(zone string, host string) (string, error) {
	instance, err := m.getHostInstance(zone, host)
	if err != nil {
		return "", err
	}
	ilen := len(instance.NetworkInterfaces)
	if ilen == 0 {
		log.Printf("host instance %s in zone %s is missing a network interface", host, zone)
		return "", app.NewInternalError("host instance missing a network interface", nil)
	}
	if ilen > 1 {
		log.Printf("host instance %s in zone %s has %d network interfaces", host, zone, ilen)
	}
	return *instance.NetworkInterfaces[0].NetworkIP, nil
}

func (m *InstanceManager) CreateHost(zone string, req *apiv1.CreateHostRequest, user app.UserInfo) (*apiv1.Operation, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	labels := map[string]string{
		labelAcloudCreatedBy: user.Username(),
		labelCreatedBy:       user.Username(),
	}
	ctx := context.TODO()
	computeReq := &computepb.InsertInstanceRequest{
		Project: m.config.GCP.ProjectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name:           proto.String(namePrefix + m.uuidFactory()),
			MachineType:    proto.String(req.CreateHostInstanceRequest.GCP.MachineType),
			MinCpuPlatform: proto.String(req.CreateHostInstanceRequest.GCP.MinCPUPlatform),
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(int64(req.CreateHostInstanceRequest.GCP.DiskSizeGB)),
						SourceImage: proto.String(m.config.GCP.HostImage),
					},
					Boot: proto.Bool(true),
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String(buildDefaultNetworkName(m.config.GCP.ProjectID)),
					AccessConfigs: []*computepb.AccessConfig{
						{
							Name: proto.String("External NAT"),
							Type: proto.String(computepb.AccessConfig_ONE_TO_ONE_NAT.String()),
						},
					},
				},
			},
			Labels: labels,
		},
	}
	op, err := m.client.Insert(ctx, computeReq)
	if err != nil {
		return nil, err
	}
	result := &apiv1.Operation{
		Name: op.Name(),
		Done: op.Done(),
	}
	return result, nil
}

func (m *InstanceManager) Close() error {
	return m.client.Close()
}

func (m *InstanceManager) getHostInstance(zone string, host string) (*computepb.Instance, error) {
	ctx := context.TODO()
	req := &computepb.GetInstanceRequest{
		Project:  m.config.GCPConfig.ProjectID,
		Zone:     zone,
		Instance: host,
	}
	return m.client.Get(ctx, req)
}

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.CreateHostInstanceRequest == nil ||
		r.CreateHostInstanceRequest.GCP == nil ||
		r.CreateHostInstanceRequest.GCP.DiskSizeGB == 0 ||
		r.CreateHostInstanceRequest.GCP.MachineType == "" {
		return app.NewBadRequestError("invalid CreateHostRequest", nil)
	}
	return nil
}

func buildDefaultNetworkName(projectID string) string {
	return fmt.Sprintf("projects/%s/global/networks/default", projectID)
}

// Internal setter method used for testing only.
func (m *InstanceManager) setUUIDFactory(newFactory func() string) {
	m.uuidFactory = newFactory
}
