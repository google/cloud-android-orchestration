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
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	Status     string
	Displays   []string
	ConnStatus *ConnStatus
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
	Host            string
	MainBuild       hoapi.AndroidCIBuild
	KernelBuild     hoapi.AndroidCIBuild
	BootloaderBuild hoapi.AndroidCIBuild
	SystemImgBuild  hoapi.AndroidCIBuild
	LocalImage      bool
	// Creates multiple instances. Only relevant if given a single build source.
	NumInstances int
}

func (o *CreateCVDOpts) AdditionalInstancesNum() uint32 {
	if o.NumInstances <= 0 {
		return 0
	}
	return uint32(o.NumInstances - 1)
}

func createCVD(service client.Service, createOpts CreateCVDOpts, statePrinter *statePrinter) ([]*CVDInfo, error) {
	creator := newCVDCreator(service, createOpts, statePrinter)
	cvds, err := creator.Create()
	if err != nil {
		return nil, fmt.Errorf("Failed to create cvd: %w", err)
	}
	result := []*CVDInfo{}
	for _, cvd := range cvds {
		result = append(result, NewCVDInfo(service.RootURI(), createOpts.Host, cvd))
	}
	return result, nil
}

type cvdCreator struct {
	service      client.Service
	opts         CreateCVDOpts
	statePrinter *statePrinter
}

func newCVDCreator(service client.Service, opts CreateCVDOpts, statePrinter *statePrinter) *cvdCreator {
	return &cvdCreator{
		service:      service,
		opts:         opts,
		statePrinter: statePrinter,
	}
}

func (c *cvdCreator) Create() ([]*hoapi.CVD, error) {
	if c.opts.LocalImage {
		return c.createCVDFromLocalBuild()
	} else {
		return c.createCVDFromAndroidCI()
	}
}

func (c *cvdCreator) createCVDFromLocalBuild() ([]*hoapi.CVD, error) {
	vars, err := GetAndroidEnvVarValues()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving Android Build environment variables: %w", err)
	}
	names, err := ListLocalImageRequiredFiles(vars)
	if err != nil {
		return nil, fmt.Errorf("Error building list of required image files: %w", err)
	}
	if err := verifyCVDHostPackageTar(vars.HostOut); err != nil {
		return nil, fmt.Errorf("Invalid cvd host package: %w", err)
	}
	names = append(names, filepath.Join(vars.HostOut, CVDHostPackageName))
	uploadDir, err := c.service.CreateUpload(c.opts.Host)
	if err != nil {
		return nil, err
	}
	if err := c.service.UploadFiles(c.opts.Host, uploadDir, names); err != nil {
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
		AdditionalInstancesNum: c.opts.AdditionalInstancesNum(),
	}
	res, err := c.service.CreateCVD(c.opts.Host, &req)
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

const (
	stateMsgFetchMainBundle = "Fetching main bundle artifacts"
	stateMsgStartCVD        = "Starting cvd and waiting for boot complete"
)

func (c *cvdCreator) createCVDFromAndroidCI() ([]*hoapi.CVD, error) {
	var mainBuild, kernelBuild, bootloaderBuild, systemImageBuild *hoapi.AndroidCIBuild
	mainBuild = &c.opts.MainBuild
	if c.opts.KernelBuild != (hoapi.AndroidCIBuild{}) {
		kernelBuild = &c.opts.KernelBuild
	}
	if c.opts.BootloaderBuild != (hoapi.AndroidCIBuild{}) {
		bootloaderBuild = &c.opts.BootloaderBuild
	}
	if c.opts.SystemImgBuild != (hoapi.AndroidCIBuild{}) {
		systemImageBuild = &c.opts.SystemImgBuild
	}
	fetchReq := &hoapi.FetchArtifactsRequest{
		AndroidCIBundle: &hoapi.AndroidCIBundle{Build: mainBuild, Type: hoapi.MainBundleType},
	}
	c.statePrinter.Print(stateMsgFetchMainBundle)
	err := c.service.FetchArtifacts(c.opts.Host, fetchReq)
	c.statePrinter.PrintDone(stateMsgFetchMainBundle, err)
	if err != nil {
		return nil, err
	}
	createReq := &hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuildSource: &hoapi.AndroidCIBuildSource{
					MainBuild:        mainBuild,
					KernelBuild:      kernelBuild,
					BootloaderBuild:  bootloaderBuild,
					SystemImageBuild: systemImageBuild,
				},
			},
		},
		AdditionalInstancesNum: c.opts.AdditionalInstancesNum(),
	}
	c.statePrinter.Print(stateMsgStartCVD)
	res, err := c.service.CreateCVD(c.opts.Host, createReq)
	c.statePrinter.PrintDone(stateMsgStartCVD, err)
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

type cvdListResult struct {
	Result []*CVDInfo
	Error  error
}

func listAllCVDs(service client.Service, controlDir string) ([]*CVDInfo, error) {
	hl, err := service.ListHosts()
	if err != nil {
		return nil, fmt.Errorf("Error listing hosts: %w", err)
	}
	var hosts []string
	for _, host := range hl.Items {
		hosts = append(hosts, host.Name)
	}
	var chans []chan cvdListResult
	statuses, merr := listCVDConnections(controlDir)
	for _, host := range hosts {
		ch := make(chan cvdListResult)
		chans = append(chans, ch)
		go func(name string, ch chan<- cvdListResult) {
			cvds, err := listHostCVDsInner(service, name, statuses)
			ch <- cvdListResult{Result: cvds, Error: err}
		}(host, ch)
	}
	var cvds []*CVDInfo
	for i, ch := range chans {
		host := hosts[i]
		result := <-ch
		if result.Error != nil {
			merr = multierror.Append(merr, fmt.Errorf("lists cvds for host %q failed: %w", host, err))
		}
		cvds = append(cvds, result.Result...)
	}
	return cvds, merr
}

func listHostCVDs(service client.Service, controlDir, host string) ([]*CVDInfo, error) {
	statuses, merr := listCVDConnectionsByHost(controlDir, host)
	result, err := listHostCVDsInner(service, host, statuses)
	if err != nil {
		merr = multierror.Append(merr, err)
	}
	return result, merr
}

// Calling listCVDConnectionsByHost is inefficient, this internal function avoids that for listAllCVDs.
func listHostCVDsInner(service client.Service, host string, statuses map[CVD]ConnStatus) ([]*CVDInfo, error) {
	cvds, err := service.ListCVDs(host)
	if err != nil {
		return nil, err
	}
	ret := make([]*CVDInfo, len(cvds))
	for i, c := range cvds {
		ret[i] = NewCVDInfo(service.RootURI(), host, c)
		if status, ok := statuses[ret[i].CVD]; ok {
			ret[i].ConnStatus = &status
		}
	}
	return ret, nil
}

const RequiredImagesFilename = "device/google/cuttlefish/required_images"

type MissingEnvVarErr string

func (s MissingEnvVarErr) Error() string {
	return fmt.Sprintf("Missing environment variable: %q", string(s))
}

const (
	CVDHostPackageDirName = "cvd-host_package"
	CVDHostPackageName    = "cvd-host_package.tar.gz"
)

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
	return result, nil
}

func verifyCVDHostPackageTar(dir string) error {
	tarInfo, err := os.Stat(filepath.Join(dir, CVDHostPackageName))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%q not found. Please run `m hosttar`.", CVDHostPackageName)
	}
	dirInfo, err := os.Stat(filepath.Join(dir, CVDHostPackageDirName))
	if err != nil {
		return fmt.Errorf("Failed getting cvd host package directory info: %w", err)
	}
	if tarInfo.ModTime().Before(dirInfo.ModTime()) {
		return fmt.Errorf("%q out of date. Please run `m hosttar`.", CVDHostPackageName)
	}
	return nil
}
