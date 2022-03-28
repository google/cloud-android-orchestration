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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	caoapi "cloud-android-orchestration/api/v1"

	"github.com/gorilla/mux"
)

type ErrorMsg struct {
	Error string `json:"error"`
}

type InfraConfig struct {
	IceServers []IceServer `json:"ice_servers"`
}
type IceServer struct {
	URLs []string `json:"urls"`
}

type HTTPHandler func(http.ResponseWriter, *http.Request) error

type DeviceFilesRequest struct {
	devId string
	path  string
	w     http.ResponseWriter
	r     *http.Request
}

type SignalingServer interface {
	// These endpoints in the SignalingServer return the (possibly modified)
	// response from the Host Orchestrator and the status code if it was
	// able to communicate with it, otherwise it returns an error.
	NewConnection(msg caoapi.NewConnMsg, user UserInfo) (*caoapi.SServerResponse, error)
	Forward(id string, msg caoapi.ForwardMsg, user UserInfo) (*caoapi.SServerResponse, error)
	Messages(id string, start int, count int, user UserInfo) (*caoapi.SServerResponse, error)

	// Forwards the reques to the device's server unless it's a for a file that
	// the signaling server needs to serve itself.
	ServeDeviceFiles(params DeviceFilesRequest, user UserInfo) error
}

type DeviceDesc struct {
	// The (internal) network address of the host where the cuttlefish device is
	// running. The address can either be an IPv4, IPv6 or a domain name.
	Addr string
	// The id under which the cuttlefish device is registered with the host
	// orchestrator (can be different from the id used in the cloud orchestrator)
	LocalId string
}

type InstanceManager interface {
	DeviceFromId(name string, user UserInfo) (DeviceDesc, error)
}

// The controller implements the web API of the cloud orchestrator. It parses
// and validates requests from the client and passes the information to the
// relevant modules
type Controller struct {
	instanceManager InstanceManager
	sigServer       SignalingServer
	accountManager  AccountManager
}

var DEFAULT_INFRA_CONFIG = InfraConfig{
	IceServers: []IceServer{
		IceServer{URLs: []string{"stun:stun.l.google.com:19302"}},
	},
}

func NewController(im InstanceManager, ss SignalingServer, am AccountManager) *Controller {
	controller := &Controller{im, ss, am}
	controller.SetupRoutes()

	return controller
}

func (c *Controller) ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func (c *Controller) SetupRoutes() {
	router := mux.NewRouter()

	// Signaling Server Routes
	router.Handle("/v1/connections/{connId}/messages", HTTPHandler(c.accountManager.Authenticate(c.Messages))).Methods("GET")
	router.Handle("/v1/connections/{connId}/:forward", HTTPHandler(c.accountManager.Authenticate(c.Forward))).Methods("POST")
	router.Handle("/v1/connections", HTTPHandler(c.accountManager.Authenticate(c.CreateConnection))).Methods("POST")
	router.Handle("/v1/devices/{deviceId}/files{path:/.+}", HTTPHandler(c.accountManager.Authenticate(c.GetDeviceFiles))).Methods("GET")

	// Global routes
	router.HandleFunc("/infra_config", func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/220891296): Make this configurable
		replyJSON(w, DEFAULT_INFRA_CONFIG, http.StatusOK)
	}).Methods("GET")

	router.Handle("/", HTTPHandler(c.accountManager.Authenticate(indexHandler)))

	http.Handle("/", router)
}

// Intercept errors returned by the HTTPHandler and transform them into HTTP
// error responses
func (fn HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, " ", r.URL, " ", r.RemoteAddr)
	if err := fn(w, r); err != nil {
		log.Println("Error: ", err)
		status := http.StatusInternalServerError
		var e *AppError
		if errors.As(err, &e) {
			status = e.StatusCode
		}
		http.Error(w, err.Error(), status)
	}
}

func (c *Controller) GetDeviceFiles(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	devId := mux.Vars(r)["deviceId"]
	path := mux.Vars(r)["path"]
	return c.sigServer.ServeDeviceFiles(DeviceFilesRequest{devId, path, w, r}, user)
}

func (c *Controller) CreateConnection(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	var msg caoapi.NewConnMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	log.Println("id: ", msg.DeviceId)
	reply, err := c.sigServer.NewConnection(msg, user)
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
	reply, err := c.sigServer.Messages(id, start, count, user)
	if err != nil {
		return fmt.Errorf("Failed to get messages: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) Forward(w http.ResponseWriter, r *http.Request, user UserInfo) error {
	id := mux.Vars(r)["connId"]
	var msg caoapi.ForwardMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	reply, err := c.sigServer.Forward(id, msg, user)
	if err != nil {
		return fmt.Errorf("Failed to send message to device: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
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
