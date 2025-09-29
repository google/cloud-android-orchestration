#!/bin/bash

set -e -x
source ${TEST_SRCDIR}/_main/cvdr/common_utils.sh
validate_components

HOSTNAME=$(cvdr host create)
cleanup() {
    cvdr host delete ${HOSTNAME}
}
trap cleanup EXIT ERR

cvdr create \
    --host=${HOSTNAME} \
    --branch=aosp-android-latest-release \
    --build_target=aosp_cf_x86_64_only_phone-userdebug

# Check the output of cvdr list
# TODO(b/448007486): cvdr list should print proper URL instead of showing <nil>.
# TODO(b/448007486): cvdr list should print proper ADB connection status.
ACTUAL_OUTPUT=$(cvdr list --host ${HOSTNAME})
EXPECTED_OUTPUT="${HOSTNAME} (<nil>/)
  cvd/1
  Status: Running
  ADB: not connected
  Displays: [720 x 1280 ( 320 )]
  Logs: <nil>/cvds/cvd/1/logs/"
diff <(echo ${EXPECTED_OUTPUT}) <(echo ${ACTUAL_OUTPUT})

# Check ADB connection
# TODO(b/448007486): Retrieve serial of the device from the output of cvdr list.
adb shell uptime
