package main

import (
	imtypes "cloud-android-orchestration/api/instancemanager/v1"
	compute "cloud.google.com/go/compute/apiv1"
	"context"
	"errors"
	"github.com/google/uuid"
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

var newUUIDString = func() string {
	return uuid.New().String()
}

// GCP implementation of the instance manager.
type GcpIM struct {
	client *compute.InstancesClient
}

func NewGcpIM(client *compute.InstancesClient) *GcpIM {
	result := &GcpIM{
		client: client,
	}
	return result
}

func (m *GcpIM) DeviceFromId(name string, _ UserInfo) (DeviceDesc, error) {
	return DeviceDesc{"127.0.0.1", "cvd-1"}, nil
}

func (m *GcpIM) InsertHost(zone string, req *imtypes.InsertHostRequest, user UserInfo) (*imtypes.Operation, error) {
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
						SourceImage: proto.String("projects/cloud-android-releases/global/images/cuttlefish-google-vsoc-0-9-21"),
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
				labelPrefix + "creator":  user.Username(),
				labelPrefix + "build_id": req.CVDInfo.BuildID,
				labelPrefix + "target":   req.CVDInfo.Target,
			},
		},
	}
	op, err := m.client.Insert(ctx, computeReq)
	if err != nil {
		return nil, err
	}
	result := &imtypes.Operation{
		Name: op.Name(),
		Done: op.Done(),
	}
	return result, nil
}

// TODO(b/226935747) Have more thorough validation error in Instance Manager.
var ErrBadInsertHostRequest = errors.New("invalid InsertHostRequest object")

func validateRequest(r *imtypes.InsertHostRequest) error {
	if r.GetCVDInfo() == nil {
		return ErrBadInsertHostRequest
	}
	if r.GetCVDInfo().GetBuildID() == "" {
		return ErrBadInsertHostRequest
	}
	if r.GetCVDInfo().GetTarget() == "" {
		return ErrBadInsertHostRequest
	}
	if r.GetHostInfo() == nil {
		return ErrBadInsertHostRequest
	}
	if r.GetHostInfo().GetGCP() == nil {
		return ErrBadInsertHostRequest
	}
	if r.GetHostInfo().GetGCP().GetDiskSizeGB() == 0 {
		return ErrBadInsertHostRequest
	}
	if r.GetHostInfo().GetGCP().GetMachineType() == "" {
		return ErrBadInsertHostRequest
	}
	return nil
}
