#!/bin/bash

# Copyright (C) 2024 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Create GCP image from https://github.com/google/android-cuttlefish host
# images.
#
# Image Name Format: cf-debian11-amd64-<YYYYMMDD>-<commit>
#
# IMPORTANT!!! Artifact download URL only work for registered users (404 for guests)
# https://github.com/actions/upload-artifact/issues/51

set -e

usage() {
  echo "usage: $0 -s head-commit-sha -t /path/to/github_auth_token.txt -p project-id"
}

commit_sha=
github_auth_token_filename=
project_id=

while getopts "hs:t:p:" opt; do
  case "${opt}" in
    h)
      usage
      exit 0
      ;;
    s)
      commit_sha="${OPTARG}"
      ;;
    t)
      github_auth_token_filename="${OPTARG}"
      ;;
    p)
      project_id="${OPTARG}"
      ;;
    \?)
      usage; exit 1
      ;;
  esac
done

# Paginate until finding the relevant artifact entry
artifact=
url="https://api.github.com/repos/google/android-cuttlefish/actions/artifacts?per_page=100"
while [[ -z "${artifact}" ]]; do
  res=$(curl --include -L \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $(cat ${github_auth_token_filename})" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "${url}" 2> /dev/null)
  json=$(echo "${res}" | sed -n '/^{$/,/^}$/p')
  artifact=$(echo "${json}" | jq ".artifacts[] | select(.workflow_run.head_sha == \"${commit_sha}\" and .name == \"image_gce_debian11_amd64\")")
  if [[ -z "${artifact}" ]]; then
    link=$(echo "${res}" | sed "/^link:/!d")
    if [[ ${link}  == *"rel=\"next\""* ]]; then
      url=$(echo "${link}" | grep -Eo "<https://[^ ]*>; rel=\"next\"" | grep -Eo "https://.*page=[0-9]+")
    else
      echo "Artifact not found for commit: ${commit_sha}"
      exit 1
    fi
  fi
done

bucket_name="${project_id}-cf-host-image-upload"

updated_at=$(echo "${artifact}" | jq -r ".updated_at")
date_suffix=$(date -u -d ${updated_at} +"%Y%m%d")
name=cf-debian11-amd64-${date_suffix}-${commit_sha:0:7}

function cleanup {
  rm "image.zip" 2> /dev/null
  rm "image.tar.gz" 2> /dev/null
  gcloud storage rm --recursive gs://${bucket_name}
}

trap cleanup EXIT

echo "Downloading artifact ..."
download_url=$(echo $artifact | jq -r ".archive_download_url")
curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $(cat ${github_auth_token_filename})" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  --output image.zip \
  ${download_url}

unzip image.zip

gcloud storage buckets create gs://${bucket_name} --location="us-east1" --project="${project_id}"
gcloud storage cp image.tar.gz  gs://${bucket_name}/${name}.tar.gz

echo "Creating image ..."
gcloud compute images create ${name} \
  --source-uri gs://${bucket_name}/${name}.tar.gz \
  --project ${project_id} \
  --family "cf-debian11-amd64"
