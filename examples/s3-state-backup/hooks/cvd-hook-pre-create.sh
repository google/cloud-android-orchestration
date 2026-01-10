#!/bin/bash
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Pre-create hook: Downloads device state from S3 before CVD creation
# This restores previously saved device state (apps, data, settings)

set -e

INSTANCE_NAME="$1"
INSTANCE_DIR="${CVD_INSTANCE_DIR}"
S3_BUCKET="${CVD_S3_BUCKET:-cuttlefish-device-states}"
S3_PREFIX="${CVD_S3_PREFIX:-states}"
USERNAME="${CVD_USERNAME:-default}"

echo "[pre-create] Restoring state for instance: $INSTANCE_NAME"
echo "[pre-create] Instance directory: $INSTANCE_DIR"

# Create instance directory if it doesn't exist
mkdir -p "$INSTANCE_DIR"

# S3 key pattern: s3://bucket/prefix/username/instance-name/
S3_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/"

echo "[pre-create] Checking for existing state at: $S3_PATH"

# Check if state exists in S3
if ! aws s3 ls "$S3_PATH" > /dev/null 2>&1; then
    echo "[pre-create] No existing state found in S3, this is a new instance"
    exit 0
fi

echo "[pre-create] Found existing state, downloading..."

# Download state files
# Only include critical state files that need to be restored
aws s3 sync "$S3_PATH" "$INSTANCE_DIR/" \
    --exclude "*" \
    --include "userdata.img" \
    --include "sdcard.img" \
    --include "persistent.img" \
    --include "instance_config.json" \
    --no-progress

if [ $? -eq 0 ]; then
    echo "[pre-create] State restored successfully"

    # Set proper ownership (adjust user/group as needed for your setup)
    chown -R vsoc-01:vsoc-01 "$INSTANCE_DIR/" 2>/dev/null || true
else
    echo "[pre-create] WARNING: Failed to download state from S3"
    exit 1
fi

exit 0
