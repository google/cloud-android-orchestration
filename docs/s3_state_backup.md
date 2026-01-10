# S3 State Backup for Cuttlefish Devices

This guide explains how to implement automatic S3 backup and restore of Cuttlefish device state using Host Orchestrator lifecycle hooks.

## Implementation Options

There are **two implementation approaches** available:

### 1. **Go Implementation** (Recommended for Production)

**Location:** `examples/s3-state-backup/go-implementation/`

✅ **Pros:**
- Type-safe, compile-time error checking
- Fully unit testable with mock S3
- Better error handling and retry logic
- No AWS CLI dependency (smaller Docker image)
- Built-in AWS SDK features (multipart upload, retries)
- Production-grade performance and reliability

❌ **Cons:**
- Requires modifying Host Orchestrator source code
- Needs Go compilation
- More upfront development time

**Best for:** Production deployments, long-term maintenance, high reliability requirements

📖 **Documentation:** See `examples/s3-state-backup/go-implementation/README.md`

### 2. **Bash Script Implementation** (Quick Prototyping)

**Location:** `examples/s3-state-backup/hooks/`

✅ **Pros:**
- Simple, easy to understand
- No recompilation needed
- Quick to prototype and iterate
- Familiar to most sysadmins

❌ **Cons:**
- Requires AWS CLI in Docker image
- Error handling is clunky
- Not unit testable
- String manipulation is fragile

**Best for:** MVPs, prototypes, proof-of-concepts, learning the architecture

📖 **Documentation:** See this document below

### Recommendation

- **Start with bash** if you want to validate the concept quickly
- **Migrate to Go** for production deployments
- **Use Go from the start** if you have Go expertise and want production quality immediately

---

## Architecture Overview

```
Client Request (Create CVD)
    ↓
Cloud Orchestrator (proxies request)
    ↓
Host Orchestrator
    ↓
[PRE-CREATE HOOK] ← Download state from S3
    ↓
Create Cuttlefish Instance (with restored state)
    ↓
[POST-CREATE HOOK] ← Optional post-creation tasks

---

Client Request (Delete CVD)
    ↓
Cloud Orchestrator (proxies request)
    ↓
Host Orchestrator
    ↓
[PRE-DELETE HOOK] ← Upload state to S3 (BACKUP!)
    ↓
Delete Cuttlefish Instance
    ↓
[POST-DELETE HOOK] ← Cleanup tasks
```

## What Gets Backed Up

The following device state files are critical for persistence:

- `userdata.img` - User data partition (apps, settings, user files)
- `sdcard.img` - SD card contents (if used)
- `persistent.img` - Persistent data partition (if configured)
- `instance_config.json` - Device configuration metadata

## Implementation Steps

### Step 1: Modify Host Orchestrator (android-cuttlefish repository)

Since Host Orchestrator is in a separate repository, you'll need to:

1. Clone the android-cuttlefish repository:
   ```bash
   git clone https://github.com/google/android-cuttlefish
   cd android-cuttlefish/frontend/src/host_orchestrator
   ```

2. Add lifecycle hook support to the CVD handlers.

#### File: `android-cuttlefish/frontend/src/host_orchestrator/orchestrator/cvd.go`

Add hook execution logic:

```go
package orchestrator

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
)

// Lifecycle hook types
const (
    HookPreCreate   = "pre-create"
    HookPostCreate  = "post-create"
    HookPreDelete   = "pre-delete"
    HookPostDelete  = "post-delete"
)

// RunLifecycleHook executes a lifecycle hook script if it exists
func RunLifecycleHook(hookType, instanceName, instanceDir string) error {
    hookScript := fmt.Sprintf("/usr/local/bin/cvd-hook-%s.sh", hookType)

    // Check if hook exists
    if _, err := os.Stat(hookScript); os.IsNotExist(err) {
        log.Printf("Hook %s not configured, skipping", hookType)
        return nil
    }

    log.Printf("Running lifecycle hook: %s for instance %s", hookType, instanceName)

    cmd := exec.Command(hookScript, instanceName)
    cmd.Env = append(os.Environ(),
        fmt.Sprintf("CVD_INSTANCE_NAME=%s", instanceName),
        fmt.Sprintf("CVD_INSTANCE_DIR=%s", instanceDir),
        fmt.Sprintf("CVD_HOOK_TYPE=%s", hookType),
    )

    // Capture output for debugging
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Hook %s failed: %v\nOutput: %s", hookType, err, string(output))
        return fmt.Errorf("hook %s failed: %w", hookType, err)
    }

    log.Printf("Hook %s completed successfully", hookType)
    if len(output) > 0 {
        log.Printf("Hook output: %s", string(output))
    }

    return nil
}
```

#### Modify CVD Creation Handler

In the file that handles CVD creation (typically `cvd.go` or similar):

```go
func (s *Server) CreateCVD(w http.ResponseWriter, r *http.Request) {
    // ... existing code to parse request ...

    instanceName := req.InstanceName
    instanceDir := getInstanceDirectory(instanceName)

    // NEW: Run pre-create hook (downloads state from S3)
    if err := RunLifecycleHook(HookPreCreate, instanceName, instanceDir); err != nil {
        // Log error but continue - might be first-time creation
        log.Printf("Pre-create hook failed (continuing anyway): %v", err)
    }

    // ... existing CVD creation logic ...
    // This includes launch_cvd execution

    // NEW: Run post-create hook
    if err := RunLifecycleHook(HookPostCreate, instanceName, instanceDir); err != nil {
        log.Printf("Post-create hook failed: %v", err)
    }

    // ... return response ...
}
```

#### Modify CVD Deletion Handler

```go
func (s *Server) DeleteCVD(w http.ResponseWriter, r *http.Request) {
    // ... existing code to parse request ...

    instanceName := mux.Vars(r)["id"]
    instanceDir := getInstanceDirectory(instanceName)

    // NEW: Run pre-delete hook (uploads state to S3) - CRITICAL!
    if err := RunLifecycleHook(HookPreDelete, instanceName, instanceDir); err != nil {
        // This is critical - we want to save state before deletion
        log.Printf("WARNING: Pre-delete hook failed, state may not be backed up: %v", err)
        // Optionally: return error to prevent deletion if backup fails
        // return err
    }

    // ... existing CVD deletion logic ...
    // This includes stopping cvd and cleanup

    // NEW: Run post-delete hook
    if err := RunLifecycleHook(HookPostDelete, instanceName, instanceDir); err != nil {
        log.Printf("Post-delete hook failed: %v", err)
    }

    // ... return response ...
}
```

### Step 2: Create Hook Scripts

Create lifecycle hook scripts that will be included in your custom Docker image.

#### File: `hooks/cvd-hook-pre-create.sh`

```bash
#!/bin/bash
set -e

INSTANCE_NAME="$1"
INSTANCE_DIR="${CVD_INSTANCE_DIR}"
S3_BUCKET="${CVD_S3_BUCKET:-cuttlefish-device-states}"
S3_PREFIX="${CVD_S3_PREFIX:-states}"
USERNAME="${CVD_USERNAME:-default}"

echo "[pre-create] Restoring state for instance: $INSTANCE_NAME"

# Create instance directory if it doesn't exist
mkdir -p "$INSTANCE_DIR"

# S3 key pattern: s3://bucket/prefix/username/instance-name/
S3_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/"

# Check if state exists in S3
if ! aws s3 ls "$S3_PATH" > /dev/null 2>&1; then
    echo "[pre-create] No existing state found in S3, this is a new instance"
    exit 0
fi

echo "[pre-create] Found existing state, downloading from: $S3_PATH"

# Download state files
aws s3 sync "$S3_PATH" "$INSTANCE_DIR/" \
    --exclude "*" \
    --include "userdata.img" \
    --include "sdcard.img" \
    --include "persistent.img" \
    --include "instance_config.json" \
    --no-progress

echo "[pre-create] State restored successfully"

# Set proper ownership
chown -R vsoc-01:vsoc-01 "$INSTANCE_DIR/" 2>/dev/null || true

exit 0
```

#### File: `hooks/cvd-hook-pre-delete.sh`

```bash
#!/bin/bash
set -e

INSTANCE_NAME="$1"
INSTANCE_DIR="${CVD_INSTANCE_DIR}"
S3_BUCKET="${CVD_S3_BUCKET:-cuttlefish-device-states}"
S3_PREFIX="${CVD_S3_PREFIX:-states}"
USERNAME="${CVD_USERNAME:-default}"

echo "[pre-delete] Backing up state for instance: $INSTANCE_NAME"

# Verify instance directory exists
if [ ! -d "$INSTANCE_DIR" ]; then
    echo "[pre-delete] Instance directory not found: $INSTANCE_DIR"
    exit 0
fi

# S3 key pattern: s3://bucket/prefix/username/instance-name/
S3_PATH="s3://${S3_BUCKET}/${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/"

echo "[pre-delete] Uploading state to: $S3_PATH"

# Upload state files to S3
aws s3 sync "$INSTANCE_DIR/" "$S3_PATH" \
    --exclude "*" \
    --include "userdata.img" \
    --include "sdcard.img" \
    --include "persistent.img" \
    --include "instance_config.json" \
    --no-progress

echo "[pre-delete] State backed up successfully"

# Optional: Add metadata tag with timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
aws s3api put-object-tagging \
    --bucket "$S3_BUCKET" \
    --key "${S3_PREFIX}/${USERNAME}/${INSTANCE_NAME}/userdata.img" \
    --tagging "TagSet=[{Key=LastBackup,Value=$TIMESTAMP},{Key=Instance,Value=$INSTANCE_NAME}]" \
    2>/dev/null || true

exit 0
```

#### File: `hooks/cvd-hook-post-create.sh` (Optional)

```bash
#!/bin/bash
set -e

INSTANCE_NAME="$1"

echo "[post-create] Instance $INSTANCE_NAME created successfully"

# Optional: Send notification, log to monitoring, etc.
# curl -X POST https://monitoring.example.com/events \
#     -d "instance=$INSTANCE_NAME&event=created"

exit 0
```

#### File: `hooks/cvd-hook-post-delete.sh` (Optional)

```bash
#!/bin/bash
set -e

INSTANCE_NAME="$1"

echo "[post-delete] Instance $INSTANCE_NAME deleted successfully"

# Optional: Cleanup, notifications, etc.

exit 0
```

### Step 3: Build Custom Host Docker Image

Create a custom Dockerfile that includes the modified Host Orchestrator and hook scripts.

#### File: `docker/Dockerfile.s3-hooks`

```dockerfile
# Start from the official cuttlefish image or build your own
FROM us-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-orchestration:unstable

# Install AWS CLI for S3 operations
RUN apt-get update && \
    apt-get install -y awscli && \
    rm -rf /var/lib/apt/lists/*

# Copy hook scripts
COPY hooks/cvd-hook-pre-create.sh /usr/local/bin/cvd-hook-pre-create.sh
COPY hooks/cvd-hook-pre-delete.sh /usr/local/bin/cvd-hook-pre-delete.sh
COPY hooks/cvd-hook-post-create.sh /usr/local/bin/cvd-hook-post-create.sh
COPY hooks/cvd-hook-post-delete.sh /usr/local/bin/cvd-hook-post-delete.sh

# Make hooks executable
RUN chmod +x /usr/local/bin/cvd-hook-*.sh

# Copy modified Host Orchestrator binary
# (You'll need to build this from the android-cuttlefish repo)
COPY host_orchestrator /usr/local/bin/host_orchestrator

# Ensure proper permissions
RUN chmod +x /usr/local/bin/host_orchestrator

# Default S3 configuration (can be overridden via env vars)
ENV CVD_S3_BUCKET=cuttlefish-device-states
ENV CVD_S3_PREFIX=states
ENV CVD_USERNAME=default

# Use the modified host orchestrator
CMD ["/usr/local/bin/host_orchestrator"]
```

#### Build the Image

```bash
# Build Host Orchestrator with hooks support
cd android-cuttlefish/frontend/src/host_orchestrator
go build -o host_orchestrator .

# Copy to docker build context
cp host_orchestrator /path/to/cloud-android-orchestration/docker/

# Build Docker image
cd /path/to/cloud-android-orchestration/docker
docker build -f Dockerfile.s3-hooks -t cuttlefish-orchestration-s3:latest .

# Tag for your registry
docker tag cuttlefish-orchestration-s3:latest myregistry.com/cuttlefish-orchestration-s3:v1.0

# Push to registry
docker push myregistry.com/cuttlefish-orchestration-s3:v1.0
```

### Step 4: Configure Cloud Orchestrator

Update your Cloud Orchestrator configuration to use the custom image.

#### File: `conf.toml`

```toml
[InstanceManager]
Type = "docker"
HostOrchestratorProtocol = "http"

[InstanceManager.Docker]
DockerImageName = "myregistry.com/cuttlefish-orchestration-s3:v1.0"
HostOrchestratorPort = 2080

# S3 Configuration Environment Variables
# These will be passed to host containers
[InstanceManager.Docker.Environment]
CVD_S3_BUCKET = "my-cuttlefish-states"
CVD_S3_PREFIX = "device-states"
AWS_REGION = "us-west-2"
AWS_ACCESS_KEY_ID = "${AWS_ACCESS_KEY_ID}"      # From environment
AWS_SECRET_ACCESS_KEY = "${AWS_SECRET_ACCESS_KEY}"  # From environment
```

#### Modify Docker Instance Manager to Pass Environment Variables

You'll need to update `pkg/app/instances/docker.go` to pass S3 configuration:

```go
func (m *DockerInstanceManager) createDockerContainer(ctx context.Context, user accounts.User) (string, error) {
    config := &container.Config{
        AttachStdin: true,
        Image:       m.Config.Docker.DockerImageName,
        Tty:         true,
        Labels:      dockerLabelsDict(user),
        Env: []string{
            fmt.Sprintf("CVD_USERNAME=%s", user.Username()),
            fmt.Sprintf("CVD_S3_BUCKET=%s", m.Config.Docker.S3Bucket),
            fmt.Sprintf("CVD_S3_PREFIX=%s", m.Config.Docker.S3Prefix),
            fmt.Sprintf("AWS_REGION=%s", m.Config.Docker.AWSRegion),
            // AWS credentials from environment or config
            fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", os.Getenv("AWS_ACCESS_KEY_ID")),
            fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", os.Getenv("AWS_SECRET_ACCESS_KEY")),
        },
    }

    // ... rest of function
}
```

### Step 5: AWS IAM Configuration

Create an IAM policy for S3 access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:ListBucket",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::cuttlefish-device-states/*",
        "arn:aws:s3:::cuttlefish-device-states"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObjectTagging",
        "s3:GetObjectTagging"
      ],
      "Resource": "arn:aws:s3:::cuttlefish-device-states/*"
    }
  ]
}
```

## Usage

Once deployed, S3 backup/restore happens automatically:

### Creating a Device (First Time)

```bash
cvdr --branch=aosp-main --build_target=aosp_cf_x86_64_phone-userdebug create
```

- Pre-create hook runs: No state in S3, continues normally
- CVD created with fresh state
- Pre-delete hook will backup state when deleted

### Creating a Device (Restoring Previous State)

```bash
# Use the same instance name as before
cvdr --instance_name=my-device --branch=aosp-main --build_target=aosp_cf_x86_64_phone-userdebug create
```

- Pre-create hook runs: Finds state in S3
- Downloads `userdata.img`, `sdcard.img`, etc.
- CVD created with restored state (apps, data intact!)

### Deleting a Device (Auto-Backup)

```bash
cvdr delete <host-id>
```

- Pre-delete hook runs: Uploads current state to S3
- CVD deleted
- State preserved in S3 for future restoration

## S3 Bucket Structure

```
s3://cuttlefish-device-states/
└── device-states/                    # Prefix
    ├── alice/                        # Username
    │   ├── my-phone/                 # Instance name
    │   │   ├── userdata.img
    │   │   ├── sdcard.img
    │   │   ├── persistent.img
    │   │   └── instance_config.json
    │   └── test-device/
    │       └── ...
    └── bob/
        └── my-tablet/
            └── ...
```

## Monitoring and Debugging

### View Hook Logs

Hook output is logged by Host Orchestrator:

```bash
# Get Host Orchestrator logs
docker logs <host-container-id>

# Look for lines like:
# [pre-create] Restoring state for instance: my-device
# [pre-delete] Backing up state for instance: my-device
```

### Check S3 Backup Status

```bash
# List all backups for a user
aws s3 ls s3://cuttlefish-device-states/device-states/alice/ --recursive

# Check backup metadata
aws s3api get-object-tagging \
    --bucket cuttlefish-device-states \
    --key device-states/alice/my-device/userdata.img
```

### Manual State Operations

```bash
# Manual restore (if needed)
aws s3 sync s3://cuttlefish-device-states/device-states/alice/my-device/ \
    /path/to/instance/dir/

# Manual backup
aws s3 sync /path/to/instance/dir/ \
    s3://cuttlefish-device-states/device-states/alice/my-device/ \
    --exclude "*" --include "userdata.img"
```

## Advanced Configuration

### Per-User S3 Buckets

Modify the pre-create/pre-delete scripts to use user-specific buckets:

```bash
S3_BUCKET="cuttlefish-${USERNAME}"
```

### Compression

Add compression to save S3 storage costs:

```bash
# In pre-delete hook
gzip -c "$INSTANCE_DIR/userdata.img" | \
    aws s3 cp - "s3://${S3_PATH}/userdata.img.gz"

# In pre-create hook
aws s3 cp "s3://${S3_PATH}/userdata.img.gz" - | \
    gunzip > "$INSTANCE_DIR/userdata.img"
```

### Versioning

Enable S3 versioning for the bucket:

```bash
aws s3api put-bucket-versioning \
    --bucket cuttlefish-device-states \
    --versioning-configuration Status=Enabled
```

### Lifecycle Policies

Automatically delete old backups:

```json
{
  "Rules": [
    {
      "Id": "Delete old device states",
      "Status": "Enabled",
      "Expiration": {
        "Days": 30
      },
      "NoncurrentVersionExpiration": {
        "NoncurrentDays": 7
      }
    }
  ]
}
```

## Troubleshooting

### Hook Not Running

Check if hook scripts exist and are executable:
```bash
docker exec <host-container> ls -l /usr/local/bin/cvd-hook-*.sh
```

### AWS Credentials Issues

Verify credentials are available in container:
```bash
docker exec <host-container> env | grep AWS
docker exec <host-container> aws s3 ls s3://cuttlefish-device-states/
```

### State Not Restoring

- Check instance name matches previous instance
- Verify S3 path exists: `aws s3 ls s3://bucket/prefix/user/instance/`
- Check hook logs for errors

### Performance Impact

- Downloads happen before CVD creation (adds startup time)
- Uploads happen before deletion (adds deletion time)
- Consider using async uploads in post-delete hook for faster deletion

## Security Considerations

1. **Use IAM Roles** instead of access keys when possible (EC2 instance roles)
2. **Encrypt S3 bucket** with SSE-S3 or SSE-KMS
3. **Restrict bucket access** to specific IAM roles
4. **Enable bucket logging** for audit trail
5. **Use S3 bucket policies** to prevent accidental deletion

## Summary

This implementation provides:

✅ Automatic backup on device deletion
✅ Automatic restore on device creation
✅ Per-user isolation in S3
✅ Clean separation of concerns (hooks in Host Orchestrator)
✅ No Cloud Orchestrator code changes needed
✅ Works with all clients (cvdr, Web UI, API)
✅ Transparent to end users

The device state is preserved across deletions and recreations, enabling persistent virtual devices in the cloud.
