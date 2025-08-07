package cli

import (
	"net/url"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	hoclient "github.com/google/android-cuttlefish/frontend/src/libhoclient"
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

func (fakeClient) HostClient(host string) hoclient.HostOrchestratorClient {
	if host == "" {
		panic("empty host")
	}
	return hoclient.NewFakeHostOrchestratorClient()
}

func (fakeClient) HostServiceURL(host string) (*url.URL, error) {
	return nil, nil
}
