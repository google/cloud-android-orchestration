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

# Pre-delete hook: Uploads device state to S3 before CVD deletion
# This preserves device state (apps, data, settings) for future restoration

set -e

INSTANCE_NAME="$1"
INSTANCE_DIR="${CVD_INSTANCE_DIR}"
S3_BUCKET="${CVD_S3_BUCKET:-cuttlefish-device-states}"
S3_PREFIX="${CVD_S3_PREFIX:-states}"
USERNAME="${CVD_USERNAME:-default}"

echo "[pre-delete] Backing up state for instance: $INSTANCE_NAME"
echo "[pre-delete] Instance directory: $INSTANCE_DIR"

# Verify instance directory exists
if [ ! -d "$INSTANCE_DIR" ]; then
    echo "[pre-delete] WARNING: Instance directory not found: $INSTANCE_DIR"
    echo "[pre-delete] Nothing to back up"
    exit 0
fi

# S3 key pattern: s3://bucket/prefix/username/instance-name/
S3_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/"

echo "[pre-delete] Uploading state to: $S3_PATH"

# Upload state files to S3
# Only include critical state files that need to be preserved
aws s3 sync "$INSTANCE_DIR/" "$S3_PATH" \
    --exclude "*" \
    --include "userdata.img" \
    --include "sdcard.img" \
    --include "persistent.img" \
    --include "instance_config.json" \
    --no-progress

if [ $? -eq 0 ]; then
    echo "[pre-delete] State backed up successfully"

    # Optional: Add metadata tags with timestamp
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Tag the userdata.img file with backup timestamp
    aws s3api put-object-tagging \
        --bucket "$S3_BUCKET" \
        --key "${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/userdata.img" \
        --tagging "TagSet=[{Key=LastBackup,Value=$TIMESTAMP},{Key=Instance,Value=$INSTANCE_NAME},{Key=Username,Value=$USERNAME}]" \
        2>/dev/null || echo "[pre-delete] Note: Could not set S3 tags (non-critical)"

    echo "[pre-delete] Backup completed at: $TIMESTAMP"
else
    echo "[pre-delete] ERROR: Failed to upload state to S3"
    exit 1
fi

exit 0
