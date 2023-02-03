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
	"os"
	"strings"
	"sync"

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

type CreateCVDOpts struct {
	BuildID    string
	Target     string
	Host       string
	LocalImage bool
}

type CreateCVDFlags struct {
	*CommonSubcmdFlags
	*CreateCVDOpts
}

type ListCVDsFlags struct {
	*CommonSubcmdFlags
	Host string
}

func newCVDCommand(opts *subCommandOpts) *cobra.Command {
	cvdFlags := &CommonSubcmdFlags{CVDRemoteFlags: opts.RootFlags}
	createFlags := &CreateCVDFlags{CommonSubcmdFlags: cvdFlags, CreateCVDOpts: &CreateCVDOpts{}}
	create := &cobra.Command{
		Use:   "create",
		Short: "Creates a CVD.",
		RunE: func(c *cobra.Command, args []string) error {
			return createCVD(c, createFlags, opts)
		},
	}
	create.Flags().StringVar(&createFlags.Host, hostFlag, "", "Specifies the host")
	create.MarkFlagRequired(hostFlag)
	create.Flags().StringVar(&createFlags.BuildID, buildIDFlag, "", "Android build identifier")
	create.Flags().StringVar(&createFlags.Target, targetFlag, "aosp_cf_x86_64_phone-userdebug",
		"Android build target")
	create.Flags().BoolVar(&createFlags.LocalImage, localImageFlag, false,
		"Builds a CVD with image files built locally, the required files are https://cs.android.com/android/platform/superproject/+/master:device/google/cuttlefish/required_images and cvd-host-packages.tar.gz")
	create.MarkFlagsMutuallyExclusive(buildIDFlag, localImageFlag)
	create.MarkFlagsMutuallyExclusive(targetFlag, localImageFlag)
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
	action := &createCVDAction{
		Service: service,
		Opts:    *flags.CreateCVDOpts,
	}
	cvd, err := action.Execute()
	if err != nil {
		return fmt.Errorf("Failed to create cvd: %w", err)
	}
	output := CVDOutput{
		BaseURL: buildBaseURL(flags.CVDRemoteFlags),
		Host:    flags.Host,
		CVD:     cvd,
	}
	c.Printf("%s\n", output.String())
	return nil
}

type createCVDAction struct {
	Service client.Service
	Opts    CreateCVDOpts
}

func (a *createCVDAction) Execute() (*hoapi.CVD, error) {
	if a.Opts.LocalImage {
		return a.createCVDFromLocalBuild()
	} else {
		return a.createCVDFromAndroidCI()
	}
}

func (a *createCVDAction) createCVDFromLocalBuild() (*hoapi.CVD, error) {
	vars, err := GetAndroidEnvVarValues()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving Android Build environment variables: %w", err)
	}
	names, err := ListLocalImageRequiredFiles(vars)
	if err != nil {
		return nil, fmt.Errorf("Error building list of required image files: %w", err)
	}
	uploadDir, err := a.Service.CreateUpload(a.Opts.Host)
	if err != nil {
		return nil, err
	}
	if err := a.Service.UploadFiles(a.Opts.Host, uploadDir, names); err != nil {
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
	return a.Service.CreateCVD(a.Opts.Host, &req)
}

func (a *createCVDAction) createCVDFromAndroidCI() (*hoapi.CVD, error) {
	req := hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuild: &hoapi.AndroidCIBuild{
					BuildID: a.Opts.BuildID,
					Target:  a.Opts.Target,
				},
			},
		},
	}
	return a.Service.CreateCVD(a.Opts.Host, &req)
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
	var wg sync.WaitGroup
	var mu sync.Mutex
	var merr error
	for _, host := range hosts {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			hostCVDs, err := service.ListCVDs(name)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				merr = multierror.Append(merr,
					fmt.Errorf("lists cvds for host %q failed: %w", name, err))
				return
			}
			for _, cvd := range hostCVDs {
				output := CVDOutput{
					BaseURL: buildBaseURL(flags.CVDRemoteFlags),
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
