// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/client"
)

type CreateHostOpts struct {
	GCP CreateGCPHostOpts
}

func (f *CreateHostOpts) Update(s *Service) {
	f.GCP.MachineType = s.Host.GCP.MachineType
	f.GCP.MinCPUPlatform = s.Host.GCP.MinCPUPlatform
	f.GCP.BootDiskSizeGB = s.Host.GCP.BootDiskSizeGB
}

type CreateGCPHostOpts struct {
	MachineType        string
	MinCPUPlatform     string
	BootDiskSizeGB     int64
	AcceleratorConfigs []acceleratorConfig
}

func createHost(srvClient client.Client, opts CreateHostOpts) (*apiv1.HostInstance, error) {
	req := apiv1.CreateHostRequest{
		HostInstance: &apiv1.HostInstance{
			GCP: &apiv1.GCPInstance{
				MachineType:    opts.GCP.MachineType,
				MinCPUPlatform: opts.GCP.MinCPUPlatform,
				BootDiskSizeGB: opts.GCP.BootDiskSizeGB,
			},
		},
	}
	if len(opts.GCP.AcceleratorConfigs) != 0 {
		s := []*apiv1.AcceleratorConfig{}
		for _, c := range opts.GCP.AcceleratorConfigs {
			c := &apiv1.AcceleratorConfig{AcceleratorCount: int64(c.Count), AcceleratorType: c.Type}
			s = append(s, c)
		}
		req.HostInstance.GCP.AcceleratorConfigs = s
	}
	return srvClient.CreateHost(&req)
}

func hostnames(srvClient client.Client) ([]string, error) {
	hosts, err := srvClient.ListHosts()
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, h := range hosts.Items {
		result = append(result, h.Name)
	}
	return result, nil
}

func findHost(srvClient client.Client, name string) (*apiv1.HostInstance, error) {
	hosts, err := srvClient.ListHosts()
	if err != nil {
		return nil, fmt.Errorf("error listing hosts: %w", err)
	}
	for _, host := range hosts.Items {
		if host.Name == name {
			return host, nil
		}
	}
	return nil, fmt.Errorf("name not found: %s", name)
}
