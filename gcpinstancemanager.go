package main

import (
	"context"
	"errors"

	apiv1 "cloud-android-orchestration/api/v1"

	compute "cloud.google.com/go/compute/apiv1"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

const (
	// TODO(b/220891296): Make this configurable
	projectId   = "google.com:cloud-android-jemoreira"
	sourceImage = "projects/cloud-android-releases/global/images/cuttlefish-google-vsoc-0-9-21"
	networkName = "projects/cloud-android-jemoreira/global/networks/default"
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
	client *compute.InstancesClient
}

func NewGCPInstanceManager(opts ...option.ClientOption) (*GCPInstanceManager, error) {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &GCPInstanceManager{
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
	ctx := context.Background()
	computeReq := &computepb.InsertInstanceRequest{
		Project: projectId,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name:           proto.String(namePrefix + newUUIDString()),
			MachineType:    proto.String(req.HostInfo.GCP.MachineType),
			MinCpuPlatform: proto.String(req.HostInfo.GCP.MinCPUPlatform),
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(int64(req.HostInfo.GCP.DiskSizeGB)),
						SourceImage: proto.String(sourceImage),
					},
					Boot: proto.Bool(true),
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String(networkName),
					AccessConfigs: []*computepb.AccessConfig{
						{
							Name: proto.String("External NAT"),
							Type: proto.String(computepb.AccessConfig_ONE_TO_ONE_NAT.String()),
						},
					},
				},
			},
			Labels: map[string]string{
				"created_by":               user.Username(), // required for acloud backwards compatibility
				labelPrefix + "created_by": user.Username(),
				labelPrefix + "build_id":   req.CVDInfo.BuildID,
				labelPrefix + "target":     req.CVDInfo.Target,
			},
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
var ErrBadCreateHostRequest = errors.New("invalid CreateHostRequest object")

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.CVDInfo == nil {
		return ErrBadCreateHostRequest
	}
	if r.CVDInfo != nil {
		if r.CVDInfo.BuildID == "" {
			return ErrBadCreateHostRequest
		}
		if r.CVDInfo.Target == "" {
			return ErrBadCreateHostRequest
		}
	}
	if r.HostInfo == nil {
		return ErrBadCreateHostRequest
	}
	if r.HostInfo.GCP == nil {
		return ErrBadCreateHostRequest
	}
	if r.HostInfo.GCP.DiskSizeGB == 0 {
		return ErrBadCreateHostRequest
	}
	if r.HostInfo.GCP.MachineType == "" {
		return ErrBadCreateHostRequest
	}
	return nil
}
