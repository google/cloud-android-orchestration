#!/bin/bash

# Copyright 2023 Google Inc. All rights reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Internal global flags for enabling checks (1 for true, 0 for false)
ENABLE_FMT=${ENABLE_FMT:-1}
ENABLE_TEST=${ENABLE_TEST:-1}
ENABLE_VET=${ENABLE_VET:-1}
ENABLE_LINT=${ENABLE_LINT:-1}
ENABLE_STATICCHECK=${ENABLE_STATICCHECK:-1}
ENABLE_BUILD=${ENABLE_BUILD:-1}

# Stop on error flag (1 for true, 0 for false)
STOP_ON_ERROR=${STOP_ON_ERROR:-0}

RESULTS_DIR=${1:-"/tmp/health_check_results"}

# Exit on unset variable
set -u

if [[ "$STOP_ON_ERROR" -ne 0 ]]; then  
  set -e
fi

# Change directory to the project root
cd "$(git rev-parse --show-toplevel)"

check_dependencies() {
  local deps=(go) # please extend this list as needed
  for dep in "${deps[@]}"; do
    if ! type "$dep" > /dev/null; then
      echo "$dep is not installed"
      return 1
    fi
  done
  return 0
}

install_golint() {
  if ! type golint >/dev/null 2>&1; then
    echo "golint could not be found, attempting to install..."
    GO111MODULE=on go install golang.org/x/lint/golint@latest
    export PATH="$(go env GOPATH)/bin:$PATH"
  fi
}

install_staticcheck() {
  if ! type staticcheck >/dev/null 2>&1; then
    echo "staticcheck could not be found, installing..."
    GO111MODULE=on go install honnef.co/go/tools/cmd/staticcheck@latest
    export PATH="$(go env GOPATH)/bin:$PATH"
  fi
}

mkdir -p "$RESULTS_DIR"
FMT_RESULTS_FILE="$RESULTS_DIR/gofmt-results.txt"
TEST_RESULTS_FILE="$RESULTS_DIR/gotest-results.txt"
VET_RESULTS_FILE="$RESULTS_DIR/govet-results.txt"
LINT_RESULTS_FILE="$RESULTS_DIR/golint-results.txt"
STATICCHECK_RESULTS_FILE="$RESULTS_DIR/staticcheck-results.txt"
BUILD_RESULTS_FILE="$RESULTS_DIR/gobuild-results.txt"

run_gofmt() {
  echo "Running go fmt..."
  go fmt ./... > "$FMT_RESULTS_FILE"
  if [ -s "$FMT_RESULTS_FILE" ]; then
    echo "go fmt needs to format files. Please check the formatting results in $FMT_RESULTS_FILE"
    cat "$FMT_RESULTS_FILE"
    return 1
  fi
  echo "All Go files are properly formatted."
}

run_gotest() {
  echo "Running go test..."
  if ! go test ./... > "$TEST_RESULTS_FILE" 2>&1; then
    echo "go test found issues. Please check the test results in $TEST_RESULTS_FILE"
    cat "$TEST_RESULTS_FILE"
    return 1
  fi
  echo "No test issues found."
}

run_govet() {
  echo "Running go vet..."
  if ! go vet ./... > "$VET_RESULTS_FILE" 2>&1; then
    echo "go vet found issues. Please check the vetting results in $VET_RESULTS_FILE"
    cat "$VET_RESULTS_FILE"
    return 1
  fi
  echo "No vetting issues found."
}

run_golint() {
  echo "Running golint..."
  golint ./... > "$LINT_RESULTS_FILE"
  if [ -s "$LINT_RESULTS_FILE" ]; then
    echo "golint found issues. Please check the linting results in $LINT_RESULTS_FILE"
    return 1
  else
    echo "No linting issues found."
  fi
}

run_staticcheck() {
  echo "Running staticcheck..."
  if ! staticcheck ./... > "$STATICCHECK_RESULTS_FILE" 2>&1; then
    echo "staticcheck found issues. Please check the linting results in $STATICCHECK_RESULTS_FILE"
    return 1
  else
    echo "No static analysis issues found."
  fi
}

run_gobuild() {
  echo "Running go build..."
  if ! go build ./... > "$BUILD_RESULTS_FILE" 2>&1; then
    echo "go build failed. Please check the build results in $BUILD_RESULTS_FILE"
    cat "$BUILD_RESULTS_FILE"
    return 1
  fi
  echo "Go build succeeded."
}

if ! check_dependencies; then
  echo "Please install the missing dependencies and try again."
  (( STOP_ON_ERROR )) && exit 1
fi
if (( ENABLE_LINT )); then
  install_golint
fi
if (( ENABLE_STATICCHECK )); then
  install_staticcheck
fi

error_count=0

# Check if feature flags are set and run the corresponding checks
(( ENABLE_FMT )) && { run_gofmt || ((error_count++)); }
(( ENABLE_TEST )) && { run_gotest || ((error_count++)); }
(( ENABLE_VET )) && { run_govet || ((error_count++)); }
(( ENABLE_LINT )) && { run_golint || ((error_count++)); }
(( ENABLE_STATICCHECK )) && { run_staticcheck || ((error_count++)); }
(( ENABLE_BUILD )) && { run_gobuild || ((error_count++)); }

if [ "$error_count" -ne 0 ]; then
  echo "Presubmit failed for  $error_count types of checks. Please review checks results at $RESULTS_DIR for more details."
else
  echo "All checks passed!"
fi
