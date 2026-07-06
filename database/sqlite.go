package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"arangodb-bk-restore/config"
)

// SQLite implements the DatabaseInterface for SQLite
// SQLite databases are file-based, so we backup/restore by copying files
type SQLite struct {
	config  config.SQLiteConfig
	options BackupOptions
}

// NewSQLite creates a new SQLite instance
func NewSQLite(cfg config.SQLiteConfig, options BackupOptions) *SQLite {
	return &SQLite{
		config:  cfg,
		options: options,
	}
}

// Backup creates a backup of the specified SQLite database file
func (s *SQLite) Backup(ctx context.Context, databaseName string) (*DatabaseBackup, error) {
	// Resolve the actual database file path
	dbPath := s.resolveDatabasePath(databaseName)

	// Check if database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("sqlite database file not found: %s", dbPath)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(s.options.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_%s.db", databaseName, timestamp)
	backupPath := filepath.Join(s.options.OutputDir, backupName)

	// Copy the database file to backup location
	if err := s.copyFile(dbPath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to copy database file: %v", err)
	}

	// Get backup information
	backupInfo, err := s.GetBackupInfo(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup info: %v", err)
	}

	return backupInfo, nil
}

// Restore restores a database from the specified backup
func (s *SQLite) Restore(ctx context.Context, backupPath string, databaseName string) error {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Resolve the target database file path
	targetPath := s.resolveDatabasePath(databaseName)

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// If target exists and overwrite not set, error
	if _, err := os.Stat(targetPath); err == nil && !s.options.Overwrite {
		return fmt.Errorf("target database file already exists (use --overwrite to replace): %s", targetPath)
	}

	// Copy backup file to target location
	if err := s.copyFile(backupPath, targetPath); err != nil {
		return fmt.Errorf("failed to restore database file: %v", err)
	}

	return nil
}

// ListDatabases returns a list of available database file names
func (s *SQLite) ListDatabases(ctx context.Context) ([]string, error) {
	dbs := s.config.Databases
	if len(dbs) == 0 && s.config.Path != "" {
		return []string{s.config.Path}, nil
	}
	return dbs, nil
}

// TestConnection tests the database connection by opening the SQLite file
func (s *SQLite) TestConnection(ctx context.Context) error {
	paths := s.config.Databases
	if len(paths) == 0 && s.config.Path != "" {
		paths = []string{s.config.Path}
	}

	for _, dbPath := range paths {
		resolvedPath := s.resolveDatabasePath(dbPath)
		file, err := os.Open(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to open sqlite database '%s': %v", dbPath, err)
		}
		file.Close()
	}

	return nil
}

// GetBackupInfo returns information about a backup file
func (s *SQLite) GetBackupInfo(backupPath string) (*DatabaseBackup, error) {
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %v", err)
	}

	// Calculate checksum
	checksum, err := s.calculateFileChecksum(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %v", err)
	}

	// Extract database name and timestamp from filename
	databaseName, timestamp, err := s.parseBackupFilename(filepath.Base(backupPath))
	if err != nil {
		// If parsing fails, use the full filename as database name and file modification time
		databaseName = strings.TrimSuffix(filepath.Base(backupPath), filepath.Ext(backupPath))
		timestamp = info.ModTime()
	}

	return &DatabaseBackup{
		DatabaseName: databaseName,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         info.Size(),
		Checksum:     checksum,
	}, nil
}

// SetOptions sets backup options
func (s *SQLite) SetOptions(options BackupOptions) {
	s.options = options
}

// GetOptions returns current backup options
func (s *SQLite) GetOptions() BackupOptions {
	return s.options
}

// resolveDatabasePath resolves a database name to an absolute file path.
// If the path is already absolute, it is returned as-is.
// If config.Path is set and the database name is relative, it is joined with config.Path.
func (s *SQLite) resolveDatabasePath(databaseName string) string {
	if filepath.IsAbs(databaseName) {
		return databaseName
	}
	if s.config.Path != "" {
		return filepath.Join(s.config.Path, databaseName)
	}
	// Return as-is if no base path is configured
	return databaseName
}

// copyFile copies a file from src to dst
func (s *SQLite) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Preserve file permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

// calculateFileChecksum calculates MD5 checksum of a file
func (s *SQLite) calculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// parseBackupFilename extracts database name and timestamp from SQLite backup filename
// Expected format: databaseName_20060102_150405.db
func (s *SQLite) parseBackupFilename(filename string) (string, time.Time, error) {
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Find the last occurrence of _YYYYMMDD_HHMMSS pattern
	parts := strings.Split(baseName, "_")
	if len(parts) < 3 {
		return "", time.Time{}, fmt.Errorf("invalid backup filename format: %s", filename)
	}

	// Timestamp is the last two parts joined: YYYYMMDD + HHMMSS
	timestampStr := parts[len(parts)-2] + "_" + parts[len(parts)-1]
	timestamp, err := time.Parse("20060102_150405", timestampStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	databaseName := strings.Join(parts[:len(parts)-2], "_")
	return databaseName, timestamp, nil
}
