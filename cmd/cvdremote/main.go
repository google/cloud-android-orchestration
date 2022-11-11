// Copyright 2022 Google LLC
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

package main

import (
	"github.com/google/cloud-android-orchestration/pkg/cli"
	"os"
)

func main() {
	opts := &cli.CommandOptions{
		IOStreams: cli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Args:      os.Args[1:],
	}
	if err := cli.NewCVDRemoteCommand(opts).Execute(); err != nil {
		os.Exit(1)
	}
}
