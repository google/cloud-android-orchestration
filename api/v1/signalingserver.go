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

package v1

type Error struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"error"`
}

type NewConnMsg struct {
	DeviceId string `json:"device_id"`
}

type NewConnReply struct {
	ConnId     string `json:"connection_id"`
	DeviceInfo any    `json:"device_info"`
}

type ForwardMsg struct {
	Payload any `json:"payload"`
}

type SServerResponse struct {
	Response   any
	StatusCode int
}

type InfraConfig struct {
	IceServers []IceServer `json:"ice_servers"`
}

type IceServer struct {
	URLs []string `json:"urls"`
}
