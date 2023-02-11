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
	"io"
	"os"
	"strings"

	"github.com/google/cloud-android-orchestration/pkg/client"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"
)

const (
	buildIDFlag    = "build_id"
	targetFlag     = "target"
	localImageFlag = "local_image"
)

const (
	hostGCPMachineTypeFlag    = "host_gcp_machine_type"
	hostGCPMinCPUPlatformFlag = "host_gcp_min_cpu_platform"
)

type CreateCVDOpts struct {
	Host       string
	BuildID    string
	Target     string
	LocalImage bool
}

type CreateCVDFlags struct {
	*CommonSubcmdFlags
	*CreateCVDOpts
	*CreateHostOpts
}

type ListCVDsFlags struct {
	*CommonSubcmdFlags
	Host string
}

func newCVDCommand(opts *subCommandOpts) *cobra.Command {
	cvdFlags := &CommonSubcmdFlags{CVDRemoteFlags: opts.RootFlags}
	createFlags := &CreateCVDFlags{
		CommonSubcmdFlags: cvdFlags,
		CreateCVDOpts:     &CreateCVDOpts{},
		CreateHostOpts:    &CreateHostOpts{},
	}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a CVD.",
		RunE: func(c *cobra.Command, args []string) error {
			return createCVD(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.Host, hostFlag, "", "Specifies the host")
	create.Flags().StringVar(&createFlags.BuildID, buildIDFlag, "", "Android build identifier")
	create.Flags().StringVar(&createFlags.Target, targetFlag, "aosp_cf_x86_64_phone-userdebug",
		"Android build target")
	create.Flags().BoolVar(&createFlags.LocalImage, localImageFlag, false,
		"Builds a CVD with image files built locally, the required files are https://cs.android.com/android/platform/superproject/+/master:device/google/cuttlefish/required_images and cvd-host-packages.tar.gz")
	create.MarkFlagsMutuallyExclusive(buildIDFlag, localImageFlag)
	create.MarkFlagsMutuallyExclusive(targetFlag, localImageFlag)
	// Host flags
	createHostFlags := []struct {
		ValueRef *string
		Name     string
		Default  string
		Desc     string
	}{
		{
			ValueRef: &createFlags.GCP.MachineType,
			Name:     gcpMachineTypeFlag,
			Default:  opts.InitialConfig.Host.GCP.MachineType,
			Desc:     gcpMachineTypeFlagDesc,
		},
		{
			ValueRef: &createFlags.GCP.MinCPUPlatform,
			Name:     gcpMinCPUPlatformFlag,
			Default:  opts.InitialConfig.Host.GCP.MinCPUPlatform,
			Desc:     gcpMinCPUPlatformFlagDesc,
		},
	}
	for _, f := range createHostFlags {
		name := "host_" + f.Name
		create.Flags().StringVar(f.ValueRef, name, f.Default, f.Desc)
		create.MarkFlagsMutuallyExclusive(hostFlag, name)
	}
	listFlags := &ListCVDsFlags{CommonSubcmdFlags: cvdFlags}
	list := &cobra.Command{
		Use:   "list",
		Short: "List CVDs",
		RunE: func(c *cobra.Command, args []string) error {
			return listCVDs(c, listFlags, opts)
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

func createCVD(c *cobra.Command, flags *CreateCVDFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
	if err != nil {
		return fmt.Errorf("Failed to build service instance: %w", err)

	}
	host := flags.CreateCVDOpts.Host
	if host == "" {
		ins, err := createHost(service, *flags.CreateHostOpts)
		if err != nil {
			return fmt.Errorf("Failed to create host: %w", err)
		}

		host = ins.Name
	}
	createOpts := *flags.CreateCVDOpts
	createOpts.Host = host
	creator := &cvdCreator{
		Service: service,
		Opts:    *flags.CreateCVDOpts,
	}
	cvd, err := creator.Create()
	if err != nil {
		return fmt.Errorf("Failed to create cvd: %w", err)
	}
	rootEndpoint := buildServiceRootEndpoint(flags.ServiceURL, flags.Zone)
	printCVDs(c.OutOrStdout(), rootEndpoint, host, []*hoapi.CVD{cvd})
	return nil
}

type cvdCreator struct {
	Service client.Service
	Opts    CreateCVDOpts
}

func (c *cvdCreator) Create() (*hoapi.CVD, error) {
	if c.Opts.LocalImage {
		return c.createCVDFromLocalBuild()
	} else {
		return c.createCVDFromAndroidCI()
	}
}

func (c *cvdCreator) createCVDFromLocalBuild() (*hoapi.CVD, error) {
	vars, err := GetAndroidEnvVarValues()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving Android Build environment variables: %w", err)
	}
	names, err := ListLocalImageRequiredFiles(vars)
	if err != nil {
		return nil, fmt.Errorf("Error building list of required image files: %w", err)
	}
	uploadDir, err := c.Service.CreateUpload(c.Opts.Host)
	if err != nil {
		return nil, err
	}
	if err := c.Service.UploadFiles(c.Opts.Host, uploadDir, names); err != nil {
		return nil, err
	}
	req := hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				UserBuild: &hoapi.UserBuild{
					ArtifactsDir: uploadDir,
				},
			},
		},
	}
	return c.Service.CreateCVD(c.Opts.Host, &req)
}

func (c *cvdCreator) createCVDFromAndroidCI() (*hoapi.CVD, error) {
	req := hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuild: &hoapi.AndroidCIBuild{
					BuildID: c.Opts.BuildID,
					Target:  c.Opts.Target,
				},
			},
		},
	}
	return c.Service.CreateCVD(c.Opts.Host, &req)
}

type cvdListResult struct {
	Result []*hoapi.CVD
	Error  error
}

func listCVDs(c *cobra.Command, flags *ListCVDsFlags, opts *subCommandOpts) error {
	service, err := opts.ServiceBuilder(flags.CommonSubcmdFlags, c)
	if err != nil {
		return err
	}
	var hosts []string
	if flags.Host != "" {
		hosts = append(hosts, flags.Host)
	} else {
		res, err := service.ListHosts()
		if err != nil {
			return fmt.Errorf("Error listing hosts: %w", err)
		}
		for _, host := range res.Items {
			hosts = append(hosts, host.Name)
		}
	}
	var chans []chan cvdListResult
	for _, host := range hosts {
		ch := make(chan cvdListResult)
		chans = append(chans, ch)
		go func(name string, ch chan<- cvdListResult) {
			cvds, err := service.ListCVDs(name)
			ch <- cvdListResult{Result: cvds, Error: err}
		}(host, ch)
	}
	rootEndpoint := buildServiceRootEndpoint(flags.ServiceURL, flags.Zone)
	var merr error
	for i, ch := range chans {
		host := hosts[i]
		result := <-ch
		if result.Error != nil {
			merr = multierror.Append(merr, fmt.Errorf("lists cvds for host %q failed: %w", host, err))
			continue
		}
		printCVDs(c.OutOrStdout(), rootEndpoint, host, result.Result)
	}
	return merr
}

type CVDOutput struct {
	ServiceRootEndpoint string
	Host                string
	CVD                 *hoapi.CVD
}

func (o *CVDOutput) String() string {
	res := fmt.Sprintf("%s (%s)", o.CVD.Name, o.Host)
	res += "\n  " + "Status: " + o.CVD.Status
	res += "\n  " + "Displays: " + fmt.Sprintf("%v", o.CVD.Displays)
	res += "\n  " + "WebRTCStream: " + client.BuildWebRTCStreamURL(o.ServiceRootEndpoint, o.Host, o.CVD.Name)
	res += "\n  " + "Logs: " + client.BuildCVDLogsURL(o.ServiceRootEndpoint, o.Host, o.CVD.Name)
	return res
}

func printCVDs(writer io.Writer, rootEndpoint, host string, cvds []*hoapi.CVD) {
	for _, cvd := range cvds {
		o := CVDOutput{
			ServiceRootEndpoint: rootEndpoint,
			Host:                host,
			CVD:                 cvd,
		}
		fmt.Fprintln(writer, o.String())
	}
}

const RequiredImagesFilename = "device/google/cuttlefish/required_images"

type MissingEnvVarErr string

func (s MissingEnvVarErr) Error() string {
	return fmt.Sprintf("Missing environment variable: %q", string(s))
}

const CVDHostPackageName = "cvd-host_package.tar.gz"

const (
	AndroidBuildTopVarName   = "ANDROID_BUILD_TOP"
	AndroidHostOutVarName    = "ANDROID_HOST_OUT"
	AndroidProductOutVarName = "ANDROID_PRODUCT_OUT"
)

type AndroidEnvVars struct {
	BuildTop   string
	ProductOut string
	HostOut    string
}

func GetAndroidEnvVarValues() (AndroidEnvVars, error) {
	androidEnvVars := []string{AndroidBuildTopVarName, AndroidProductOutVarName, AndroidHostOutVarName}
	for _, name := range androidEnvVars {
		if _, ok := os.LookupEnv(name); !ok {
			return AndroidEnvVars{}, MissingEnvVarErr(name)
		}
	}
	return AndroidEnvVars{
		BuildTop:   os.Getenv(AndroidBuildTopVarName),
		HostOut:    os.Getenv(AndroidHostOutVarName),
		ProductOut: os.Getenv(AndroidProductOutVarName),
	}, nil
}

func ListLocalImageRequiredFiles(vars AndroidEnvVars) ([]string, error) {
	reqImgsFilename := vars.BuildTop + "/" + RequiredImagesFilename
	f, err := os.Open(reqImgsFilename)
	if err != nil {
		return nil, fmt.Errorf("Error opening the required images list file: %w", err)
	}
	defer f.Close()
	content, err := os.ReadFile(reqImgsFilename)
	if err != nil {
		return nil, fmt.Errorf("Error reading the required images list file: %w", err)
	}
	contentStr := strings.TrimRight(string(content), "\n")
	lines := strings.Split(contentStr, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, vars.ProductOut+"/"+line)
	}
	result = append(result, vars.HostOut+"/"+CVDHostPackageName)
	return result, nil
}
