// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package net

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app"
)

type HostClient struct {
	url    *url.URL
	client *http.Client
}

func NewHostClient(url *url.URL, allowSelfSigned bool) *HostClient {
	ret := &HostClient{
		url:    url,
		client: http.DefaultClient,
	}
	if allowSelfSigned {
		// This creates a copy of the default transport and casts it to the right
		// structure. The cast is safe because the package documentation explicitly
		// says the variable is of the http.Transport type.
		transport := *http.DefaultTransport.(*http.Transport)
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		ret.client = &http.Client{Transport: &transport}
	}
	return ret
}

func (c *HostClient) Get(path, query string, out *app.HostResponse) (int, error) {
	url := *c.url // Shallow copy
	url.Path = path
	url.RawQuery = query
	res, err := c.client.Get(url.String())
	if err != nil {
		return -1, fmt.Errorf("Failed to connect to device host: %w", err)
	}
	defer res.Body.Close()
	if out != nil {
		err = parseReply(res, out.Result, out.Error)
	}
	return res.StatusCode, err
}

func (c *HostClient) Post(path, query string, bodyJSON any, out *app.HostResponse) (int, error) {
	bodyStr, err := json.Marshal(bodyJSON)
	if err != nil {
		return -1, fmt.Errorf("Failed to parse JSON request: %w", err)
	}
	url := *c.url // Shallow copy
	url.Path = path
	url.RawQuery = query
	res, err := c.client.Post(url.String(), "application/json", bytes.NewBuffer(bodyStr))
	if err != nil {
		return -1, fmt.Errorf("Failed to connecto to device host: %w", err)
	}
	defer res.Body.Close()
	if out != nil {
		err = parseReply(res, out.Result, out.Error)
	}
	return res.StatusCode, err
}

func (c *HostClient) GetReverseProxy() *httputil.ReverseProxy {
	devProxy := httputil.NewSingleHostReverseProxy(c.url)
	if c.client != http.DefaultClient {
		// Make sure the reverse proxy has the same customizations as the http client.
		devProxy.Transport = c.client.Transport
	}
	return devProxy
}

func parseReply(res *http.Response, resObj any, resErr *apiv1.Error) error {
	var err error
	dec := json.NewDecoder(res.Body)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = dec.Decode(resErr)
	} else {
		err = dec.Decode(resObj)
	}
	if err != nil {
		return fmt.Errorf("Failed to parse device response: %w", err)
	}
	return nil
}
