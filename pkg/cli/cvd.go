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
	"sync"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	"github.com/hashicorp/go-multierror"
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

type listCVDsFlags struct {
	*subCommandFlags
	Host string
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
	listFlags := &listCVDsFlags{subCommandFlags: cvdFlags}
	list := &cobra.Command{
		Use:   "list",
		Short: "List CVDs",
		RunE: func(c *cobra.Command, args []string) error {
			return runListCVDsCommand(listFlags, c, args)
		},
	}
	list.Flags().StringVar(&listFlags.Host, hostFlag, "", "Specifies the host")
	cvd := &cobra.Command{
		Use:   "cvd",
		Short: "Work with CVDs",
	}
	addCommonSubcommandFlags(cvd, cvdFlags)
	cvd.AddCommand(create)
	cvd.AddCommand(list)
	return cvd
}

func runCreateCVDCommand(flags *createCVDFlags, c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(flags.subCommandFlags, c)
	if err != nil {
		return err
	}
	req := hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuild: &hoapi.AndroidCIBuild{
					BuildID: flags.BuildID,
					Target:  flags.Target,
				},
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

func runListCVDsCommand(flags *listCVDsFlags, c *cobra.Command, _ []string) error {
	apiClient, err := buildAPIClient(flags.subCommandFlags, c)
	if err != nil {
		return err
	}
	var hosts []string
	if flags.Host != "" {
		hosts = append(hosts, flags.Host)
	} else {
		res, err := apiClient.ListHosts()
		if err != nil {
			return err
		}
		for _, host := range res.Items {
			hosts = append(hosts, host.Name)
		}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var merr error
	for _, host := range hosts {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			hostCVDs, err := apiClient.ListCVDs(name)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				merr = multierror.Append(merr,
					fmt.Errorf("lists cvds for host %q failed: %w", name, err))
				return
			}
			for _, cvd := range hostCVDs {
				output := CVDOutput{
					BaseURL: buildBaseURL(flags.configFlags),
					Host:    name,
					CVD:     cvd,
				}
				c.Printf("%s\n", output.String())
			}
		}(host)
	}
	wg.Wait()
	return merr
}

type CVDOutput struct {
	BaseURL string
	Host    string
	CVD     *hoapi.CVD
}

func (o *CVDOutput) String() string {
	res := fmt.Sprintf("%s (%s)", o.CVD.Name, o.Host)
	res += "\n  " + "Status: " + o.CVD.Status
	res += "\n  " + "Displays: " + fmt.Sprintf("%v", o.CVD.Displays)
	res += "\n  " + "WebRTCStream: " +
		fmt.Sprintf("%s/hosts/%s/devices/%s/files/client.html", o.BaseURL, o.Host, o.CVD.Name)
	res += "\n  " + "Logs: " +
		fmt.Sprintf("%s/hosts/%s/cvds/%s/logs/", o.BaseURL, o.Host, o.CVD.Name)
	return res
}
