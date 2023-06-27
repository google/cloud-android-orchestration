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
	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/client"
)

type CreateHostOpts struct {
	GCP CreateGCPHostOpts
}

type CreateGCPHostOpts struct {
	MachineType    string
	MinCPUPlatform string
}

func createHost(service client.Service, opts CreateHostOpts) (*apiv1.HostInstance, error) {
	req := apiv1.CreateHostRequest{
		HostInstance: &apiv1.HostInstance{
			GCP: &apiv1.GCPInstance{
				MachineType:    opts.GCP.MachineType,
				MinCPUPlatform: opts.GCP.MinCPUPlatform,
			},
		},
	}
	return service.CreateHost(&req)
}

func hostnames(service client.Service) ([]string, error) {
	hosts, err := service.ListHosts()
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, h := range hosts.Items {
		result = append(result, h.Name)
	}
	return result, nil
}
