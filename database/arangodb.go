package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"arangodb-bk-restore/config"
)

// ArangoDB implements the DatabaseInterface for ArangoDB
type ArangoDB struct {
	config  config.ArangoDBConfig
	options BackupOptions
}

// NewArangoDB creates a new ArangoDB instance
func NewArangoDB(cfg config.ArangoDBConfig, options BackupOptions) *ArangoDB {
	return &ArangoDB{
		config:  cfg,
		options: options,
	}
}

// Backup creates a backup of the specified database using arangodump
func (a *ArangoDB) Backup(ctx context.Context, databaseName string) (*DatabaseBackup, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(a.options.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(a.options.OutputDir, fmt.Sprintf("%s_%s", databaseName, timestamp))

	// Build arangodump command
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/dump", a.options.OutputDir),
		"arangodb/arangodb:3.11.8",
		"arangodump",
		"--server.endpoint", fmt.Sprintf("tcp://%s:%d", a.config.Host, a.config.Port),
		"--server.username", a.config.Username,
		"--server.password", a.config.Password,
		"--server.database", databaseName,
		"--output-directory", fmt.Sprintf("/dump/%s_%s", databaseName, timestamp),
	}

	if a.options.IncludeSystem {
		args = append(args, "--include-system-collections", "true")
	}

	if a.options.Overwrite {
		args = append(args, "--overwrite", "true")
	}

	if a.options.Compress {
		args = append(args, "--compress-output")
	}

	// Execute arangodump command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("arangodump failed: %v", err)
	}

	// Get backup information
	backupInfo, err := a.GetBackupInfo(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup info: %v", err)
	}

	return backupInfo, nil
}

// Restore restores a database from the specified backup using arangorestore
func (a *ArangoDB) Restore(ctx context.Context, backupPath string, databaseName string) error {
	// Check if backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory not found: %s", backupPath)
	}

	// Build arangorestore command
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/dump", filepath.Dir(backupPath)),
		"arangodb/arangodb:3.11.8",
		"arangorestore",
		"--server.endpoint", fmt.Sprintf("tcp://%s:%d", a.config.Host, a.config.Port),
		"--server.username", a.config.Username,
		"--server.password", a.config.Password,
		"--server.database", databaseName,
		"--input-directory", fmt.Sprintf("/dump/%s", filepath.Base(backupPath)),
	}

	if a.options.IncludeSystem {
		args = append(args, "--include-system-collections", "true")
	}

	if a.options.Overwrite {
		args = append(args, "--overwrite", "true")
	}

	// Execute arangorestore command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("arangorestore failed: %v", err)
	}

	return nil
}

// ListDatabases returns a list of available databases
func (a *ArangoDB) ListDatabases(ctx context.Context) ([]string, error) {
	// For now, return the configured databases
	// In the future, this could query ArangoDB directly
	return a.config.Database, nil
}

// TestConnection tests the database connection
func (a *ArangoDB) TestConnection(ctx context.Context) error {
	// Build a simple test command
	args := []string{
		"run", "--rm",
		"arangodb/arangodb:3.11.8",
		"arangosh",
		"--server.endpoint", fmt.Sprintf("tcp://%s:%d", a.config.Host, a.config.Port),
		"--server.username", a.config.Username,
		"--server.password", a.config.Password,
		"--server.database", "_system",
		"--javascript.execute-string", "db._name()",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connection test failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetBackupInfo returns information about a backup file
func (a *ArangoDB) GetBackupInfo(backupPath string) (*DatabaseBackup, error) {
	// Get file info
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup path: %v", err)
	}

	// Calculate directory size
	size, err := a.calculateDirectorySize(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate directory size: %v", err)
	}

	// Calculate checksum
	checksum, err := a.calculateDirectoryChecksum(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %v", err)
	}

	// Extract database name and timestamp from path
	baseName := filepath.Base(backupPath)
	parts := strings.Split(baseName, "_")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid backup path format: %s", backupPath)
	}

	databaseName := parts[0]
	timestampStr := strings.Join(parts[1:], "_")
	timestamp, err := time.Parse("20060102_150405", timestampStr)
	if err != nil {
		// If timestamp parsing fails, use file modification time
		timestamp = info.ModTime()
	}

	return &DatabaseBackup{
		DatabaseName: databaseName,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         size,
		Checksum:     checksum,
	}, nil
}

// calculateDirectorySize calculates the total size of a directory
func (a *ArangoDB) calculateDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// calculateDirectoryChecksum calculates MD5 checksum of directory contents
func (a *ArangoDB) calculateDirectoryChecksum(path string) (string, error) {
	hash := md5.New()
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Add file path and size to hash
			hash.Write([]byte(info.Name()))
			hash.Write([]byte(strconv.FormatInt(info.Size(), 10)))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SetOptions sets backup options
func (a *ArangoDB) SetOptions(options BackupOptions) {
	a.options = options
}

// GetOptions returns current backup options
func (a *ArangoDB) GetOptions() BackupOptions {
	return a.options
}
