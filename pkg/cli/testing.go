package cli

import (
	"io"
	"net/url"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	hoapi "github.com/google/android-cuttlefish/frontend/src/host_orchestrator/api/v1"
	hoclient "github.com/google/android-cuttlefish/frontend/src/libhoclient"
	wclient "github.com/google/android-cuttlefish/frontend/src/libhoclient/webrtcclient"
	"github.com/gorilla/websocket"
)

const unitTestServiceURL = "test://unit"

type fakeClient struct{}

func (fakeClient) CreateHost(req *apiv1.CreateHostRequest) (*apiv1.HostInstance, error) {
	return &apiv1.HostInstance{Name: "foo"}, nil
}

func (fakeClient) ListHosts() (*apiv1.ListHostsResponse, error) {
	return &apiv1.ListHostsResponse{
		Items: []*apiv1.HostInstance{{Name: "foo"}, {Name: "bar"}},
	}, nil
}

func (fakeClient) DeleteHosts(name []string) error {
	return nil
}

func (fakeClient) RootURI() string {
	return unitTestServiceURL + "/v1"
}

func (fakeClient) HostService(host string) hoclient.HostOrchestratorService {
	if host == "" {
		panic("empty host")
	}
	return &fakeHostService{}
}

func (fakeClient) HostServiceURL(host string) (*url.URL, error) {
	return nil, nil
}

type fakeHostService struct{}

func (fakeHostService) GetInfraConfig() (*apiv1.InfraConfig, error) {
	return nil, nil
}

func (fakeHostService) ConnectWebRTC(device string, observer wclient.Observer, logger io.Writer, opts hoclient.ConnectWebRTCOpts) (*wclient.Connection, error) {
	return nil, nil
}

func (fakeHostService) ConnectADBWebSocket(device string) (*websocket.Conn, error) {
	return nil, nil
}

func (fakeHostService) FetchArtifacts(req *hoapi.FetchArtifactsRequest, creds hoclient.BuildAPICreds) (*hoapi.FetchArtifactsResponse, error) {
	return &hoapi.FetchArtifactsResponse{AndroidCIBundle: &hoapi.AndroidCIBundle{}}, nil
}

func (fakeHostService) CreateCVD(req *hoapi.CreateCVDRequest, creds hoclient.BuildAPICreds) (*hoapi.CreateCVDResponse, error) {
	return &hoapi.CreateCVDResponse{CVDs: []*hoapi.CVD{{Name: "cvd-1"}}}, nil
}

func (fakeHostService) CreateCVDOp(req *hoapi.CreateCVDRequest, creds hoclient.BuildAPICreds) (*hoapi.Operation, error) {
	return nil, nil
}

func (fakeHostService) DeleteCVD(id string) error {
	return nil
}

func (fakeHostService) ListCVDs() ([]*hoapi.CVD, error) {
	return []*hoapi.CVD{{Name: "cvd-1"}}, nil
}

func (fakeHostService) CreateUploadDir() (string, error) {
	return "", nil
}

func (fakeHostService) UploadFile(uploadDir string, name string) error {
	return nil
}

func (fakeHostService) UploadFileWithOptions(uploadDir string, name string, options hoclient.UploadOptions) error {
	return nil
}

func (fakeHostService) ExtractFile(string, string) (*hoapi.Operation, error) { return nil, nil }

func (fakeHostService) DownloadRuntimeArtifacts(dst io.Writer) error {
	return nil
}

func (fakeHostService) WaitForOperation(string, any) error { return nil }

func (fakeHostService) CreateBugreport(string, io.Writer) error { return nil }

func (fakeHostService) Powerwash(groupName, instanceName string) error { return nil }

func (fakeHostService) Stop(groupName, instanceName string) error { return nil }

func (fakeHostService) Start(groupName, instanceName string, req *hoapi.StartCVDRequest) error {
	return nil
}

func (fakeHostService) CreateSnapshot(groupName, instanceName string) (*hoapi.CreateSnapshotResponse, error) {
	return nil, nil
}
