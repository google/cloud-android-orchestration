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

	"github.com/google/cloud-android-orchestration/pkg/client"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	"github.com/hashicorp/go-multierror"
)

type CVD struct {
	ServiceRootEndpoint string `json:"service_root_endpoint"`
	Host                string `json:"host"`
	Name                string `json:"name"`
}

type CVDInfo struct {
	CVD
	Status   string
	Displays []string
}

func NewCVDInfo(url, host string, cvd *hoapi.CVD) *CVDInfo {
	return &CVDInfo{
		CVD: CVD{
			ServiceRootEndpoint: url,
			Host:                host,
			Name:                cvd.Name,
		},
		Status:   cvd.Status,
		Displays: cvd.Displays,
	}
}

type CreateCVDOpts struct {
	Host       string
	Branch     string
	BuildID    string
	Target     string
	LocalImage bool
}

func createCVD(service client.Service, createOpts CreateCVDOpts) (*CVDInfo, error) {
	creator := &cvdCreator{
		Service: service,
		Opts:    createOpts,
	}
	cvd, err := creator.Create()
	if err != nil {
		return nil, fmt.Errorf("Failed to create cvd: %w", err)
	}
	return NewCVDInfo(service.RootURI(), createOpts.Host, cvd), nil
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
				UserBuildSource: &hoapi.UserBuildSource{
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
				AndroidCIBuildSource: &hoapi.AndroidCIBuildSource{
					MainBuild: &hoapi.AndroidCIBuild{
						Branch:  c.Opts.Branch,
						BuildID: c.Opts.BuildID,
						Target:  c.Opts.Target,
					},
				},
			},
		},
	}
	return c.Service.CreateCVD(c.Opts.Host, &req)
}

type cvdListResult struct {
	Result []*CVDInfo
	Error  error
}

func listAllCVDs(service client.Service) ([]*CVDInfo, error) {
	hl, err := service.ListHosts()
	if err != nil {
		return nil, fmt.Errorf("Error listing hosts: %w", err)
	}
	var hosts []string
	for _, host := range hl.Items {
		hosts = append(hosts, host.Name)
	}
	var chans []chan cvdListResult
	for _, host := range hosts {
		ch := make(chan cvdListResult)
		chans = append(chans, ch)
		go func(name string, ch chan<- cvdListResult) {
			cvds, err := listHostCVDs(service, name)
			ch <- cvdListResult{Result: cvds, Error: err}
		}(host, ch)
	}
	var merr error
	var cvds []*CVDInfo
	for i, ch := range chans {
		host := hosts[i]
		result := <-ch
		if result.Error != nil {
			merr = multierror.Append(merr, fmt.Errorf("lists cvds for host %q failed: %w", host, err))
			continue
		}
		cvds = append(cvds, result.Result...)
	}
	return cvds, merr
}

func listHostCVDs(service client.Service, host string) ([]*CVDInfo, error) {
	cvds, err := service.ListCVDs(host)
	if err != nil {
		return nil, err
	}
	ret := make([]*CVDInfo, len(cvds))
	for i, c := range cvds {
		ret[i] = NewCVDInfo(service.RootURI(), host, c)
	}
	return ret, nil
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
