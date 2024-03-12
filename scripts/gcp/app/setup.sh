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

# One time project setup.

usage() {
  echo "usage: $0 -p project-name"
}

project=

while getopts ":hp:" opt; do
  case "${opt}" in
    h)
      usage
      exit 0
      ;;
    p)
      project="${OPTARG}"
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

app_region=$(gcloud app describe --format="value(locationId)")
# Note: Two locations, which are called europe-west and us-central in App Engine commands and in the Google Cloud
# console, are called europe-west1 and us-central1, respectively, elsewhere in Google documentation.
# https://cloud.google.com/appengine/docs/standard/locations
if [ "${app_region}" = "europe-west" ] || [ "${app_region}" = "us-central" ]; then
  app_region="${app_region}1"
fi
echo "App Region: ${app_region}"

# Grant GAE service account admin role.
gae_service_account_name="$project@appspot.gserviceaccount.com"
gcloud projects add-iam-policy-binding $project \
  --member="serviceAccount:$gae_service_account_name" \
  --role="roles/compute.admin"

gcloud services enable vpcaccess.googleapis.com

serverless_vpc_region="${app_region}"
connector=$(gcloud compute networks vpc-access connectors list --filter="name:co-vpc-connector" \
  --region=${serverless_vpc_region})
if [ -z "${connector}" ]; then
  gcloud compute networks vpc-access connectors create co-vpc-connector \
    --region=${serverless_vpc_region} \
    --network=${network} \
    --range=10.8.0.0/28
fi
