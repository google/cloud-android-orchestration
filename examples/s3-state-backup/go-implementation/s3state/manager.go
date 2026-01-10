// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package s3state provides S3-based state backup and restore for CVD instances.
// This code should be added to the android-cuttlefish repository.
package s3state

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Config holds S3 state manager configuration
type Config struct {
	Bucket    string
	Prefix    string
	Region    string
	Enabled   bool
}

// Manager handles backup and restore of CVD state to/from S3
type Manager struct {
	config     Config
	client     *s3.Client
	uploader   *manager.Uploader
	downloader *manager.Downloader
}

// StateFiles are the files that contain CVD persistent state
var StateFiles = []string{
	"userdata.img",
	"sdcard.img",
	"persistent.img",
	"instance_config.json",
}

// NewManager creates a new S3 state manager
func NewManager(cfg Config) (*Manager, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("S3 state backup is not enabled")
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket not specified")
	}

	if cfg.Region == "" {
		cfg.Region = "us-west-2" // Default region
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig)

	return &Manager{
		config:     cfg,
		client:     client,
		uploader:   manager.NewUploader(client),
		downloader: manager.NewDownloader(client),
	}, nil
}

// RestoreState downloads CVD state from S3 to the instance directory
func (m *Manager) RestoreState(ctx context.Context, username, instanceName, instanceDir string) error {
	log.Printf("[S3] Restoring state for instance '%s' (user: %s)", instanceName, username)

	// Ensure instance directory exists
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return fmt.Errorf("failed to create instance directory: %w", err)
	}

	// Check if any state exists in S3
	stateExists, err := m.checkStateExists(ctx, username, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check S3 state: %w", err)
	}

	if !stateExists {
		log.Printf("[S3] No existing state found in S3 for instance '%s' (first-time creation)", instanceName)
		return nil
	}

	log.Printf("[S3] Found existing state in S3, downloading...")

	// Download each state file
	downloadedCount := 0
	for _, filename := range StateFiles {
		key := m.buildS3Key(username, instanceName, filename)
		localPath := filepath.Join(instanceDir, filename)

		// Check if object exists
		exists, err := m.objectExists(ctx, key)
		if err != nil {
			log.Printf("[S3] Warning: Failed to check if %s exists: %v", filename, err)
			continue
		}

		if !exists {
			log.Printf("[S3] Skipping %s (not found in S3)", filename)
			continue
		}

		// Download the file
		if err := m.downloadFile(ctx, key, localPath); err != nil {
			return fmt.Errorf("failed to download %s: %w", filename, err)
		}

		downloadedCount++
		log.Printf("[S3] Downloaded %s", filename)
	}

	if downloadedCount == 0 {
		log.Printf("[S3] Warning: State directory exists but no files were downloaded")
		return nil
	}

	log.Printf("[S3] Successfully restored %d state file(s)", downloadedCount)
	return nil
}

// BackupState uploads CVD state from the instance directory to S3
func (m *Manager) BackupState(ctx context.Context, username, instanceName, instanceDir string) error {
	log.Printf("[S3] Backing up state for instance '%s' (user: %s)", instanceName, username)

	// Verify instance directory exists
	if _, err := os.Stat(instanceDir); os.IsNotExist(err) {
		log.Printf("[S3] Instance directory does not exist, nothing to backup")
		return nil
	}

	uploadedCount := 0
	for _, filename := range StateFiles {
		localPath := filepath.Join(instanceDir, filename)

		// Check if file exists locally
		info, err := os.Stat(localPath)
		if os.IsNotExist(err) {
			log.Printf("[S3] Skipping %s (not found locally)", filename)
			continue
		}
		if err != nil {
			log.Printf("[S3] Warning: Failed to stat %s: %v", filename, err)
			continue
		}

		key := m.buildS3Key(username, instanceName, filename)

		// Upload the file
		if err := m.uploadFile(ctx, localPath, key); err != nil {
			return fmt.Errorf("failed to upload %s: %w", filename, err)
		}

		uploadedCount++
		log.Printf("[S3] Uploaded %s (%d bytes)", filename, info.Size())
	}

	if uploadedCount == 0 {
		log.Printf("[S3] Warning: No files were backed up")
		return nil
	}

	// Tag the backup with metadata
	if err := m.tagBackup(ctx, username, instanceName); err != nil {
		log.Printf("[S3] Warning: Failed to tag backup: %v", err)
	}

	log.Printf("[S3] Successfully backed up %d state file(s)", uploadedCount)
	return nil
}

// DeleteState removes CVD state from S3
func (m *Manager) DeleteState(ctx context.Context, username, instanceName string) error {
	log.Printf("[S3] Deleting state for instance '%s' (user: %s)", instanceName, username)

	deletedCount := 0
	for _, filename := range StateFiles {
		key := m.buildS3Key(username, instanceName, filename)

		exists, err := m.objectExists(ctx, key)
		if err != nil {
			log.Printf("[S3] Warning: Failed to check if %s exists: %v", filename, err)
			continue
		}

		if !exists {
			continue
		}

		_, err = m.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(m.config.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("[S3] Warning: Failed to delete %s: %v", filename, err)
			continue
		}

		deletedCount++
		log.Printf("[S3] Deleted %s", filename)
	}

	log.Printf("[S3] Deleted %d state file(s)", deletedCount)
	return nil
}

// buildS3Key constructs the S3 key for a state file
func (m *Manager) buildS3Key(username, instanceName, filename string) string {
	if m.config.Prefix == "" {
		return fmt.Sprintf("%s/%s/%s", username, instanceName, filename)
	}
	return fmt.Sprintf("%s/%s/%s/%s", m.config.Prefix, username, instanceName, filename)
}

// checkStateExists checks if any state files exist in S3
func (m *Manager) checkStateExists(ctx context.Context, username, instanceName string) (bool, error) {
	prefix := m.buildS3Key(username, instanceName, "")

	result, err := m.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(m.config.Bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return false, err
	}

	return result.KeyCount != nil && *result.KeyCount > 0, nil
}

// objectExists checks if a specific S3 object exists
func (m *Manager) objectExists(ctx context.Context, key string) (bool, error) {
	_, err := m.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		var noSuchKey *types.NoSuchKey
		if aws.IsA[*types.NotFound](err) || aws.IsA[*types.NoSuchKey](err) {
			return false, nil
		}
		// Check using error type assertion as fallback
		if _, ok := err.(*types.NotFound); ok {
			return false, nil
		}
		if _, ok := err.(*types.NoSuchKey); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// downloadFile downloads a file from S3 to local path
func (m *Manager) downloadFile(ctx context.Context, key, localPath string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	_, err = m.downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		os.Remove(localPath) // Clean up partial download
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// uploadFile uploads a local file to S3
func (m *Manager) uploadFile(ctx context.Context, localPath, key string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	_, err = m.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	return nil
}

// tagBackup adds metadata tags to the backup
func (m *Manager) tagBackup(ctx context.Context, username, instanceName string) error {
	// Tag the userdata.img file as a representative of the backup
	key := m.buildS3Key(username, instanceName, "userdata.img")

	exists, err := m.objectExists(ctx, key)
	if err != nil || !exists {
		return fmt.Errorf("userdata.img not found for tagging")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	_, err = m.client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("LastBackup"), Value: aws.String(timestamp)},
				{Key: aws.String("Instance"), Value: aws.String(instanceName)},
				{Key: aws.String("Username"), Value: aws.String(username)},
			},
		},
	})

	return err
}

// GetBackupInfo retrieves metadata about a backup
func (m *Manager) GetBackupInfo(ctx context.Context, username, instanceName string) (*BackupInfo, error) {
	key := m.buildS3Key(username, instanceName, "userdata.img")

	exists, err := m.objectExists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil // No backup exists
	}

	// Get object metadata
	head, err := m.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	// Get tags
	tags, err := m.client.GetObjectTagging(ctx, &s3.GetObjectTaggingInput{
		Bucket: aws.String(m.config.Bucket),
		Key:    aws.String(key),
	})

	info := &BackupInfo{
		InstanceName:  instanceName,
		Username:      username,
		LastModified:  *head.LastModified,
		Size:          *head.ContentLength,
		Tags:          make(map[string]string),
	}

	if err == nil && tags != nil {
		for _, tag := range tags.TagSet {
			info.Tags[*tag.Key] = *tag.Value
		}
	}

	return info, nil
}

// BackupInfo contains metadata about a backup
type BackupInfo struct {
	InstanceName string
	Username     string
	LastModified time.Time
	Size         int64
	Tags         map[string]string
}
