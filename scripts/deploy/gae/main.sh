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

set -e

usage() {
  echo "usage: $0 -p project-name [-s service-name]"
}

project=
service="default"

while getopts ":hp:s:" opt; do
  case "${opt}" in
    h)
      usage
      exit 0
      ;;
    p)
      project="${OPTARG}"
      ;;
    s)
      service="${OPTARG}"
      ;;
    \?)
      echo "Invalid option: ${OPTARG}" >&2
      usage
      exit 1
      ;;
    :)
      echo "Invalid option: ${OPTARG} requires an argument" >&2
      usage
      exit 1
      ;;
  esac
done

network="default"

gcloud config set project $project

# Creates the App Engine if does not exist yet.
gcloud app describe 1> /dev/null 2> /dev/null
if [ $? -ne 0 ]; then
  gcloud app create
fi

gcloud services enable vpcaccess.googleapis.com

app_region=$(gcloud app describe --format="value(locationId)")

serverless_vpc_region="${app_region}1"
connector=$(gcloud compute networks vpc-access connectors list --filter="name:co-vpc-connector" \
  --region=${serverless_vpc_region} 2> /dev/null)
if [ -z "${connector}" ]; then
  gcloud compute networks vpc-access connectors create co-vpc-connector \
    --region=${serverless_vpc_region} \
    --network=${network} \
    --range=10.8.0.0/28
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)

# Generates app.yaml
export PROJECT_ID="${project}"
export SERVERLESS_VPC_REGION="${serverless_vpc_region}"
export SERVICE="${service}"

readonly VARS='\
  $PROJECT_ID\
  $SERVERLESS_VPC_REGION\
  $SERVICE'

envsubst "$VARS" < ${script_dir}/app.yaml.tmpl > app.yaml

# Generates conf.toml
export PROJECT_ID="${project}"
envsubst '$PROJECT_ID' < ${script_dir}/conf.toml.tmpl > conf.toml

sudo apt-get install google-cloud-sdk-app-engine-go

# Deploy
gcloud app deploy
