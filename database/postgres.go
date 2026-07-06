package database

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"arangodb-bk-restore/config"
)

// PostgreSQL implements DatabaseInterface for PostgreSQL using docker-run pg_dump/pg_restore
type PostgreSQL struct {
	config  config.PostgreSQLConfig
	options BackupOptions
}

// NewPostgreSQL creates a new PostgreSQL instance
func NewPostgreSQL(cfg config.PostgreSQLConfig, options BackupOptions) *PostgreSQL {
	return &PostgreSQL{
		config:  cfg,
		options: options,
	}
}

// Backup creates a backup of the specified PostgreSQL database using pg_dump via Docker
func (p *PostgreSQL) Backup(ctx context.Context, databaseName string) (*DatabaseBackup, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(p.options.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s_%s.dump", databaseName, timestamp)
	backupPath := filepath.Join(p.options.OutputDir, backupName)

	// Build pg_dump command via Docker
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/dump", p.options.OutputDir),
		"postgres:16",
		"pg_dump",
		"--host", p.config.Host,
		"--port", fmt.Sprintf("%d", p.config.Port),
		"--username", p.config.Username,
		"--dbname", databaseName,
		"--format", "custom",
		"--file", fmt.Sprintf("/dump/%s", backupName),
	}

	if p.options.Compress {
		args = append(args, "--compress", "6")
	}

	// Build command with environment for password
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.config.Password))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pg_dump failed: %v", err)
	}

	// Get backup information
	backupInfo, err := p.GetBackupInfo(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup info: %v", err)
	}

	return backupInfo, nil
}

// Restore restores a database from a pg_dump custom-format backup using pg_restore via Docker
func (p *PostgreSQL) Restore(ctx context.Context, backupPath string, databaseName string) error {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Determine the local directory and the inside-docker path
	backupDir := filepath.Dir(backupPath)
	backupFileName := filepath.Base(backupPath)

	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/dump", backupDir),
		"postgres:16",
		"pg_restore",
		"--host", p.config.Host,
		"--port", fmt.Sprintf("%d", p.config.Port),
		"--username", p.config.Username,
		"--dbname", databaseName,
		"--no-owner",
		"--no-acl",
	}

	if p.options.Overwrite {
		args = append(args, "--clean", "--if-exists", "--create")
	}

	args = append(args, fmt.Sprintf("/dump/%s", backupFileName))

	// Build command with environment for password
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.config.Password))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %v", err)
	}

	return nil
}

// ListDatabases returns configured database names
func (p *PostgreSQL) ListDatabases(ctx context.Context) ([]string, error) {
	return p.config.Databases, nil
}

// TestConnection tests the database connection using pg_isready via Docker
func (p *PostgreSQL) TestConnection(ctx context.Context) error {
	args := []string{
		"run", "--rm",
		"postgres:16",
		"pg_isready",
		"--host", p.config.Host,
		"--port", fmt.Sprintf("%d", p.config.Port),
		"--username", p.config.Username,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connection test failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetBackupInfo returns information about a backup file
func (p *PostgreSQL) GetBackupInfo(backupPath string) (*DatabaseBackup, error) {
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %v", err)
	}

	// For PostgreSQL we store single dump files (not directories like ArangoDB).
	// Extract database name and timestamp from filename.
	databaseName, timestamp, err := p.parseBackupFilename(filepath.Base(backupPath))
	if err != nil {
		databaseName = strings.TrimSuffix(filepath.Base(backupPath), filepath.Ext(backupPath))
		timestamp = info.ModTime()
	}

	return &DatabaseBackup{
		DatabaseName: databaseName,
		BackupPath:   backupPath,
		Timestamp:    timestamp,
		Size:         info.Size(),
		Checksum:     "", // could add checksum computation if needed
	}, nil
}

// SetOptions sets backup options
func (p *PostgreSQL) SetOptions(options BackupOptions) {
	p.options = options
}

// GetOptions returns current backup options
func (p *PostgreSQL) GetOptions() BackupOptions {
	return p.options
}

// parseBackupFilename extracts database name and timestamp from PostgreSQL backup filename
// Expected format: databaseName_20060102_150405.dump
func (p *PostgreSQL) parseBackupFilename(filename string) (string, time.Time, error) {
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(baseName, "_")
	if len(parts) < 3 {
		return "", time.Time{}, fmt.Errorf("invalid backup filename format: %s", filename)
	}

	timestampStr := parts[len(parts)-2] + "_" + parts[len(parts)-1]
	timestamp, err := time.Parse("20060102_150405", timestampStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to parse timestamp '%s': %v", timestampStr, err)
	}

	databaseName := strings.Join(parts[:len(parts)-2], "_")
	return databaseName, timestamp, nil
}
