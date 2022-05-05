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

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"

	compute "cloud.google.com/go/compute/apiv1"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

const (
	namePrefix  = "cf-"
	labelPrefix = "cf-"
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

func (m *InstanceManager) DeviceFromId(zone string, host string, name string, _ app.UserInfo) (app.DeviceDesc, error) {
	return app.DeviceDesc{"127.0.0.1", "cvd-1"}, nil
}

func (m *InstanceManager) CreateHost(zone string, req *apiv1.CreateHostRequest, user app.UserInfo) (*apiv1.Operation, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	labels := map[string]string{
		"created_by":               user.Username(), // required for acloud backwards compatibility
		labelPrefix + "created_by": user.Username(),
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

// TODO(b/226935747) Have more thorough validation error in Instance Manager.
var ErrBadCreateHostRequest = app.NewBadRequestError("invalid CreateHostRequest", nil)

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.CreateHostInstanceRequest == nil {
		return ErrBadCreateHostRequest
	}
	if r.CreateHostInstanceRequest.GCP == nil {
		return ErrBadCreateHostRequest
	}
	if r.CreateHostInstanceRequest.GCP.DiskSizeGB == 0 {
		return ErrBadCreateHostRequest
	}
	if r.CreateHostInstanceRequest.GCP.MachineType == "" {
		return ErrBadCreateHostRequest
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
