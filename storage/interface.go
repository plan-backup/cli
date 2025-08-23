package storage

import (
	"context"
	"time"
)

// BackupMetadata represents metadata about a stored backup
type BackupMetadata struct {
	Key          string
	Size         int64
	LastModified time.Time
	Checksum     string
	DatabaseName string
	Timestamp    time.Time
}

// StorageInterface defines the interface for storage operations
type StorageInterface interface {
	// Upload uploads a backup to storage
	Upload(ctx context.Context, localPath, remoteKey string) error

	// Download downloads a backup from storage
	Download(ctx context.Context, remoteKey, localPath string) error

	// ListBackups lists available backups with optional prefix filtering
	ListBackups(ctx context.Context, prefix string) ([]*BackupMetadata, error)

	// DeleteBackup deletes a backup from storage
	DeleteBackup(ctx context.Context, remoteKey string) error

	// GetBackupInfo returns information about a backup
	GetBackupInfo(ctx context.Context, remoteKey string) (*BackupMetadata, error)

	// TestConnection tests the storage connection
	TestConnection(ctx context.Context) error
}

// UploadOptions represents options for upload operations
type UploadOptions struct {
	Compress      bool
	Encrypt       bool
	RetentionDays int
}

// DownloadOptions represents options for download operations
type DownloadOptions struct {
	VerifyChecksum bool
	ResumePartial  bool
}
