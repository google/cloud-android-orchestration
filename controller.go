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
	sigServer SignalingServer
}

func NewController(ss SignalingServer) *Controller {
	controller := &Controller{ss}
	controller.SetupRoutes()

	return controller
}

func (c *Controller) ListenAndServe(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func (c *Controller) SetupRoutes() {
	router := mux.NewRouter()

	// Signaling Server Routes
	router.HandleFunc("/connections/{connId}/messages", func(w http.ResponseWriter, r *http.Request) {
		c.Messages(w, r)
	}).Methods("GET")
	router.HandleFunc("/connections/{connId}/:forward", func(w http.ResponseWriter, r *http.Request) {
		c.Forward(w, r)
	}).Methods("POST")
	router.HandleFunc("/connections", func(w http.ResponseWriter, r *http.Request) {
		c.CreateConnection(w, r)
	})

	router.HandleFunc("/", indexHandler)

	http.Handle("/", router)
}

func (c *Controller) CreateConnection(w http.ResponseWriter, r *http.Request) {
	var msg NewConnMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		log.Println("Failed to parse json  from client: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("id: ", msg.DeviceId)
	reply := c.sigServer.NewConnection(msg)
	replyJSON(w, reply)
}

func (c *Controller) Messages(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["connId"]
	start, err := intFormValue(r, "start", 0)
	if err != nil {
		http.Error(w, "Invalid value for start field", http.StatusBadRequest)
		return
	}
	// -1 means all messages
	count, err := intFormValue(r, "count", -1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reply, err := c.sigServer.Messages(id, start, count)
	if err != nil {
		log.Println("Failed to get messages: ", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	replyJSON(w, reply)
}

func (c *Controller) Forward(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["connId"]
	var msg ForwardMsg
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		log.Println("Failed to parse json from client: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := c.sigServer.Forward(id, msg); err != nil {
		log.Println("Failed to send message to device: ", err)
		http.Error(w, "Device disconnected", http.StatusNotFound)
		return
	}
	replyJSON(w, "ok")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Home page")
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
		log.Println("Invalid ", name, " value: ", str)
	}
	return i, e
}

// Send a JSON http response to the client
func replyJSON(w http.ResponseWriter, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}
