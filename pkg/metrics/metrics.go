// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate protoc --go_out=. --go_opt=paths=source_relative clientanalytics.proto
//go:generate protoc --go_out=. --go_opt=paths=source_relative internal_user_log.proto
package metrics

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
)

const (
	ClientTypeValue    = 16
	LogSourceValue     = 971
	LogSourceNameValue = "CUTTLEFISH_METRICS"
	ToolName           = "cvdr"
)

var (
	UserTypeVal    = UserType_GOOGLE
	TestReferences = []string{"cvdr"}
)

func currentTimeMillis() int64 {
	return time.Now().UnixMilli()
}

func encodeLogRequest(extension []byte) ([]byte, error) {
	currentTimeMs := currentTimeMillis()

	clientInfo := &ClientInfo{
		ClientType: proto.Int32(ClientTypeValue),
	}

	logEvent := &LogEvent{
		EventTimeMs:     proto.Int64(currentTimeMs),
		SourceExtension: extension,
	}

	req := &LogRequest{
		ClientInfo:    clientInfo,
		LogSource:     proto.Int32(LogSourceValue),
		RequestTimeMs: proto.Int64(currentTimeMs),
		LogEvent:      []*LogEvent{logEvent},
		LogSourceName: proto.String(LogSourceNameValue),
	}

	return proto.Marshal(req)
}

func createAndEncodeATestLogEventInternal(commandLine string) ([]byte, error) {
	userKey := uuid.New().String()
	runID := uuid.New().String()

	atestStartEvent := &AtestLogEventInternal_AtestStartEvent{
		CommandLine:    proto.String(commandLine),
		TestReferences: TestReferences,
		Os:             proto.String(runtime.GOOS),
	}

	logEvent := &AtestLogEventInternal{
		UserKey:  proto.String(userKey),
		RunId:    proto.String(runID),
		UserType: &UserTypeVal,
		ToolName: proto.String(ToolName),
		Event:    &AtestLogEventInternal_AtestStartEvent_{AtestStartEvent: atestStartEvent},
	}

	return proto.Marshal(logEvent)
}

type ClearcutServerEnvironment int

const (
	Local ClearcutServerEnvironment = iota + 1
	Staging
	Prod
)

func clearcutServerURL(server ClearcutServerEnvironment) string {
	switch server {
	case Local:
		return "http://localhost:27910/log"
	case Staging:
		return "https://play.googleapis.com:443/staging/log"
	case Prod:
		return "https://play.googleapis.com:443/log"
	default:
		panic("Invalid Clearcut server configuration")
	}
}

func postRequest(output []byte) error {
	clearcutURL := clearcutServerURL(Prod)

	resp, err := http.Post(clearcutURL, "application/json", bytes.NewBuffer(output))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("HTTP error code: %d", resp.StatusCode)
			log.Printf("HTTP response body: %s", string(body))
			return err
		}

		return errors.New("metrics message failed with status code " + resp.Status)
	}
	return nil
}

func SendLaunchCommand(commandLine string) error {
	encoded, err := createAndEncodeATestLogEventInternal(commandLine)
	if err != nil {
		return err
	}

	data, err := encodeLogRequest(encoded)
	if err != nil {
		return err
	}

	return postRequest(data)
}
