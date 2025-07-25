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

# TODO: Retrieve the root endpoint of HO from `cvdr list`, and execute curl for
# checking some endpoints.

# TODO: Extend test for checking ADB connection
