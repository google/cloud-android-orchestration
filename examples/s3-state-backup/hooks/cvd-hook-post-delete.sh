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

# Post-delete hook: Runs after CVD deletion completes
# Use this for cleanup, notifications, logging, etc.

set -e

INSTANCE_NAME="$1"
USERNAME="${CVD_USERNAME:-default}"

echo "[post-delete] Instance $INSTANCE_NAME deleted successfully"
echo "[post-delete] User: $USERNAME"

# Example: Send notification
# curl -X POST https://monitoring.example.com/events \
#     -H "Content-Type: application/json" \
#     -d "{\"event\":\"cvd_deleted\",\"instance\":\"$INSTANCE_NAME\",\"user\":\"$USERNAME\"}"

# Example: Log to syslog
# logger -t cvd-lifecycle "CVD deleted: $INSTANCE_NAME (user: $USERNAME)"

# Example: Update inventory database
# psql -h db.example.com -U cvd -c \
#     "UPDATE instances SET deleted_at = NOW() WHERE name = '$INSTANCE_NAME'"

# Example: Clean up any temporary resources
# rm -rf /tmp/cvd-temp-$INSTANCE_NAME 2>/dev/null || true

exit 0
