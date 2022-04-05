package main

import (
	"context"
	"fmt"

	apiv1 "cloud-android-orchestration/api/v1"

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

// modified during testing
var newUUIDString = func() string {
	return uuid.New().String()
}

// GCP implementation of the instance manager.
type GCPInstanceManager struct {
	config *Config
	client *compute.InstancesClient
}

func NewGCPInstanceManager(config *Config, opts ...option.ClientOption) (*GCPInstanceManager, error) {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &GCPInstanceManager{
		config: config,
		client: client,
	}, nil
}

func (m *GCPInstanceManager) DeviceFromId(name string, _ UserInfo) (DeviceDesc, error) {
	return DeviceDesc{"127.0.0.1", "cvd-1"}, nil
}

func (m *GCPInstanceManager) CreateHost(zone string, req *apiv1.CreateHostRequest, user UserInfo) (*apiv1.Operation, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	labels := map[string]string{
		"created_by":               user.Username(), // required for acloud backwards compatibility
		labelPrefix + "created_by": user.Username(),
	}
	if req.CreateCVDRequest != nil {
		labels[labelPrefix+"build_id"] = req.CreateCVDRequest.BuildID
		labels[labelPrefix+"target"] = req.CreateCVDRequest.Target
	}
	ctx := context.Background()
	computeReq := &computepb.InsertInstanceRequest{
		Project: m.config.GCPConfig.ProjectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name:           proto.String(namePrefix + newUUIDString()),
			MachineType:    proto.String(req.CreateHostInstanceRequest.GCP.MachineType),
			MinCpuPlatform: proto.String(req.CreateHostInstanceRequest.GCP.MinCPUPlatform),
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(int64(req.CreateHostInstanceRequest.GCP.DiskSizeGB)),
						SourceImage: proto.String(m.config.GCPConfig.SourceImage),
					},
					Boot: proto.Bool(true),
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String(buildDefaultNetworkName(m.config.GCPConfig.ProjectID)),
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

func (m *GCPInstanceManager) Close() error {
	return m.client.Close()
}

// TODO(b/226935747) Have more thorough validation error in Instance Manager.
var ErrBadCreateHostRequest = NewBadRequestError("invalid CreateHostRequest", nil)

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.CreateCVDRequest != nil {
		if r.CreateCVDRequest.BuildID == "" {
			return ErrBadCreateHostRequest
		}
		if r.CreateCVDRequest.Target == "" {
			return ErrBadCreateHostRequest
		}
	}
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
