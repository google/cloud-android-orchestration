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
	client "github.com/google/cloud-android-orchestration/pkg/client"

	"github.com/spf13/cobra"
)

const (
	buildIDFlag = "build_id"
	targetFlag  = "target"
)

type createCVDFlags struct {
	*subCommandFlags
	BuildID string
	Target  string
	Host    string
}

func newCVDCommand(cfgFlags *configFlags) *cobra.Command {
	cvdFlags := &subCommandFlags{
		cfgFlags,
		false,
	}
	createFlags := &createCVDFlags{subCommandFlags: cvdFlags}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a CVD.",
		RunE: func(c *cobra.Command, args []string) error {
			return runCreateCVDCommand(createFlags, c, args)
		},
	}
	create.Flags().StringVar(&createFlags.Host, hostFlag, "", "Specifies the host")
	create.MarkFlagRequired(hostFlag)
	create.Flags().StringVar(&createFlags.BuildID, buildIDFlag, "", "Android build identifier")
	create.MarkFlagRequired(buildIDFlag)
	create.Flags().StringVar(&createFlags.Target, targetFlag, "aosp_cf_x86_64_phone-userdebug",
		"Android build target")
	cvd := &cobra.Command{
		Use:   "cvd",
		Short: "Work with CVDs",
	}
	addCommonSubcommandFlags(cvd, cvdFlags)
	cvd.AddCommand(create)
	return cvd
}

func runCreateCVDCommand(flags *createCVDFlags, c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(flags.subCommandFlags, c)
	if err != nil {
		return err
	}
	req := client.CreateCVDRequest{
		CVD: &client.CVD{
			BuildInfo: &client.BuildInfo{
				BuildID: flags.BuildID,
				Target:  flags.Target,
			},
		},
	}
	cvd, err := apiClient.CreateCVD(flags.Host, &req)
	if err != nil {
		return err
	}
	c.Printf("%s\n", cvd.Name)
	return nil
}
