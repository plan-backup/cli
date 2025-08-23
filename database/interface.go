package database

import (
	"context"
	"time"
)

// DatabaseBackup represents a database backup operation
type DatabaseBackup struct {
	DatabaseName string
	BackupPath   string
	Timestamp    time.Time
	Size         int64
	Checksum     string
}

// DatabaseRestore represents a database restore operation
type DatabaseRestore struct {
	DatabaseName string
	BackupPath   string
	Timestamp    time.Time
}

// DatabaseInterface defines the interface for database operations
type DatabaseInterface interface {
	// Backup creates a backup of the specified database
	Backup(ctx context.Context, databaseName string) (*DatabaseBackup, error)

	// Restore restores a database from the specified backup
	Restore(ctx context.Context, backupPath string, databaseName string) error

	// ListDatabases returns a list of available databases
	ListDatabases(ctx context.Context) ([]string, error)

	// TestConnection tests the database connection
	TestConnection(ctx context.Context) error

	// GetBackupInfo returns information about a backup file
	GetBackupInfo(backupPath string) (*DatabaseBackup, error)
}

// BackupOptions represents options for backup operations
type BackupOptions struct {
	Compress      bool
	IncludeSystem bool
	Overwrite     bool
	OutputDir     string
	InputDir      string // For restore operations
}
