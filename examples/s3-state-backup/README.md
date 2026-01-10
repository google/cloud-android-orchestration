# S3 State Backup Example

This directory contains example files for implementing S3 state backup for Cuttlefish devices.

## Overview

These examples demonstrate how to automatically backup and restore Cuttlefish device state (userdata, apps, settings) to Amazon S3 using lifecycle hooks in the Host Orchestrator.

## Files

- `hooks/cvd-hook-pre-create.sh` - Downloads device state from S3 before CVD creation
- `hooks/cvd-hook-pre-delete.sh` - Uploads device state to S3 before CVD deletion
- `hooks/cvd-hook-post-create.sh` - Optional post-creation tasks
- `hooks/cvd-hook-post-delete.sh` - Optional post-deletion tasks
- `conf.toml` - Example Cloud Orchestrator configuration with S3 enabled

## Quick Start

### 1. Build Custom Host Image

These hook scripts must be included in your Host Orchestrator Docker image. See the [full documentation](../../docs/s3_state_backup.md) for detailed instructions.

```bash
# Copy hooks to your docker build context
cp hooks/*.sh /path/to/docker/build/context/

# Ensure they're included in your Dockerfile
# COPY hooks/*.sh /usr/local/bin/
# RUN chmod +x /usr/local/bin/cvd-hook-*.sh
```

### 2. Configure Cloud Orchestrator

Use the provided `conf.toml` as a template:

```bash
cp conf.toml /path/to/your/conf.toml

# Edit to match your setup
vim /path/to/your/conf.toml
```

Key configuration:

```toml
[InstanceManager.Docker.S3StateBackup]
Enabled = true
Bucket = "your-bucket-name"
Prefix = "states"
AWSRegion = "us-west-2"
```

### 3. Set AWS Credentials

```bash
# Using environment variables (development)
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"

# Or use IAM instance roles (production - recommended)
# Attach a role with S3 read/write permissions to your EC2 instance
```

### 4. Create S3 Bucket

```bash
aws s3 mb s3://cuttlefish-device-states --region us-west-2

# Optional: Enable versioning
aws s3api put-bucket-versioning \
    --bucket cuttlefish-device-states \
    --versioning-configuration Status=Enabled

# Optional: Enable encryption
aws s3api put-bucket-encryption \
    --bucket cuttlefish-device-states \
    --server-side-encryption-configuration '{
        "Rules": [{
            "ApplyServerSideEncryptionByDefault": {
                "SSEAlgorithm": "AES256"
            }
        }]
    }'
```

### 5. Run Cloud Orchestrator

```bash
docker run -d \
    -p 8080:8080 \
    -e CONFIG_FILE="/conf.toml" \
    -e AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID}" \
    -e AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY}" \
    -v $(pwd)/conf.toml:/conf.toml \
    -v /var/run/docker.sock:/var/run/docker.sock \
    cuttlefish-cloud-orchestrator:latest
```

## Usage

Once configured, backup and restore happens automatically:

### Create Device (First Time)

```bash
cvdr --instance_name=my-phone \
     --branch=aosp-main \
     --build_target=aosp_cf_x86_64_phone-userdebug \
     create
```

- No state in S3, creates fresh device
- State will be backed up when deleted

### Delete Device (Auto-Backup)

```bash
cvdr delete my-phone
```

- Pre-delete hook uploads state to S3
- Device deleted
- State preserved: `s3://bucket/states/username/my-phone/`

### Create Device (Restore State)

```bash
cvdr --instance_name=my-phone \
     --branch=aosp-main \
     --build_target=aosp_cf_x86_64_phone-userdebug \
     create
```

- Pre-create hook finds state in S3
- Downloads userdata.img, sdcard.img, etc.
- Device created with restored state
- Apps and data from previous session intact!

## S3 Bucket Structure

```
s3://cuttlefish-device-states/
└── states/                     # Prefix (configurable)
    ├── alice/                  # Username
    │   ├── my-phone/           # Instance name
    │   │   ├── userdata.img
    │   │   ├── sdcard.img
    │   │   ├── persistent.img
    │   │   └── instance_config.json
    │   └── test-tablet/
    │       └── ...
    └── bob/
        └── work-phone/
            └── ...
```

## Customization

### Modify Hook Scripts

Edit the hook scripts to customize behavior:

- **Compression**: Add gzip to save storage
- **Encryption**: Add client-side encryption
- **Selective backup**: Change which files to backup
- **Notifications**: Add webhook calls
- **Logging**: Send logs to centralized system

### Example: Add Compression

In `cvd-hook-pre-delete.sh`:

```bash
# Compress before upload
gzip -c "$INSTANCE_DIR/userdata.img" | \
    aws s3 cp - "s3://${S3_PATH}/userdata.img.gz"
```

In `cvd-hook-pre-create.sh`:

```bash
# Download and decompress
aws s3 cp "s3://${S3_PATH}/userdata.img.gz" - | \
    gunzip > "$INSTANCE_DIR/userdata.img"
```

## IAM Policy

Minimal IAM policy for S3 access:

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
        "s3:PutObjectTagging"
      ],
      "Resource": [
        "arn:aws:s3:::cuttlefish-device-states/*",
        "arn:aws:s3:::cuttlefish-device-states"
      ]
    }
  ]
}
```

## Troubleshooting

### Verify Hooks Are Installed

```bash
docker exec <host-container-id> ls -la /usr/local/bin/cvd-hook-*.sh
```

### Check Hook Logs

```bash
docker logs <host-container-id> | grep -E '\[pre-create\]|\[pre-delete\]'
```

### Test AWS Credentials

```bash
docker exec <host-container-id> aws s3 ls s3://cuttlefish-device-states/
```

### Manually Verify Backup

```bash
# List backups for a user
aws s3 ls s3://cuttlefish-device-states/states/alice/ --recursive

# Download a backup
aws s3 cp s3://cuttlefish-device-states/states/alice/my-phone/userdata.img ./
```

## Security Best Practices

1. **Use IAM Roles** instead of access keys in production
2. **Enable S3 encryption** (SSE-S3 or SSE-KMS)
3. **Enable versioning** to protect against accidental deletion
4. **Use bucket policies** to restrict access
5. **Enable S3 access logging** for audit trail
6. **Implement lifecycle policies** to auto-delete old backups

## Full Documentation

See [docs/s3_state_backup.md](../../docs/s3_state_backup.md) for complete implementation guide.

## Support

For issues or questions, please file a GitHub issue.
