package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"arangodb-bk-restore/config"
	"arangodb-bk-restore/database"
	"arangodb-bk-restore/storage"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Backup databases to storage",
		Long: `Backup one or more databases to S3-compatible storage.
		
Examples:
  # Backup all configured databases
  arangodb-bk-restore backup
  
  # Backup specific database
  arangodb-bk-restore backup --database mydb
  
  # Backup with custom options
  arangodb-bk-restore backup --compress --include-system`,
		RunE: runBackup,
	}

	// Backup options
	backupDatabase      string
	backupCompress      bool
	backupIncludeSystem bool
	backupOverwrite     bool
	backupOutputDir     string
)

func init() {
	rootCmd.AddCommand(backupCmd)

	// Backup flags
	backupCmd.Flags().StringVarP(&backupDatabase, "database", "d", "", "specific database to backup (default: all configured databases)")
	backupCmd.Flags().BoolVarP(&backupCompress, "compress", "c", true, "compress backup files")
	backupCmd.Flags().BoolVarP(&backupIncludeSystem, "include-system", "s", true, "include system collections")
	backupCmd.Flags().BoolVarP(&backupOverwrite, "overwrite", "o", true, "overwrite existing backups")
	backupCmd.Flags().StringVarP(&backupOutputDir, "output-dir", "O", "", "output directory for backups (default: from config)")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Setup logging
	logger := setupLogger()
	logger.Info("Starting backup operation")

	// Get database configuration
	dbConfig := cfg.GetDatabaseConfig()
	arangodbConfig, ok := dbConfig.(config.ArangoDBConfig)
	if !ok {
		return fmt.Errorf("invalid database configuration type")
	}

	// Get storage configuration
	storageConfig := cfg.GetStorageConfig()
	s3Config, ok := storageConfig.(config.S3Config)
	if !ok {
		return fmt.Errorf("invalid storage configuration type")
	}

	// Initialize storage
	storage, err := storage.NewS3Storage(s3Config, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %v", err)
	}

	// Test storage connection
	if err := storage.TestConnection(context.Background()); err != nil {
		return fmt.Errorf("storage connection test failed: %v", err)
	}

	// Initialize database
	db := database.NewArangoDB(arangodbConfig, database.BackupOptions{
		Compress:      backupCompress,
		IncludeSystem: backupIncludeSystem,
		Overwrite:     backupOverwrite,
		OutputDir:     getBackupOutputDir(cfg, backupOutputDir),
	})

	// Test database connection
	if err := db.TestConnection(context.Background()); err != nil {
		return fmt.Errorf("database connection test failed: %v", err)
	}

	// Determine which databases to backup
	databasesToBackup := arangodbConfig.Database
	if backupDatabase != "" {
		// Check if specified database exists in configuration
		found := false
		for _, db := range databasesToBackup {
			if db == backupDatabase {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("database '%s' not found in configuration", backupDatabase)
		}
		databasesToBackup = []string{backupDatabase}
	}

	// Perform backup for each database
	for _, dbName := range databasesToBackup {
		logger.Infof("Backing up database: %s", dbName)

		// Create backup
		backup, err := db.Backup(context.Background(), dbName)
		if err != nil {
			logger.Errorf("Failed to backup database %s: %v", dbName, err)
			continue
		}

		logger.Infof("Database %s backed up successfully", dbName)
		logger.Infof("Backup path: %s", backup.BackupPath)
		logger.Infof("Backup size: %d bytes", backup.Size)
		logger.Infof("Backup checksum: %s", backup.Checksum)

		// Upload to storage
		remoteKey := generateRemoteKey(cfg.General.BackupPrefix, dbName, backup.Timestamp)
		if err := storage.Upload(context.Background(), backup.BackupPath, remoteKey); err != nil {
			logger.Errorf("Failed to upload backup for database %s: %v", dbName, err)
			continue
		}

		logger.Infof("Backup for database %s uploaded to storage successfully", dbName)
		logger.Infof("Remote key: %s", remoteKey)

		// Clean up local backup
		if err := os.RemoveAll(backup.BackupPath); err != nil {
			logger.Warnf("Failed to clean up local backup for database %s: %v", dbName, err)
		}
	}

	logger.Info("Backup operation completed")
	return nil
}

// getBackupOutputDir returns the backup output directory
func getBackupOutputDir(cfg *config.Config, customDir string) string {
	if customDir != "" {
		return customDir
	}

	// Use storage path from config
	storageConfig := cfg.GetStorageConfig()
	if s3Config, ok := storageConfig.(config.S3Config); ok {
		return s3Config.Path
	}

	// Fallback to default
	return "/tmp/apito-backup"
}

// generateRemoteKey generates a remote storage key for the backup
func generateRemoteKey(prefix, databaseName string, timestamp time.Time) string {
	timestampStr := timestamp.Format("20060102_150405")
	return fmt.Sprintf("%s/arangodb/%s_%s.tar.gz", prefix, databaseName, timestampStr)
}

// setupLogger sets up the logger with appropriate level
func setupLogger() *logrus.Logger {
	logger := logrus.New()

	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return logger
}
