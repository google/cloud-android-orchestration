#!/bin/bash

set -e -x
source ${TEST_SRCDIR}/_main/cvdr/common_utils.sh
validate_components

HOSTNAME=$(cvdr host create)
cvdr host delete ${HOSTNAME}
