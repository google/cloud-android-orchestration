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
	"strconv"

	apiv1 "cloud-android-orchestration/api/v1"

	"github.com/gorilla/mux"
)

// The controller implements the web API of the cloud orchestrator. It parses
// and validates requests from the client and passes the information to the
// relevant modules
type Controller struct {
	infraConfig     apiv1.InfraConfig
	instanceManager InstanceManager
	sigServer       SignalingServer
	accountManager  AccountManager
}

func NewController(servers []string, im InstanceManager, ss SignalingServer, am AccountManager) *Controller {
	infraCfg := buildInfraCfg(servers)
	controller := &Controller{infraCfg, im, ss, am}
	controller.SetupRoutes()

	return controller
}

func (c *Controller) ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func (c *Controller) SetupRoutes() {
	router := mux.NewRouter()

	// Signaling Server Routes
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections/{connId}/messages", HTTPHandler(c.accountManager.Authenticate(c.Messages))).Methods("GET")
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections/{connId}/:forward", HTTPHandler(c.accountManager.Authenticate(c.Forward))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}/connections", HTTPHandler(c.accountManager.Authenticate(c.CreateConnection))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}/devices/{deviceId}/files{path:/.+}", HTTPHandler(c.accountManager.Authenticate(c.GetDeviceFiles))).Methods("GET")

	// Instance Manager Routes
	router.Handle("/v1/zones/{zone}/hosts", HTTPHandler(c.accountManager.Authenticate(c.CreateHost))).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts", HTTPHandler(c.accountManager.Authenticate(c.ListHosts))).Methods("GET")

	// Infra route
	router.HandleFunc("/v1/zones/{zone}/hosts/{host}/infra_config", func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/220891296): Make this configurable
		replyJSON(w, c.infraConfig, http.StatusOK)
	}).Methods("GET")

	// Global routes
	router.Handle("/", HTTPHandler(c.accountManager.Authenticate(indexHandler)))

	http.Handle("/", router)
}

// Intercept errors returned by the HTTPHandler and transform them into HTTP
// error responses
func (fn HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, " ", r.URL, " ", r.RemoteAddr)
	if err := fn(w, r); err != nil {
		log.Println("Error: ", err)
		var e *AppError
		if errors.As(err, &e) {
			replyJSON(w, e.JSONResponse(), e.StatusCode)
		} else {
			replyJSON(w, apiv1.ErrorMsg{Error: "Internal Server Error"}, http.StatusInternalServerError)
		}
	}
}

func (c *Controller) GetDeviceFiles(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	devId := mux.Vars(r)["deviceId"]
	path := mux.Vars(r)["path"]
	return c.sigServer.ServeDeviceFiles(getZone(r), getHost(r), DeviceFilesRequest{devId, path, w, r}, user)
}

func (c *Controller) CreateConnection(w http.ResponseWriter, r *http.Request, user UserInfo) error {
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

func (c *Controller) Messages(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	id := mux.Vars(r)["connId"]
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

func (c *Controller) Forward(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	id := mux.Vars(r)["connId"]
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

func (c *Controller) CreateHost(w http.ResponseWriter, r *http.Request, user UserInfo) error {
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

func (c *Controller) ListHosts(w http.ResponseWriter, r *http.Request, user UserInfo) error {
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
func replyJSON(w http.ResponseWriter, obj interface{}, statusCode int) error {
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
