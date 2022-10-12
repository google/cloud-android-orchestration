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
	"net/url"
	"regexp"

	apiv1 "cloud-android-orchestration/api/v1"
	"cloud-android-orchestration/app"

	"google.golang.org/api/compute/v1"
)

const (
	labelPrefix          = "cf-"
	labelAcloudCreatedBy = "created_by" // required for acloud backwards compatibility
	labelCreatedBy       = labelPrefix + "created_by"
)

// GCP implementation of the instance manager.
type InstanceManager struct {
	Config                app.IMConfig
	Service               *compute.Service
	InstanceNameGenerator NameGenerator
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
	return instance.NetworkInterfaces[0].NetworkIP, nil
}

const (
	hostURLScheme = "http"
	hostURLPort   = 1080
)

func (m *InstanceManager) GetHostURL(zone string, host string) (*url.URL, error) {
	addr, err := m.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	return url.Parse(fmt.Sprintf("%s://%s:%d", hostURLScheme, addr, hostURLPort))
}

const operationStatusDone = "DONE"

func (m *InstanceManager) CreateHost(zone string, req *apiv1.CreateHostRequest, user app.UserInfo) (*apiv1.Operation, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	labels := map[string]string{
		labelAcloudCreatedBy: user.Username(),
		labelCreatedBy:       user.Username(),
	}
	payload := &compute.Instance{
		Name:           m.InstanceNameGenerator.NewName(),
		MachineType:    req.CreateHostInstanceRequest.GCP.MachineType,
		MinCpuPlatform: req.CreateHostInstanceRequest.GCP.MinCPUPlatform,
		Disks: []*compute.AttachedDisk{
			{
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: m.Config.GCP.HostImage,
				},
				Boot: true,
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Name: buildDefaultNetworkName(m.Config.GCP.ProjectID),
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Labels: labels,
	}
	op, err := m.Service.Instances.
		Insert(m.Config.GCP.ProjectID, zone, payload).
		Context(context.TODO()).
		Do()
	if err != nil {
		return nil, err
	}
	return m.buildOperation(op)
}

const listHostsRequestMaxResultsLimit uint32 = 500

func (m *InstanceManager) ListHosts(zone string, user app.UserInfo, req *app.ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	var maxResults uint32
	if req.MaxResults <= listHostsRequestMaxResultsLimit {
		maxResults = req.MaxResults
	} else {
		maxResults = listHostsRequestMaxResultsLimit
	}
	res, err := m.Service.Instances.
		List(m.Config.GCP.ProjectID, zone).
		Context(context.TODO()).
		MaxResults(int64(maxResults)).
		PageToken(req.PageToken).
		Filter(fmt.Sprintf("labels.%s:%s", labelAcloudCreatedBy, user.Username())).
		Do()
	if err != nil {
		return nil, err
	}
	var items []*apiv1.HostInstance
	for _, i := range res.Items {
		hi, err := BuildHostInstance(i)
		if err != nil {
			return nil, err
		}
		items = append(items, hi)
	}
	return &apiv1.ListHostsResponse{
		Items:         items,
		NextPageToken: res.NextPageToken,
	}, nil
}

func (m *InstanceManager) WaitOperation(zone string, user app.UserInfo, name string) (*apiv1.Operation, error) {
	op, err := m.Service.ZoneOperations.Wait(m.Config.GCP.ProjectID, zone, name).Do()
	if err != nil {
		return nil, err
	}
	return m.buildOperation(op)
}

func (m *InstanceManager) getHostInstance(zone string, host string) (*compute.Instance, error) {
	return m.Service.Instances.
		Get(m.Config.GCP.ProjectID, zone, host).
		Context(context.TODO()).
		Do()
}

func (m *InstanceManager) buildOperation(op *compute.Operation) (*apiv1.Operation, error) {
	opBuilder := operationBuilder{Service: m.Service, Operation: op}
	return opBuilder.Build()
}

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.CreateHostInstanceRequest == nil ||
		r.CreateHostInstanceRequest.GCP == nil ||
		r.CreateHostInstanceRequest.GCP.BootDiskSizeGB != 0 ||
		r.CreateHostInstanceRequest.GCP.MachineType == "" {
		return app.NewBadRequestError("invalid CreateHostRequest", nil)
	}
	return nil
}

func buildDefaultNetworkName(projectID string) string {
	return fmt.Sprintf("projects/%s/global/networks/default", projectID)
}

func BuildHostInstance(in *compute.Instance) (*apiv1.HostInstance, error) {
	disksLen := len(in.Disks)
	if disksLen == 0 {
		log.Printf("invalid host instance %q: has 0 disks", in.SelfLink)
		return nil, app.NewInternalError("invalid host instance: has 0 disks", nil)
	}
	if disksLen > 1 {
		log.Printf("invalid host instance %q: has %d (more than one) disks", in.SelfLink, disksLen)
	}
	return &apiv1.HostInstance{
		Name: in.Name,
		GCP: &apiv1.GCPInstance{
			BootDiskSizeGB: in.Disks[0].DiskSizeGb,
			MachineType:    in.MachineType,
			MinCPUPlatform: in.MinCpuPlatform,
		},
	}, nil
}

const hostInstanceNamePrefix = "cf-"

type NameGenerator interface {
	NewName() string
}

type InstanceNameGenerator struct {
	UUIDFactory func() string
}

func (g *InstanceNameGenerator) NewName() string {
	return hostInstanceNamePrefix + g.UUIDFactory()
}

var (
	instanceTargetLinkRe = regexp.MustCompile(`^https://.+/compute/v1/projects/(.+)/zones/(.+)/instances/(.+)$`)
)

type operationBuilder struct {
	Service   *compute.Service
	Operation *compute.Operation
}

func (b *operationBuilder) Build() (*apiv1.Operation, error) {
	done := b.Operation.Status == operationStatusDone
	if !done {
		return b.buildNotDone()
	}
	return b.buildDone()
}

func (b *operationBuilder) buildNotDone() (*apiv1.Operation, error) {
	return &apiv1.Operation{
		Name: b.Operation.Name,
		Done: false,
	}, nil
}

func (b *operationBuilder) buildDone() (*apiv1.Operation, error) {
	if b.Operation.Error != nil {
		return nil, &app.AppError{
			Msg:        b.Operation.HttpErrorMessage,
			StatusCode: int(b.Operation.HttpErrorStatusCode),
			Err:        fmt.Errorf("gcp operation failed: %+v", b.Operation),
		}
	}
	if b.isCreateInstance() {
		result, err := b.buildCreateInstanceResult()
		if err != nil {
			return nil, err
		}
		return &apiv1.Operation{
			Name:   b.Operation.Name,
			Done:   true,
			Result: result,
		}, nil
	}
	return nil, fmt.Errorf("not handled operation type: %v", b.Operation)
}

func (b *operationBuilder) isCreateInstance() bool {
	return b.Operation.OperationType == "insert" &&
		instanceTargetLinkRe.MatchString(b.Operation.TargetLink)
}

func (b *operationBuilder) buildCreateInstanceResult() (*apiv1.OperationResult, error) {
	matches := instanceTargetLinkRe.FindStringSubmatch(b.Operation.TargetLink)
	if len(matches) != 4 {
		err := fmt.Errorf("invalid target link for instance insert operation: %q",
			b.Operation.TargetLink)
		return nil, err
	}
	gcpInstance, err := b.Service.Instances.
		Get(matches[1], matches[2], matches[3]).
		Context(context.TODO()).
		Do()
	if err != nil {
		return nil, err
	}
	hostInstance, err := BuildHostInstance(gcpInstance)
	if err != nil {
		return nil, err
	}
	return &apiv1.OperationResult{Response: hostInstance}, nil
}
