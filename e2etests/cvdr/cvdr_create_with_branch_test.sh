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
