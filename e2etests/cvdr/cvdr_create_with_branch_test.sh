#!/bin/bash

set -e -x
source ${TEST_SRCDIR}/_main/cvdr/common_utils.sh
validate_components

SERVICE=$(cvdr get_config user_default_service || cvdr get_config system_default_service)
SERVICE_URL=$(cvdr get_config "services[\"$SERVICE\"].service_url")

HOSTNAME=$(cvdr host create)
cleanup() {
    cvdr host delete ${HOSTNAME}
}
trap cleanup EXIT

cvdr create \
    --host=${HOSTNAME} \
    --branch=aosp-android-latest-release \
    --build_target=aosp_cf_x86_64_only_phone-userdebug

# Check the output of cvdr list
# TODO(b/448209030): cvdr list should print proper ADB connection status.
ACTUAL_OUTPUT=$(cvdr list --host ${HOSTNAME})
EXPECTED_OUTPUT="Host: ${HOSTNAME}
  WebUI: $SERVICE_URL/v1/zones/local/hosts/${HOSTNAME}/
  Group: cvd
    Instance: 1
      Status: Running
      ADB: not connected
      Displays: [720 x 1280 ( 320 )]
      Logs: $SERVICE_URL/v1/zones/local/hosts/${HOSTNAME}/cvds/cvd/1/logs/"
diff <(echo ${EXPECTED_OUTPUT}) <(echo ${ACTUAL_OUTPUT})

# Check ADB connection
# TODO(b/448209030): Retrieve serial of the device from the output of cvdr list.
adb shell uptime
