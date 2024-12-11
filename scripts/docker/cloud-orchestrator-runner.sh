#!/usr/bin/env bash

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

script_location=`realpath -s $(dirname ${BASH_SOURCE[0]})`
cloud_android_orchestration_root_dir=$(realpath -s $script_location/../..)

pushd $cloud_android_orchestration_root_dir
CONFIG_FILE=scripts/docker/conf.toml ./cloud_orchestrator
popd
