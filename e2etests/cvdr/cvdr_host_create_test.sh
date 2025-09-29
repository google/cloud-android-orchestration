#!/bin/bash

set -e -x
source ${TEST_SRCDIR}/_main/cvdr/common_utils.sh
validate_components

cleanup() {
    cvdr host delete ${HOSTNAME[@]}
}
trap cleanup EXIT ERR
for i in $(seq 0 1); do
    HOSTNAME[i]=$(cvdr host create)
    if [[ ! $(cvdr host list) =~ ${HOSTNAME[i]} ]]; then
        echo "Host ${HOSTNAME[i]} doesn't exist"
        exit 1
    fi
done
