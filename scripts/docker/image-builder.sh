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

script_location=`realpath -s $(dirname ${BASH_SOURCE[0]})`
cloud_android_orchestration_root_dir=$(realpath -s $script_location/../..)

if [[ "$1" == "" ]]; then
    tag=cuttlefish-cloud-orchestration
else
    tag=$1
fi

# Build docker image
pushd $cloud_android_orchestration_root_dir
DOCKER_BUILDKIT=1 docker build \
    --force-rm \
    --no-cache \
    --target runner-docker \
    -t $tag \
    .
popd
