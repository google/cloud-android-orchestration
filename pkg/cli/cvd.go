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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/cloud-android-orchestration/pkg/cli/authz"
	"github.com/google/cloud-android-orchestration/pkg/client"

	hoapi "github.com/google/android-cuttlefish/frontend/src/host_orchestrator/api/v1"
	hoclient "github.com/google/android-cuttlefish/frontend/src/libhoclient"
	"github.com/hashicorp/go-multierror"
)

type RemoteCVDLocator struct {
	ServiceRootEndpoint string `json:"service_root_endpoint"`
	Host                string `json:"host"`
	// Identifier within the whole fleet.
	ID string `json:"id"`
	// Identifier within a group.
	Name string `json:"name"`
	// Instead of `Name`, `WebRTCDeviceID` is the identifier used for setting up the adb connections. It
	// contains the group name and the device name, eg: "cvd-1_1".
	WebRTCDeviceID string `json:"webrtc_device_id"`
	// ADB port of Cuttlefish instance.
	ADBSerial string `json:"adb_serial"`
}

func (l *RemoteCVDLocator) Group() string {
	return strings.Split(l.ID, "/")[0]
}

type RemoteCVD struct {
	RemoteCVDLocator
	Status     string
	Displays   []string
	ConnStatus *ConnStatus
}

type RemoteHost struct {
	ServiceRootEndpoint string `json:"service_root_endpoint"`
	Name                string `json:"host"`
	CVDs                []*RemoteCVD
}

func NewRemoteCVD(url, host string, cvd *hoapi.CVD) *RemoteCVD {
	return &RemoteCVD{
		RemoteCVDLocator: RemoteCVDLocator{
			ServiceRootEndpoint: url,
			Host:                host,
			ID:                  cvd.ID(),
			Name:                cvd.Name,
			WebRTCDeviceID:      cvd.WebRTCDeviceID,
			ADBSerial:           cvd.ADBSerial,
		},
		Status:   cvd.Status,
		Displays: cvd.Displays,
	}
}

const (
	NoneCredentialsSource     = "none"
	InjectedCredentialsSource = "injected"
)

type CreateCVDLocalOpts struct {
	LocalBootloaderSrc string
	LocalCVDHostPkgSrc string
	LocalImagesSrcs    []string
	LocalImagesZipSrc  string
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
	// Structure: https://android.googlesource.com/device/google/cuttlefish/+/8bbd3b9cd815f756f332791d45c4f492b663e493/host/commands/cvd/parser/README.md
	// Example: https://cs.android.com/android/platform/superproject/main/+/main:device/google/cuttlefish/host/cvd_test_configs/main_phone-main_watch.json;drc=b2e8f4f014abb7f9cb56c0ae199334aacb04542d
	EnvConfig map[string]interface{}
	// If true, perform the ADB connection automatically.
	AutoConnect               bool
	BuildAPICredentialsSource string
	BuildAPIUserProjectID     string
	CreateCVDLocalOpts
}

func (o *CreateCVDOpts) AdditionalInstancesNum() uint32 {
	if o.NumInstances <= 0 {
		return 0
	}
	return uint32(o.NumInstances - 1)
}

func (o *CreateCVDOpts) Update(s *Service) {
	if s.BuildAPICredentialsSource != "" {
		o.BuildAPICredentialsSource = s.BuildAPICredentialsSource
	}
}

func createCVD(service client.Service, createOpts CreateCVDOpts, statePrinter *statePrinter) ([]*RemoteCVD, error) {
	creator, err := newCVDCreator(service, createOpts, statePrinter)
	if err != nil {
		return nil, fmt.Errorf("failed to create cvd: %w", err)
	}
	cvds, err := creator.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create cvd: %w", err)
	}
	result := []*RemoteCVD{}
	for _, cvd := range cvds {
		result = append(result, NewRemoteCVD(service.RootURI(), createOpts.Host, cvd))
	}
	return result, nil
}

type CredentialsFactory func() hoclient.BuildAPICredential

type cvdCreator struct {
	service            client.Service
	opts               CreateCVDOpts
	statePrinter       *statePrinter
	credentialsFactory CredentialsFactory
}

func newCVDCreator(service client.Service, opts CreateCVDOpts, statePrinter *statePrinter) (*cvdCreator, error) {
	cf, err := credentialsFactoryFromSource(opts.BuildAPICredentialsSource, opts.BuildAPIUserProjectID)
	if err != nil {
		return nil, err
	}
	return &cvdCreator{
		service:            service,
		opts:               opts,
		statePrinter:       statePrinter,
		credentialsFactory: cf,
	}, nil
}

func (c *cvdCreator) Create() ([]*hoapi.CVD, error) {
	if c.opts.LocalImage {
		return c.createCVDFromLocalBuild()
	}
	if !c.opts.CreateCVDLocalOpts.empty() {
		return c.createCVDFromLocalSrcs()
	}
	return c.createCVDFromAndroidCI()
}

const uaEnvConfigTmplStr = `
{
  "common": {
    "host_package": "@user_artifacts/{{.ArtifactsDir}}"
  },
  "instances": [
    {
      "vm": {
        "memory_mb": 8192,
        "setupwizard_mode": "OPTIONAL",
        "cpus": 8
      },
      "disk": {
        "default_build": "@user_artifacts/{{.ArtifactsDir}}"
      },
      "streaming": {
        "device_id": "cvd-1"
      }
    }
  ]
}
`

var uaEnvConfigTmpl *template.Template

func init() {
	var err error
	if uaEnvConfigTmpl, err = template.New("").Parse(uaEnvConfigTmplStr); err != nil {
		panic(err)
	}
}

type uaEnvConfigTmplData struct {
	ArtifactsDir string
}

func buildUAEnvConfig(data uaEnvConfigTmplData) (map[string]interface{}, error) {
	var b bytes.Buffer
	if err := uaEnvConfigTmpl.Execute(&b, uaEnvConfigTmplData{ArtifactsDir: data.ArtifactsDir}); err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	if err := json.Unmarshal(b.Bytes(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *cvdCreator) createCVDFromLocalBuild() ([]*hoapi.CVD, error) {
	buildTop, err := envVar(AndroidBuildTopVarName)
	if err != nil {
		return nil, err
	}
	productOut, err := envVar(AndroidProductOutVarName)
	if err != nil {
		return nil, err
	}
	names, err := ListLocalImageRequiredFiles(buildTop, productOut)
	if err != nil {
		return nil, err
	}
	targetArch, err := getTargetArch(buildTop)
	if err != nil {
		return nil, err
	}
	hostOutRelativePath, err := GetHostOutRelativePath(targetArch)
	if err != nil {
		return nil, err
	}
	hostOut := filepath.Join(buildTop, hostOutRelativePath)
	if err := verifyCVDHostPackageTar(hostOut); err != nil {
		return nil, err
	}
	names = append(names, filepath.Join(hostOut, CVDHostPackageName))
	hostSrv := c.service.HostService(c.opts.Host)
	uploadDir, err := hostSrv.CreateUploadDir()
	if err != nil {
		return nil, err
	}
	envConfig, err := buildUAEnvConfig(uaEnvConfigTmplData{ArtifactsDir: uploadDir})
	if err != nil {
		return nil, err
	}
	if err := uploadFiles(hostSrv, uploadDir, names, c.statePrinter); err != nil {
		return nil, err
	}
	req := &hoapi.CreateCVDRequest{EnvConfig: envConfig}
	res, err := hostSrv.CreateCVD(req, c.credentialsFactory())
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

const (
	stateMsgFetchMainBundle = "Fetching main bundle artifacts"
	stateMsgStartCVD        = "Starting and waiting for boot complete"
	stateMsgFetchAndStart   = "Fetching, starting and waiting for boot complete"
)

func (c *cvdCreator) createCVDFromAndroidCI() ([]*hoapi.CVD, error) {
	if c.opts.EnvConfig != nil {
		return c.createWithCanonicalConfig()
	}
	return c.createWithOpts()
}

func (c *cvdCreator) createWithCanonicalConfig() ([]*hoapi.CVD, error) {
	createReq := &hoapi.CreateCVDRequest{
		EnvConfig: c.opts.EnvConfig,
	}
	c.statePrinter.Print(stateMsgFetchAndStart)
	res, err := c.service.HostService(c.opts.Host).CreateCVD(createReq, c.credentialsFactory())
	c.statePrinter.PrintDone(stateMsgFetchAndStart, err)
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

func (c *cvdCreator) createWithOpts() ([]*hoapi.CVD, error) {
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
	fetchMainBuildRes, err := c.service.HostService(c.opts.Host).FetchArtifacts(fetchReq, c.credentialsFactory())
	c.statePrinter.PrintDone(stateMsgFetchMainBundle, err)
	if err != nil {
		return nil, err
	}
	createReq := &hoapi.CreateCVDRequest{
		CVD: &hoapi.CVD{
			BuildSource: &hoapi.BuildSource{
				AndroidCIBuildSource: &hoapi.AndroidCIBuildSource{
					MainBuild:        fetchMainBuildRes.AndroidCIBundle.Build,
					KernelBuild:      kernelBuild,
					BootloaderBuild:  bootloaderBuild,
					SystemImageBuild: systemImageBuild,
				},
			},
		},
		AdditionalInstancesNum: c.opts.AdditionalInstancesNum(),
	}
	c.statePrinter.Print(stateMsgStartCVD)
	res, err := c.service.HostService(c.opts.Host).CreateCVD(createReq, c.credentialsFactory())
	c.statePrinter.PrintDone(stateMsgStartCVD, err)
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

func (c *cvdCreator) createCVDFromLocalSrcs() ([]*hoapi.CVD, error) {
	if err := c.opts.CreateCVDLocalOpts.validate(); err != nil {
		return nil, fmt.Errorf("invalid local source: %w", err)
	}
	uploadDir, err := c.service.HostService(c.opts.Host).CreateUploadDir()
	if err != nil {
		return nil, err
	}
	envConfig, err := buildUAEnvConfig(uaEnvConfigTmplData{ArtifactsDir: uploadDir})
	if err != nil {
		return nil, err
	}
	hostSrv := c.service.HostService(c.opts.Host)
	if err := uploadFiles(hostSrv, uploadDir, c.opts.CreateCVDLocalOpts.srcs(), c.statePrinter); err != nil {
		return nil, err
	}
	req := &hoapi.CreateCVDRequest{EnvConfig: envConfig}
	res, err := hostSrv.CreateCVD(req, c.credentialsFactory())
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

func credentialsFactoryFromSource(source string, projectID string) (CredentialsFactory, error) {
	switch source {
	case NoneCredentialsSource:
		return func() hoclient.BuildAPICredential { return hoclient.BuildAPICredential{} }, nil
	case InjectedCredentialsSource:
		if projectID != "" {
			return nil, fmt.Errorf("project ID is not supported with injected credentials")
		}
		return func() hoclient.BuildAPICredential {
			return hoclient.BuildAPICredential{AccessToken: client.InjectedCredentials}
		}, nil
	default:
		// expected: `(jwt|oauth):/dir/credentialFile`
		strs := strings.SplitN(source, ":", 2)
		if strs == nil || len(strs) != 2 {
			return nil, fmt.Errorf("unknown credential type, only accept: `none`/`injected`/`%s:'<filepath>'`/`%s:'<filepath>'`", jwtAuthType, oauthAuthType)
		}
		authType, filepath := strs[0], strs[1]
		token, err := accessToken(authType, filepath)
		if err != nil {
			return nil, fmt.Errorf("retrieve access token error: %w", err)
		}
		return func() hoclient.BuildAPICredential {
			return hoclient.BuildAPICredential{AccessToken: token, UserProjectID: projectID}
		}, nil
	}
}

const (
	jwtAuthType   = "jwt"
	oauthAuthType = "oauth"
)

func accessToken(authType string, filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("cannot read content from credential filepath %s: %w", filepath, err)
	}
	switch authType {
	case jwtAuthType:
		tk, err := authz.JWTAccessToken(content)
		if err != nil {
			return "", err
		}
		return tk.AccessToken, nil
	case oauthAuthType:
		tk, err := authz.OAuthAccessToken(content)
		if err != nil {
			return "", err
		}
		return tk.AccessToken, nil
	default:
		return "", fmt.Errorf("unknown authType, get '%s' (expected: '%s' or '%s')", authType, jwtAuthType, oauthAuthType)
	}
}

type cvdListResult struct {
	Result []*RemoteCVD
	Error  error
}

func listCVDs(service client.Service, controlDir string) ([]*RemoteHost, error) {
	hl, err := service.ListHosts()
	if err != nil {
		return nil, fmt.Errorf("error listing hosts: %w", err)
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
	var result []*RemoteHost
	for i, ch := range chans {
		hostName := hosts[i]
		listResult := <-ch
		if listResult.Error != nil {
			merr = multierror.Append(merr, fmt.Errorf("lists cvds for host %q failed: %w", hostName, err))
		}
		host := &RemoteHost{
			ServiceRootEndpoint: service.RootURI(),
			Name:                hostName,
			CVDs:                listResult.Result,
		}
		result = append(result, host)
	}
	return result, merr
}

func listCVDsSingleHost(service client.Service, controlDir, host string) ([]*RemoteHost, error) {
	statuses, merr := listCVDConnectionsByHost(controlDir, host)
	cvds, err := listHostCVDsInner(service, host, statuses)
	if err != nil {
		merr = multierror.Append(merr, err)
	}
	result := []*RemoteHost{
		{
			ServiceRootEndpoint: service.RootURI(),
			Name:                host,
			CVDs:                cvds,
		},
	}
	return result, merr
}

func flattenCVDs(hosts []*RemoteHost) []*RemoteCVD {
	result := []*RemoteCVD{}
	for _, h := range hosts {
		result = append(result, h.CVDs...)
	}
	return result
}

// Calling listCVDConnectionsByHost is inefficient, this internal function avoids that for listAllCVDs.
func listHostCVDsInner(service client.Service, host string, statuses map[RemoteCVDLocator]ConnStatus) ([]*RemoteCVD, error) {
	cvds, err := service.HostService(host).ListCVDs()
	if err != nil {
		return nil, err
	}
	ret := make([]*RemoteCVD, len(cvds))
	for i, c := range cvds {
		ret[i] = NewRemoteCVD(service.RootURI(), host, c)
		if status, ok := statuses[ret[i].RemoteCVDLocator]; ok {
			ret[i].ConnStatus = &status
		}
	}
	return ret, nil
}

func findCVD(service client.Service, controlDir, host, device string) (*RemoteCVD, error) {
	cvdHosts, err := listCVDsSingleHost(service, controlDir, host)
	if err != nil {
		return nil, fmt.Errorf("error listing CVDs: %w", err)
	}
	for _, cvd := range cvdHosts[0].CVDs {
		if device == cvd.WebRTCDeviceID {
			return cvd, nil
		}
	}
	return nil, fmt.Errorf("failed to find CVD for %s in %s", device, host)
}

const RequiredImagesFilename = "device/google/cuttlefish/required_images"

const (
	CVDHostPackageDirName = "cvd-host_package"
	CVDHostPackageName    = "cvd-host_package.tar.gz"
)

const (
	AndroidBuildTopVarName   = "ANDROID_BUILD_TOP"
	AndroidProductOutVarName = "ANDROID_PRODUCT_OUT"
)

// List the required filenames to create a cuttlefish instance given the `build top` and `product out`
// values from an environment where Android was built.
func ListLocalImageRequiredFiles(buildTop, productOut string) ([]string, error) {
	reqImgsFilename := filepath.Join(buildTop, RequiredImagesFilename)
	f, err := os.Open(reqImgsFilename)
	if err != nil {
		return nil, fmt.Errorf("error opening the required images list file: %w", err)
	}
	defer f.Close()
	content, err := os.ReadFile(reqImgsFilename)
	if err != nil {
		return nil, fmt.Errorf("error reading the required images list file: %w", err)
	}
	contentStr := strings.TrimRight(string(content), "\n")
	lines := strings.Split(contentStr, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, filepath.Join(productOut, line))
	}
	return result, nil
}

func getTargetArch(buildTop string) (string, error) {
	// `$ANDROID_BUILD_TOP/out/soong_ui` can bring values of build variables, set by `lunch` command.
	// https://cs.android.com/android/platform/superproject/main/+/main:build/soong/cmd/soong_ui/main.go;l=298
	bin := filepath.Join(buildTop, "out/soong_ui")
	cmdOut, err := exec.Command(bin, "--dumpvar-mode", "TARGET_ARCH").Output()
	if err != nil {
		return "", fmt.Errorf("error while getting target arch: %w", err)
	}
	return strings.TrimSpace(string(cmdOut)), nil
}

func GetHostOutRelativePath(targetArch string) (string, error) {
	m := map[string]string{
		"x86_64": "out/host/linux-x86",
		"arm64":  "out/host/linux_musl-arm64",
	}
	relativePath, ok := m[targetArch]
	if !ok {
		return "", fmt.Errorf("Unexpected target architecture: %q", targetArch)
	}
	return relativePath, nil
}

func verifyCVDHostPackageTar(dir string) error {
	tarInfo, err := os.Stat(filepath.Join(dir, CVDHostPackageName))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%q not found. Please run `m hosttar`", CVDHostPackageName)
	}
	dirInfo, err := os.Stat(filepath.Join(dir, CVDHostPackageDirName))
	if err != nil {
		return fmt.Errorf("failed getting cvd host package directory info: %w", err)
	}
	if tarInfo.ModTime().Before(dirInfo.ModTime()) {
		return fmt.Errorf("%q out of date. Please run `m hosttar`", CVDHostPackageName)
	}
	return nil
}

func (o *CreateCVDLocalOpts) validate() error {
	if o.LocalBootloaderSrc == "" && o.LocalImagesZipSrc == "" {
		return errors.New("missing bootloader source")
	}
	if o.LocalCVDHostPkgSrc == "" {
		return errors.New("missing cvd host package source")
	}
	return nil
}

func (o *CreateCVDLocalOpts) srcs() []string {
	result := []string{}
	if o.LocalBootloaderSrc != "" {
		result = append(result, o.LocalBootloaderSrc)
	}
	if o.LocalCVDHostPkgSrc != "" {
		result = append(result, o.LocalCVDHostPkgSrc)
	}
	if o.LocalImagesZipSrc != "" {
		result = append(result, o.LocalImagesZipSrc)
	}
	for _, v := range o.LocalImagesSrcs {
		result = append(result, v)
	}
	return result
}

func (o *CreateCVDLocalOpts) empty() bool {
	return o.LocalBootloaderSrc == "" && o.LocalCVDHostPkgSrc == "" &&
		len(o.LocalImagesSrcs) == 0
}

type MissingEnvVarErr string

func (s MissingEnvVarErr) Error() string {
	return fmt.Sprintf("Missing environment variable: %q", string(s))
}

func envVar(name string) (string, error) {
	if _, ok := os.LookupEnv(name); !ok {
		return "", MissingEnvVarErr(name)
	}
	return os.Getenv(name), nil
}

func uploadFiles(srv hoclient.HostOrchestratorService, uploadDir string, names []string, statePrinter *statePrinter) error {
	extractOps := []string{}
	for _, name := range names {
		state := fmt.Sprintf("Uploading %q", filepath.Base(name))
		statePrinter.Print(state)
		err := srv.UploadFile(uploadDir, name)
		statePrinter.PrintDone(state, err)
		if err != nil {
			return err
		}
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
			op, err := srv.ExtractFile(uploadDir, filepath.Base(name))
			if err != nil {
				return fmt.Errorf("failed uploading files: %w", err)
			}
			extractOps = append(extractOps, op.Name)
		}
	}
	for _, name := range extractOps {
		if err := srv.WaitForOperation(name, nil); err != nil {
			return fmt.Errorf("failed uploading files: %w", err)
		}
	}
	return nil
}
