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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"

	apiv1 "github.com/google/cloud-android-orchestration/api/v1"

	"github.com/gorilla/mux"
)

// The controller implements the web API of the cloud orchestrator. It parses
// and validates requests from the client and passes the information to the
// relevant modules
type Controller struct {
	infraConfig     apiv1.InfraConfig
	opsConfig       OperationsConfig
	instanceManager InstanceManager
	sigServer       SignalingServer
	accountManager  AccountManager
}

func NewController(
	servers []string,
	opsConfig OperationsConfig,
	im InstanceManager,
	ss SignalingServer,
	am AccountManager) *Controller {
	infraCfg := buildInfraCfg(servers)
	return &Controller{infraCfg, opsConfig, im, ss, am}
}

func (c *Controller) Handler() http.Handler {
	router := mux.NewRouter()
	hf := &HostForwarder{newReverseProxy: newReverseProxyBuilder(c.instanceManager)}

	// Signaling Server Routes
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections/{connID}/messages", HTTPHandler(c.accountManager.Authenticate(c.messages))).Methods("GET")
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections/{connID}/:forward", HTTPHandler(c.accountManager.Authenticate(c.forward))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections", HTTPHandler(c.accountManager.Authenticate(c.createConnection))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}/devices/{deviceId}/files{path:/.+}", HTTPHandler(c.accountManager.Authenticate(c.getDeviceFiles))).Methods("GET")

	// Instance Manager Routes
	router.Handle("/v1/zones/{zone}/hosts", c.createHostHTTPHandler()).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts", HTTPHandler(c.accountManager.Authenticate(c.listHosts))).Methods("GET")
	// Waits for the specified operation to be DONE or for the request to approach the specified deadline,
	// `503 Service Unavailable` error will be returned if the deadline is reached and the operation is not done.
	// Be prepared to retry if the deadline was reached.
	// It returns the expected response of the operation in case of success. If the original method returns no
	// data on success, such as `Delete`, response will be empty. If the original method is standard
	// `Get`/`Create`/`Update`, the response should be the relevant resource.
	router.Handle("/v1/zones/{zone}/operations/{operation}/:wait", HTTPHandler(c.accountManager.Authenticate(c.waitOperation))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}", HTTPHandler(c.accountManager.Authenticate(c.deleteHost))).Methods("DELETE")

	// Host Orchestrator Proxy Routes
	router.PathPrefix(
		"/v1/zones/{zone}/hosts/{host}/{resource:devices|operations|cvds|userartifacts}").Handler(hf.Handler())

	// Infra route
	router.HandleFunc("/v1/zones/{zone}/hosts/{host}/infra_config", func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/220891296): Make this configurable
		replyJSON(w, c.infraConfig, http.StatusOK)
	}).Methods("GET")

	c.accountManager.RegisterAuthHandlers(router)

	// Global routes
	router.Handle("/", HTTPHandler(c.accountManager.Authenticate(indexHandler)))

	// http.Handle("/", router)
	return router
}

func (c *Controller) createHostHTTPHandler() HTTPHandler {
	if c.opsConfig.CreateHostDisabled {
		return notAllowedHttpHandler
	}
	return c.accountManager.Authenticate(c.createHost)
}

// Intercept errors returned by the HTTPHandler and transform them into HTTP
// error responses
func (h HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, " ", r.URL, " ", r.RemoteAddr)
	if err := h(w, r); err != nil {
		log.Println("Error: ", err)
		var e *AppError
		if errors.As(err, &e) {
			replyJSON(w, e.JSONResponse(), e.StatusCode)
		} else {
			replyJSON(w, apiv1.Error{ErrorMsg: "Internal Server Error"}, http.StatusInternalServerError)
		}
	}
}

type ReverseProxyFactory func(zone, host string) (*httputil.ReverseProxy, error)

func newReverseProxyBuilder(im InstanceManager) ReverseProxyFactory {
	return func(zone, host string) (*httputil.ReverseProxy, error) {
		client, err := im.GetHostClient(zone, host)
		if err != nil {
			return nil, err
		}
		return client.GetReverseProxy(), nil
	}
}

type HostForwarder struct {
	newReverseProxy ReverseProxyFactory
}

func (f *HostForwarder) Handler() HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		zone := vars["zone"]
		if zone == "" {
			return fmt.Errorf("invalid url missing zone value: %q", r.URL.String())
		}
		host := vars["host"]
		if host == "" {
			return fmt.Errorf("invalid url missing host value: %q", r.URL.String())
		}
		proxy, err := f.buildReverseProxy(zone, host)
		if err != nil {
			return err
		}
		proxy.ServeHTTP(w, r)
		return nil
	}
}

func (f *HostForwarder) buildReverseProxy(zone, host string) (*httputil.ReverseProxy, error) {
	proxy, err := f.newReverseProxy(zone, host)
	if err != nil {
		return nil, err
	}
	oldDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		split := strings.SplitN(r.URL.Path, "hosts/"+host, 2)
		r.URL.Path = split[1]
		oldDirector(r)
	}
	return proxy, nil
}

func (c *Controller) getDeviceFiles(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	devId := mux.Vars(r)["deviceId"]
	path := mux.Vars(r)["path"]
	return c.sigServer.ServeDeviceFiles(getZone(r), getHost(r), DeviceFilesRequest{devId, path, w, r}, user)
}

func (c *Controller) createConnection(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	var msg apiv1.NewConnMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	log.Println("id: ", msg.DeviceId)
	reply, err := c.sigServer.NewConnection(getZone(r), getHost(r), msg, user)
	if err != nil {
		return fmt.Errorf("Failed to communicate with device: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) messages(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	id := mux.Vars(r)["connID"]
	start, err := intFormValue(r, "start", 0)
	if err != nil {
		return NewBadRequestError("Invalid value for start field", err)
	}
	// -1 means all messages
	count, err := intFormValue(r, "count", -1)
	if err != nil {
		return NewBadRequestError("Invalid value for count field", err)
	}
	reply, err := c.sigServer.Messages(getZone(r), getHost(r), id, start, count, user)
	if err != nil {
		return fmt.Errorf("Failed to get messages: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) forward(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	id := mux.Vars(r)["connID"]
	var msg apiv1.ForwardMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	reply, err := c.sigServer.Forward(getZone(r), getHost(r), id, msg, user)
	if err != nil {
		return fmt.Errorf("Failed to send message to device: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) createHost(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	var msg apiv1.CreateHostRequest
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	op, err := c.instanceManager.CreateHost(getZone(r), &msg, user)
	if err != nil {
		return err
	}
	replyJSON(w, op, http.StatusOK)
	return nil
}

func (c *Controller) listHosts(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	listReq, err := BuildListHostsRequest(r)
	if err != nil {
		return err
	}
	res, err := c.instanceManager.ListHosts(getZone(r), user, listReq)
	if err != nil {
		return err
	}
	replyJSON(w, res, http.StatusOK)
	return nil
}

func (c *Controller) deleteHost(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	name := mux.Vars(r)["host"]
	res, err := c.instanceManager.DeleteHost(getZone(r), user, name)
	if err != nil {
		return err
	}
	replyJSON(w, res, http.StatusOK)
	return nil
}

func (c *Controller) waitOperation(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	name := mux.Vars(r)["operation"]
	op, err := c.instanceManager.WaitOperation(getZone(r), user, name)
	if err != nil {
		return err
	}
	replyJSON(w, op, http.StatusOK)
	return nil
}

const (
	queryParamMaxResults = "maxResults"
)

func BuildListHostsRequest(r *http.Request) (*ListHostsRequest, error) {
	maxResultsRaw := r.URL.Query().Get(queryParamMaxResults)
	maxResults, err := uint32Value(maxResultsRaw)
	if err != nil {
		return nil, NewInvalidQueryParamError(queryParamMaxResults, maxResultsRaw, err)
	}
	res := &ListHostsRequest{
		MaxResults: maxResults,
		PageToken:  r.URL.Query().Get("pageToken"),
	}
	return res, nil
}

func uint32Value(value string) (uint32, error) {
	if value == "" {
		return uint32(0), nil
	}
	uint64v, err := strconv.ParseUint(value, 10, 32)
	return uint32(uint64v), err
}

func buildInfraCfg(servers []string) apiv1.InfraConfig {
	iceServers := []apiv1.IceServer{}
	for _, server := range servers {
		iceServers = append(iceServers, apiv1.IceServer{URLs: []string{server}})
	}
	return apiv1.InfraConfig{
		IceServers: iceServers,
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	fmt.Fprintln(w, "Home page")
	return nil
}

// Get an int form value from the request. A form value can be in the query
// string or in the request body in either URL or Multipart encoding. If there
// is no form parameter with the given name the default value is returned. If a
// parameter with that name exists but the value can't be converted to an int an
// error is returned.
func intFormValue(r *http.Request, name string, def int) (int, error) {
	str := r.FormValue(name)
	if str == "" {
		return def, nil
	}
	i, e := strconv.Atoi(str)
	if e != nil {
		return 0, fmt.Errorf("Invalid %s value: %s", name, str)
	}
	return i, nil
}

// Send a JSON http response to the client
func replyJSON(w http.ResponseWriter, obj any, statusCode int) error {
	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}

func getZone(r *http.Request) string {
	return mux.Vars(r)["zone"]
}

func getHost(r *http.Request) string {
	return mux.Vars(r)["host"]
}

func notAllowedHttpHandler(w http.ResponseWriter, r *http.Request) error {
	return NewMethodNotAllowedError("Operation is disabled", nil)
}
