#!/bin/bash

set -e -x
source ${TEST_SRCDIR}/_main/cvdr/common_utils.sh
validate_components

CVD_HOST_PKG=${TEST_SRCDIR}/+_repo_rules+aosp_artifact/cvd-host_package.tar.gz
if [ ! -f ${CVD_HOST_PKG} ]; then
    echo "Cannot find CVD host package from ${CVD_HOST_PKG}"
    exit 1
fi
IMAGE_ZIP=${TEST_SRCDIR}/+_repo_rules+aosp_artifact/images.zip
if [ ! -f "${IMAGE_ZIP}" ]; then
    echo "Cannot find image zip file from ${IMAGE_ZIP}"
    exit 1
fi

HOSTNAME=$(cvdr host create)
cleanup() {
    cvdr host delete ${HOSTNAME}
}
trap cleanup EXIT ERR

cvdr create \
    --host=${HOSTNAME} \
    --local_cvd_host_pkg_src=${CVD_HOST_PKG} \
    --local_images_zip_src=${IMAGE_ZIP}

# Check the output of cvdr list
# TODO(b/448007486): cvdr list should print proper URL instead of showing <nil>.
# TODO(b/448007486): cvdr list should print proper ADB connection status.
ACTUAL_OUTPUT=$(cvdr list --host ${HOSTNAME})
EXPECTED_OUTPUT="${HOSTNAME} (<nil>/)
  cvd_1/1
  Status: Running
  ADB: not connected
  Displays: [720 x 1280 ( 320 )]
  Logs: <nil>/cvds/cvd_1/1/logs/"
diff <(echo ${EXPECTED_OUTPUT}) <(echo ${ACTUAL_OUTPUT})

# Check ADB connection
# TODO(b/448007486): Retrieve serial of the device from the output of cvdr list.
adb shell uptime
