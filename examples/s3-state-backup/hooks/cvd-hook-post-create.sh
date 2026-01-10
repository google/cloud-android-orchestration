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

# Post-create hook: Runs after CVD creation completes
# Use this for notifications, logging, monitoring, etc.

set -e

INSTANCE_NAME="$1"
INSTANCE_DIR="${CVD_INSTANCE_DIR}"
USERNAME="${CVD_USERNAME:-default}"

echo "[post-create] Instance $INSTANCE_NAME created successfully"
echo "[post-create] User: $USERNAME"
echo "[post-create] Directory: $INSTANCE_DIR"

# Example: Send notification to monitoring system
# curl -X POST https://monitoring.example.com/events \
#     -H "Content-Type: application/json" \
#     -d "{\"event\":\"cvd_created\",\"instance\":\"$INSTANCE_NAME\",\"user\":\"$USERNAME\"}"

# Example: Log to syslog
# logger -t cvd-lifecycle "CVD created: $INSTANCE_NAME (user: $USERNAME)"

# Example: Update inventory database
# psql -h db.example.com -U cvd -c \
#     "INSERT INTO instances (name, username, created_at) VALUES ('$INSTANCE_NAME', '$USERNAME', NOW())"

exit 0
