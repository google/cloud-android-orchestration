// Copyright 2022 Google LLC
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

package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
)

// The ForwardingSignalingServer implements the SignalingServer interface by
// communicating with the host orchestrator in the appropriate instance and
// forwarding all requests to it.
type ForwardingSignalingServer struct {
	instanceManager InstanceManager
}

func NewForwardingSignalingServer(im InstanceManager) *ForwardingSignalingServer {
	return &ForwardingSignalingServer{im}
}

func (s *ForwardingSignalingServer) NewConnection(zone string, host string, msg apiv1.NewConnMsg, user UserInfo) (*apiv1.SServerResponse, error) {
	hostAddr, err := s.instanceManager.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	var resErr apiv1.ErrorMsg
	var reply apiv1.NewConnReply
	status, err := POSTRequest(hostURL(hostAddr, "/polled_connections", ""), apiv1.NewConnMsg{msg.DeviceId}, &reply, &resErr)
	if err != nil {
		return nil, err
	}
	if resErr.Error != "" {
		log.Println("The device host returned an error: ", resErr.Error)
		return &apiv1.SServerResponse{Response: resErr, StatusCode: status}, nil
	}
	// Add the device id to the connection id to reference it in future calls
	reply.ConnId = encodeConnId(reply.ConnId, msg.DeviceId)
	return &apiv1.SServerResponse{Response: reply, StatusCode: status}, nil
}

func (s *ForwardingSignalingServer) Forward(zone string, host string, id string, msg apiv1.ForwardMsg, user UserInfo) (*apiv1.SServerResponse, error) {
	dec, err := decodeConnId(id)
	if err != nil {
		return nil, NewNotFoundError("Invalid connection Id", err)
	}
	connId := dec.ConnId
	hostAddr, err := s.instanceManager.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	var resErr apiv1.ErrorMsg
	var reply interface{}
	status, err := POSTRequest(hostURL(hostAddr, "/polled_connections/"+connId+"/:forward", ""), msg, &reply, &resErr)
	if err != nil {
		return nil, err
	}
	return &apiv1.SServerResponse{reply, status}, nil
}

func (s *ForwardingSignalingServer) Messages(zone string, host string, id string, start int, count int, user UserInfo) (*apiv1.SServerResponse, error) {
	dec, err := decodeConnId(id)
	if err != nil {
		return nil, NewNotFoundError("Invalid connection id", err)
	}
	connId := dec.ConnId
	hostAddr, err := s.instanceManager.GetHostAddr(zone, host)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("start=%d", start)
	if count > 0 {
		query = fmt.Sprintf("%s&count=%d", query, count)
	}
	var resErr apiv1.ErrorMsg
	var reply interface{}
	status, err := GETRequest(hostURL(hostAddr, "/polled_connections/"+connId+"/messages", query), &reply, &resErr)
	if err != nil {
		return nil, err
	}
	return &apiv1.SServerResponse{reply, status}, nil
}

func (s *ForwardingSignalingServer) ServeDeviceFiles(zone string, host string, params DeviceFilesRequest, user UserInfo) error {
	hostAddr, err := s.instanceManager.GetHostAddr(zone, host)
	if err != nil {
		return err
	}
	if shouldIntercept(params.path) {
		http.ServeFile(params.w, params.r, fmt.Sprintf("intercept%s", params.path))
	} else {
		devUrl, err := url.Parse(hostURL(hostAddr, "", ""))
		if err != nil {
			return err
		}
		devProxy := httputil.NewSingleHostReverseProxy(devUrl)

		params.r.URL.Path = fmt.Sprintf("/devices/%s/files%s", params.devId, params.path)
		devProxy.ServeHTTP(params.w, params.r)
	}
	return nil
}

func hostURL(addr string, path string, query string) string {
	url := "http://" + addr + ":1080" + path
	if query != "" {
		url += "?" + query
	}
	return url
}

// Returns the http response's status code or an error.
// If the status code indicates success (in the 2xx range) the response will be
// in resObj, otherwise resErr will contain the error message.
func POSTRequest(url string, msg interface{}, resObj interface{}, resErr *apiv1.ErrorMsg) (int, error) {
	jsonBody, err := json.Marshal(msg)
	if err != nil {
		return -1, fmt.Errorf("Failed to parse JSON request: %w", err)
	}
	reqBody := bytes.NewBuffer(jsonBody)
	res, err := http.Post(url, "application/json", reqBody)
	if err != nil {
		return -1, fmt.Errorf("Failed to connecto to device host: %w", err)
	}
	defer res.Body.Close()
	return parseReply(res, resObj, resErr)
}

func GETRequest(url string, resObj interface{}, resErr *apiv1.ErrorMsg) (int, error) {
	res, err := http.Get(url)
	if err != nil {
		return -1, fmt.Errorf("Failed to connect to device host: %w", err)
	}
	defer res.Body.Close()
	return parseReply(res, resObj, resErr)
}

func parseReply(res *http.Response, resObj interface{}, resErr *apiv1.ErrorMsg) (int, error) {
	var err error
	dec := json.NewDecoder(res.Body)
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = dec.Decode(resErr)
	} else {
		err = dec.Decode(resObj)
	}
	if err != nil {
		return -1, fmt.Errorf("Failed to parse device response: %w", err)
	}
	return res.StatusCode, nil
}

func shouldIntercept(path string) bool {
	return path == "/js/server_connector.js"
}

const CONN_ID_SEPARATOR string = ":"

func encodeConnId(connId string, deviceId string) string {
	// Both the device id and the connection id could have any characters in it,
	// the connection id is base64 enocoded to ensure it doesn't have the
	// separator ('/').
	return deviceId + CONN_ID_SEPARATOR + base64.StdEncoding.EncodeToString([]byte(connId))
}

type IdDecodeResult struct {
	ConnId string
	DevId  string
}

func decodeConnId(connId string) (IdDecodeResult, error) {
	idx := strings.LastIndex(connId, CONN_ID_SEPARATOR)
	if idx < 0 {
		return IdDecodeResult{}, fmt.Errorf("Malformed connection id (Missing separator): %s", connId)
	}
	devId := connId[:idx]
	bytes, err := base64.StdEncoding.DecodeString(connId[idx+1:])
	if err != nil {
		return IdDecodeResult{}, err
	}
	return IdDecodeResult{string(bytes), devId}, nil
}
