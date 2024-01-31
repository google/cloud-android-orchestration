#!/bin/bash

# Copyright 2024 Google Inc. All rights reserved.
#
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

# Generates an OIDC token from a local service account key file.
# Adapted from https://cloud.google.com/iap/docs/authentication-howto#obtaining_an_oidc_token_from_a_local_service_account_key_file

set -euo pipefail

usage() {
  echo "usage: $0 -k /path/to/key -c iap-client-id"
}

key_file_path=
iap_client_id=

while getopts ":hk:c:" opt; do
  case "${opt}" in
    h)
      usage
      exit 0
      ;;
    k)
      key_file_path="${OPTARG}"
      ;;
    c)
      iap_client_id="${OPTARG}"
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

get_token() {
  # Get the bearer token in exchange for the service account credentials.
  local service_account_key_file_path="${1}"
  local iap_client_id="${2}"

  local iam_scope="https://www.googleapis.com/auth/iam"
  local oauth_token_uri="https://www.googleapis.com/oauth2/v4/token"

  local private_key_id="$(cat "${service_account_key_file_path}" | jq -r '.private_key_id')"
  local client_email="$(cat "${service_account_key_file_path}" | jq -r '.client_email')"
  local private_key="$(cat "${service_account_key_file_path}" | jq -r '.private_key')"
  local issued_at="$(date +%s)"
  local expires_at="$((issued_at + 600))"
  local header="{'alg':'RS256','typ':'JWT','kid':'${private_key_id}'}"
  local header_base64="$(echo "${header}" | base64)"
  local payload="{'iss':'${client_email}','aud':'${oauth_token_uri}','exp':${expires_at},'iat':${issued_at},'sub':'${client_email}','target_audience':'${iap_client_id}'}"
  local payload_base64="$(echo "${payload}" | base64)"
  local signature_base64="$(printf %s "${header_base64}.${payload_base64}" | openssl dgst -binary -sha256 -sign <(printf '%s\n' "${private_key}")  | base64)"
  local assertion="${header_base64}.${payload_base64}.${signature_base64}"
  local token_payload="$(curl -s \
    --data-urlencode "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer" \
    --data-urlencode "assertion=${assertion}" \
    https://www.googleapis.com/oauth2/v4/token)"
  local bearer_id_token="$(echo "${token_payload}" | jq -r '.id_token')"
  echo "${bearer_id_token}"
}

token=$(get_token "${key_file_path}" "${iap_client_id}")

printf ${token}

