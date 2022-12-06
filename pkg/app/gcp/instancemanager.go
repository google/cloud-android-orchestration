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
	"path"
	"regexp"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
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
		Name: m.InstanceNameGenerator.NewName(),
		// This is required in the format: "zones/zone/machineTypes/machine-type".
		// Read more: https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert#request-body
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", zone, req.HostInstance.GCP.MachineType),
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
		AdvancedMachineFeatures: &compute.AdvancedMachineFeatures{
			EnableNestedVirtualization: true,
		},
	}
	op, err := m.Service.Instances.
		Insert(m.Config.GCP.ProjectID, zone, payload).
		Context(context.TODO()).
		Do()
	if err != nil {
		return nil, toAppError(err)
	}
	return &apiv1.Operation{Name: op.Name, Done: op.Status == operationStatusDone}, nil
}

const listHostsRequestMaxResultsLimit uint32 = 500

func (m *InstanceManager) ListHosts(zone string, user app.UserInfo, req *app.ListHostsRequest) (*apiv1.ListHostsResponse, error) {
	var maxResults uint32
	if req.MaxResults <= listHostsRequestMaxResultsLimit {
		maxResults = req.MaxResults
	} else {
		maxResults = listHostsRequestMaxResultsLimit
	}
	statusFilterExpr := "status=RUNNING"
	ownerFilterExpr := fmt.Sprintf("labels.%s:%s", labelAcloudCreatedBy, user.Username())
	res, err := m.Service.Instances.
		List(m.Config.GCP.ProjectID, zone).
		Context(context.TODO()).
		MaxResults(int64(maxResults)).
		PageToken(req.PageToken).
		Filter(fmt.Sprintf("%s AND %s", ownerFilterExpr, statusFilterExpr)).
		Do()
	if err != nil {
		return nil, toAppError(err)
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

func (m *InstanceManager) DeleteHost(zone string, user app.UserInfo, name string) (*apiv1.Operation, error) {
	nameFilterExpr := "name=" + name
	ownerFilterExpr := fmt.Sprintf("labels.%s:%s", labelAcloudCreatedBy, user.Username())
	res, err := m.Service.Instances.
		List(m.Config.GCP.ProjectID, zone).
		Context(context.TODO()).
		Filter(fmt.Sprintf("%s AND %s", nameFilterExpr, ownerFilterExpr)).
		Do()
	if err != nil {
		return nil, toAppError(err)
	}
	if len(res.Items) == 0 {
		return nil, app.NewBadRequestError(fmt.Sprintf("Host instance %q not found.", name), nil)
	}
	op, err := m.Service.Instances.
		Delete(m.Config.GCP.ProjectID, zone, name).
		Context(context.TODO()).
		Do()
	if err != nil {
		return nil, toAppError(err)
	}
	return &apiv1.Operation{Name: op.Name, Done: op.Status == operationStatusDone}, nil
}

func (m *InstanceManager) WaitOperation(zone string, user app.UserInfo, name string) (interface{}, error) {
	op, err := m.Service.ZoneOperations.Wait(m.Config.GCP.ProjectID, zone, name).Do()
	if err != nil {
		return nil, toAppError(err)
	}
	if op.Status != operationStatusDone {
		return nil, app.NewServiceUnavailableError("Wait for operation timed out", nil)
	}
	getter := opResultGetter{Service: m.Service, Op: op}
	return getter.Get()
}

func (m *InstanceManager) getHostInstance(zone string, host string) (*compute.Instance, error) {
	return m.Service.Instances.
		Get(m.Config.GCP.ProjectID, zone, host).
		Context(context.TODO()).
		Do()
}

func validateRequest(r *apiv1.CreateHostRequest) error {
	if r.HostInstance == nil ||
		r.HostInstance.Name != "" ||
		r.HostInstance.BootDiskSizeGB != 0 ||
		r.HostInstance.GCP == nil ||
		r.HostInstance.GCP.MachineType == "" {
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
		Name:           in.Name,
		BootDiskSizeGB: in.Disks[0].DiskSizeGb,
		GCP: &apiv1.GCPInstance{
			MachineType: path.Base(in.MachineType),
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

type opResultGetter struct {
	Service *compute.Service
	Op      *compute.Operation
}

func (g *opResultGetter) Get() (interface{}, error) {
	done := g.Op.Status == operationStatusDone
	if !done {
		return nil, app.NewInternalError("cannot get the result of an operation that is not done yet", nil)
	}
	if g.Op.Error != nil {
		return nil, &app.AppError{
			Msg:        g.Op.HttpErrorMessage,
			StatusCode: int(g.Op.HttpErrorStatusCode),
			Err:        fmt.Errorf("gcp operation failed: %+v", g.Op),
		}
	}
	if g.Op.OperationType == "delete" && instanceTargetLinkRe.MatchString(g.Op.TargetLink) {
		return struct{}{}, nil
	}
	if g.Op.OperationType == "insert" && instanceTargetLinkRe.MatchString(g.Op.TargetLink) {
		return g.buildCreateInstanceResult()
	}
	return nil, app.NewNotFoundError("operation result not found", nil)
}

func (g *opResultGetter) buildCreateInstanceResult() (*apiv1.HostInstance, error) {
	matches := instanceTargetLinkRe.FindStringSubmatch(g.Op.TargetLink)
	if len(matches) != 4 {
		err := fmt.Errorf("invalid target link for instance insert operation: %q", g.Op.TargetLink)
		return nil, err
	}
	ins, err := g.Service.Instances.
		Get(matches[1], matches[2], matches[3]).
		Context(context.TODO()).
		Do()
	if err != nil {
		return nil, err
	}
	return BuildHostInstance(ins)
}

// Converts compute API errors to AppError if relevant, return the same error otherwise
func toAppError(err error) error {
	apiErr, ok := err.(*googleapi.Error)
	if !ok {
		return err
	}
	return &app.AppError{
		Msg:        apiErr.Message,
		StatusCode: apiErr.Code,
		Err:        err,
	}
}
