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

	"github.com/gorilla/mux"
)

// The controller implements the web API of the cloud orchestrator. It parses
// and validates requests from the client and passes the information to the
// relevant modules
type Controller struct {
	instanceManager InstanceManager
	sigServer       SignalingServer
}

func NewController(im InstanceManager, ss SignalingServer) *Controller {
	controller := &Controller{im, ss}
	controller.SetupRoutes()

	return controller
}

func (c *Controller) ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func (c *Controller) SetupRoutes() {
	router := mux.NewRouter()

	// Signaling Server Routes
	router.Handle("/connections/{connId}/messages", appHandler(func(w http.ResponseWriter, r *http.Request) error {
		return c.Messages(w, r)
	})).Methods("GET")
	router.Handle("/connections/{connId}/:forward", appHandler(func(w http.ResponseWriter, r *http.Request) error {
		return c.Forward(w, r)
	})).Methods("POST")
	router.Handle("/connections", appHandler(func(w http.ResponseWriter, r *http.Request) error {
		return c.CreateConnection(w, r)
	})).Methods("GET")

	router.Handle("/", appHandler(indexHandler))

	http.Handle("/", router)
}

type appHandler func(w http.ResponseWriter, r *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (c *Controller) CreateConnection(w http.ResponseWriter, r *http.Request) error {
	var msg NewConnMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	log.Println("id: ", msg.DeviceId)
	reply, err := c.sigServer.NewConnection(msg)
	if err != nil {
		return fmt.Errorf("Failed to communicate with device: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) Messages(w http.ResponseWriter, r *http.Request) error {
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
	reply, err := c.sigServer.Messages(id, start, count)
	if err != nil {
		return fmt.Errorf("Failed to get messages: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func (c *Controller) Forward(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["connId"]
	var msg ForwardMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return NewBadRequestError("Malformed JSON in request", err)
	}
	reply, err := c.sigServer.Forward(id, msg)
	if err != nil {
		return fmt.Errorf("Failed to send message to device: %w", err)
	}
	replyJSON(w, reply.Response, reply.StatusCode)
	return nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) error {
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
