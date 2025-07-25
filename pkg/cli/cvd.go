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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/cloud-android-orchestration/pkg/cli/authz"
	"github.com/google/cloud-android-orchestration/pkg/client"

	lcpb "github.com/google/android-cuttlefish/base/cvd/cuttlefish/host/commands/cvd/cli/parser/golang"
	hoapi "github.com/google/android-cuttlefish/frontend/src/host_orchestrator/api/v1"
	hoclient "github.com/google/android-cuttlefish/frontend/src/libhoclient"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/encoding/protojson"
)

type RemoteCVDLocator struct {
	Host string `json:"host"`
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
	ServiceURL *url.URL
	Name       string
	CVDs       []*RemoteCVD
}

func NewRemoteCVD(host string, cvd *hoapi.CVD) *RemoteCVD {
	return &RemoteCVD{
		RemoteCVDLocator: RemoteCVDLocator{
			Host:           host,
			ID:             cvd.ID(),
			Name:           cvd.Name,
			WebRTCDeviceID: cvd.WebRTCDeviceID,
			ADBSerial:      cvd.ADBSerial,
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
	// TODO(b/378123925): Work with https://github.com/google/android-cuttlefish/blob/main/base/cvd/cuttlefish/host/commands/cvd/cli/parser/load_config.proto
	EnvConfig map[string]interface{}
	// If true, perform the ADB connection automatically.
	AutoConnect               bool
	ConnectAgent              string
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
	o.BuildAPICredentialsSource = s.BuildAPICredentialsSource
	o.ConnectAgent = s.ConnectAgent
}

func createCVD(srvClient client.Client, createOpts CreateCVDOpts, statePrinter *statePrinter) ([]*RemoteCVD, error) {
	creator, err := newCVDCreator(srvClient, createOpts, statePrinter)
	if err != nil {
		return nil, fmt.Errorf("failed to create cvd: %w", err)
	}
	cvds, err := creator.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create cvd: %w", err)
	}
	result := []*RemoteCVD{}
	for _, cvd := range cvds {
		result = append(result, NewRemoteCVD(createOpts.Host, cvd))
	}
	return result, nil
}

type CredentialsFactory func() hoclient.BuildAPICreds

type cvdCreator struct {
	client             client.Client
	opts               CreateCVDOpts
	statePrinter       *statePrinter
	credentialsFactory CredentialsFactory
}

func newCVDCreator(srvClient client.Client, opts CreateCVDOpts, statePrinter *statePrinter) (*cvdCreator, error) {
	cf, err := credentialsFactoryFromSource(opts.BuildAPICredentialsSource, opts.BuildAPIUserProjectID)
	if err != nil {
		return nil, err
	}
	return &cvdCreator{
		client:             srvClient,
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
    "host_package": "{{.HostPkg}}"
  },
  "instances": [
    {
      "vm": {
        "memory_mb": 8192,
        "setupwizard_mode": "OPTIONAL",
        "cpus": 8
      },
      "disk": {
        "default_build": "{{.Artifacts}}"
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
	Artifacts string
	HostPkg   string
}

func buildUAEnvConfig(artifacts []string, hostPkg string) (map[string]interface{}, error) {
	var b bytes.Buffer
	if err := uaEnvConfigTmpl.Execute(&b, uaEnvConfigTmplData{Artifacts: strings.Join(artifacts, ","), HostPkg: hostPkg}); err != nil {
		return nil, fmt.Errorf("failed to fulfill template: %w", err)
	}

	es := lcpb.EnvironmentSpecification{}
	if err := protojson.Unmarshal(b.Bytes(), &es); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(b.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %w", err)
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
	artifacts, err := ListLocalImageRequiredFiles(buildTop, productOut)
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
	envConfig, err := buildUAEnvConfig(artifacts, filepath.Join(hostOut, CVDHostPackageName))
	if err != nil {
		return nil, err
	}
	c.opts.EnvConfig = envConfig
	return c.createWithCanonicalConfig()
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
	hostSrv := c.client.HostClient(c.opts.Host)
	envConfig := make(map[string]any)
	if err := deepCopy(c.opts.EnvConfig, &envConfig); err != nil {
		return nil, fmt.Errorf("deep copying env config failed: %w", err)
	}
	if err := c.uploadFilesAndUpdateEnvConfig(hostSrv, envConfig); err != nil {
		return nil, fmt.Errorf("failed uploading files from environment config: %w", err)
	}

	es := lcpb.EnvironmentSpecification{}
	b, err := json.Marshal(envConfig)
	if err != nil {
		return nil, fmt.Errorf("config is not convertible to JSON")
	}
	if err := protojson.Unmarshal(b, &es); err != nil {
		return nil, fmt.Errorf("config is not convertible to load_config")
	}

	createReq := &hoapi.CreateCVDRequest{
		EnvConfig: envConfig,
	}
	c.statePrinter.Print(stateMsgFetchAndStart)
	res, err := hostSrv.CreateCVD(createReq, c.credentialsFactory())
	c.statePrinter.PrintDone(stateMsgFetchAndStart, err)
	if err != nil {
		return nil, err
	}
	return res.CVDs, nil
}

func (c *cvdCreator) uploadFilesAndUpdateEnvConfig(client hoclient.HostOrchestratorClient, config map[string]interface{}) error {
	if err := c.uploadCVDHostPackageAndUpdateEnvConfig(client, config); err != nil {
		return err
	}
	return c.uploadImagesAndUpdateEnvConfig(client, config)
}

// TODO(b/378123925) Work with https://github.com/google/android-cuttlefish/blob/main/base/cvd/cuttlefish/host/commands/cvd/cli/parser/load_config.proto
func (c *cvdCreator) uploadCVDHostPackageAndUpdateEnvConfig(client hoclient.HostOrchestratorClient, config map[string]interface{}) error {
	common, ok := config["common"]
	if !ok {
		return nil
	}
	commonMap, ok := common.(map[string]any)
	if !ok {
		return nil
	}
	hostPackage, ok := commonMap["host_package"]
	if !ok {
		return nil
	}
	if val, ok := hostPackage.(string); ok && !strings.HasPrefix(val, "@ab") {
		isDir, err := isDirectory(val)
		if err != nil {
			return fmt.Errorf("directory test for %q failed: %w", val, err)
		}
		if isDir {
			return fmt.Errorf("uploading directory not supported")
		}
		imageDirID, err := uploadFilesAndCreateImageDir(client, []string{val}, c.statePrinter)
		if err != nil {
			return fmt.Errorf("failed uploading %q: %w", val, err)
		}
		commonMap["host_package"] = "@image_dirs/" + imageDirID
	}
	return nil
}

// TODO(b/378123925) Work with https://github.com/google/android-cuttlefish/blob/main/base/cvd/cuttlefish/host/commands/cvd/cli/parser/load_config.proto
func (c *cvdCreator) uploadImagesAndUpdateEnvConfig(client hoclient.HostOrchestratorClient, config map[string]interface{}) error {
	instances, ok := config["instances"]
	if !ok {
		return nil
	}
	instancesArr, ok := instances.([]any)
	if !ok {
		return nil
	}
	for _, ins := range instancesArr {
		ins, ok := ins.(map[string]any)
		if !ok {
			continue
		}
		disk, ok := ins["disk"]
		if !ok {
			continue
		}
		diskMap, ok := disk.(map[string]any)
		if !ok {
			continue
		}
		defaultBuild, ok := diskMap["default_build"]
		if !ok {
			continue
		}
		if val, ok := defaultBuild.(string); ok && !strings.HasPrefix(val, "@ab") {
			images := strings.Split(val, ",")
			for _, image := range images {
				if isDir, err := isDirectory(image); err != nil {
					return fmt.Errorf("directory test for %q failed: %w", image, err)
				} else if isDir {
					return fmt.Errorf("uploading directory not supported")
				}
			}
			imageDirID, err := uploadFilesAndCreateImageDir(client, images, c.statePrinter)
			if err != nil {
				return fmt.Errorf("failed uploading %q: %w", images, err)
			}
			diskMap["default_build"] = "@image_dirs/" + imageDirID
		}
	}
	return nil
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
	fetchMainBuildRes, err := c.client.HostClient(c.opts.Host).FetchArtifacts(fetchReq, c.credentialsFactory())
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
	res, err := c.client.HostClient(c.opts.Host).CreateCVD(createReq, c.credentialsFactory())
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
	envConfig, err := buildUAEnvConfig(c.opts.CreateCVDLocalOpts.artifacts(), c.opts.CreateCVDLocalOpts.LocalCVDHostPkgSrc)
	if err != nil {
		return nil, err
	}
	c.opts.EnvConfig = envConfig
	return c.createWithCanonicalConfig()
}

type coInjectBuildAPICreds struct{}

func (c *coInjectBuildAPICreds) ApplyToHTTPRequest(rb *hoclient.HTTPRequestBuilder) {
	rb.AddHeader("X-Cutf-Cloud-Orchestrator-Inject-BuildAPI-Creds" /* avoid empty header value */, "inject")
}

func credentialsFactoryFromSource(source string, projectID string) (CredentialsFactory, error) {
	switch source {
	case NoneCredentialsSource:
		return func() hoclient.BuildAPICreds { return &hoclient.AccessTokenBuildAPICreds{} }, nil
	case InjectedCredentialsSource:
		if projectID != "" {
			return nil, fmt.Errorf("project ID is not supported with injected credentials")
		}
		return func() hoclient.BuildAPICreds { return &coInjectBuildAPICreds{} }, nil
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
		return func() hoclient.BuildAPICreds {
			return &hoclient.AccessTokenBuildAPICreds{AccessToken: token, UserProjectID: projectID}
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

func listCVDs(srvClient client.Client, controlDir string) ([]*RemoteHost, error) {
	hl, err := srvClient.ListHosts()
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
			cvds, err := listHostCVDsInner(srvClient, name, statuses)
			ch <- cvdListResult{Result: cvds, Error: err}
		}(host, ch)
	}
	var result []*RemoteHost
	for i, ch := range chans {
		hostName := hosts[i]
		listResult := <-ch
		if listResult.Error != nil {
			merr = multierror.Append(merr, fmt.Errorf("lists cvds for host %q failed: %w", hostName, err))
			continue
		}
		srvURL, err := srvClient.HostServiceURL(hostName)
		if err != nil {
			merr = multierror.Append(merr, fmt.Errorf("failed getting host service url: %w", err))
			continue
		}
		host := &RemoteHost{
			ServiceURL: srvURL,
			Name:       hostName,
			CVDs:       listResult.Result,
		}
		result = append(result, host)
	}
	return result, merr
}

func listCVDsSingleHost(srvClient client.Client, controlDir, host string) ([]*RemoteHost, error) {
	statuses, merr := listCVDConnectionsByHost(controlDir, host)
	cvds, err := listHostCVDsInner(srvClient, host, statuses)
	if err != nil {
		merr = multierror.Append(merr, err)
	}
	result := []*RemoteHost{{Name: host, CVDs: cvds}}
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
func listHostCVDsInner(srvClient client.Client, host string, statuses map[RemoteCVDLocator]ConnStatus) ([]*RemoteCVD, error) {
	cvds, err := srvClient.HostClient(host).ListCVDs()
	if err != nil {
		return nil, err
	}
	ret := make([]*RemoteCVD, len(cvds))
	for i, c := range cvds {
		ret[i] = NewRemoteCVD(host, c)
		if status, ok := statuses[ret[i].RemoteCVDLocator]; ok {
			ret[i].ConnStatus = &status
		}
	}
	return ret, nil
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
		return "", fmt.Errorf("unexpected target architecture: %q", targetArch)
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

func (o *CreateCVDLocalOpts) artifacts() []string {
	result := []string{}
	if o.LocalBootloaderSrc != "" {
		result = append(result, o.LocalBootloaderSrc)
	}
	if o.LocalImagesZipSrc != "" {
		result = append(result, o.LocalImagesZipSrc)
	}
	return append(result, o.LocalImagesSrcs...)
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

func uploadFilesAndCreateImageDir(client hoclient.HostOrchestratorClient, filenames []string, statePrinter *statePrinter) (string, error) {
	op, err := client.CreateImageDirectory()
	if err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}
	res := &hoapi.CreateImageDirectoryResponse{}
	if err := client.WaitForOperation(op.Name, &res); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}
	imageDirID := res.ID
	extractOps := make(map[string]string)
	updateImageDirOpNames := []string{}
	for _, filename := range filenames {
		state := fmt.Sprintf("Uploading %q", filepath.Base(filename))
		statePrinter.Print(state)
		err := client.UploadArtifact(filename)
		statePrinter.PrintDone(state, err)
		if err != nil {
			return "", fmt.Errorf("failed to upload artifact: %w", err)
		}
		if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".zip") {
			op, err := client.ExtractArtifact(filename)
			if err != nil {
				return "", fmt.Errorf("failed to extract artifact: %w", err)
			}
			extractOps[op.Name] = filename
		} else {
			op, err := client.UpdateImageDirectoryWithUserArtifact(imageDirID, filename)
			if err != nil {
				return "", fmt.Errorf("failed to update image directory: %w", err)
			}
			updateImageDirOpNames = append(updateImageDirOpNames, op.Name)
		}
	}
	for extractOpName, filename := range extractOps {
		state := fmt.Sprintf("Extracting %q", filepath.Base(filename))
		statePrinter.Print(state)
		err := client.WaitForOperation(extractOpName, nil)
		if err != nil {
			if apiErr, ok := err.(*hoclient.ApiCallError); !ok || apiErr.HTTPStatusCode != http.StatusConflict {
				statePrinter.PrintDone(state, err)
				return "", fmt.Errorf("failed to wait for extracting operation: %w", err)
			}
		}
		statePrinter.PrintDone(state, nil)
		updateImageDirOp, err := client.UpdateImageDirectoryWithUserArtifact(imageDirID, filename)
		if err != nil {
			return "", fmt.Errorf("failed to update image directory: %w", err)
		}
		updateImageDirOpNames = append(updateImageDirOpNames, updateImageDirOp.Name)
	}
	state := "Preparing image directory"
	statePrinter.Print(state)
	for _, opName := range updateImageDirOpNames {
		if err := client.WaitForOperation(opName, nil); err != nil {
			statePrinter.PrintDone(state, err)
			return "", fmt.Errorf("failed to update image directory: %w", err)
		}
	}
	statePrinter.PrintDone(state, nil)
	return imageDirID, nil
}

// Deep copies src to dst using json marshaling.
func deepCopy(src, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}
