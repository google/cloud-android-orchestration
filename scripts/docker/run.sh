#!/bin/bash

# Copyright 2024 Google Inc. All rights reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Shell script for running Cloud Orchestrator based on Docker
# Please run in the root directory of this repository.

# Specify docker API version for running cloud orchestrator
docker_api_version_code=1.43
docker_api_version_installed=$(docker version --format '{{.Client.APIVersion}}')
docker_api_version=$(\
  echo -e $docker_api_version_code\\n$docker_api_version_installed \
  | sort --version-sort \
  | head -n 1)

CONFIG_FILE=scripts/docker/conf.toml \
DOCKER_API_VERSION=$docker_api_version \
./cloud_orchestrator
