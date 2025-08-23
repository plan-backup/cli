package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"arangodb-bk-restore/config"
	"arangodb-bk-restore/database"
	"arangodb-bk-restore/storage"

	"github.com/spf13/cobra"
)

var (
	restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restore databases from storage",
		Long: `Restore databases from stored backups in S3-compatible storage.
		
This command will:
1. List available backups from storage
2. Allow you to select which backup to restore
3. Ask for multiple confirmations before proceeding
4. Download and restore the selected backup

Examples:
  # Interactive restore
  arangodb-bk-restore restore
  
  # Restore specific backup by key
  arangodb-bk-restore restore --backup-key "apito/mydb_20241201_143022.tar.gz"`,
		RunE: runRestore,
	}

	// Restore options
	restoreBackupKey     string
	restoreDatabase      string
	restoreOverwrite     bool
	restoreIncludeSystem bool
	restoreInputDir      string
)

func init() {
	rootCmd.AddCommand(restoreCmd)

	// Restore flags
	restoreCmd.Flags().StringVarP(&restoreBackupKey, "backup-key", "k", "", "specific backup key to restore")
	restoreCmd.Flags().StringVarP(&restoreDatabase, "database", "d", "", "target database name for restore")
	restoreCmd.Flags().BoolVarP(&restoreOverwrite, "overwrite", "o", false, "overwrite existing database")
	restoreCmd.Flags().BoolVarP(&restoreIncludeSystem, "include-system", "s", true, "include system collections")
	restoreCmd.Flags().StringVarP(&restoreInputDir, "input-dir", "i", "", "input directory for restore (default: from config)")
}

func runRestore(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Setup logging
	logger := setupLogger()
	logger.Info("Starting restore operation")

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
	storageClient, err := storage.NewS3Storage(s3Config, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %v", err)
	}

	// Test storage connection
	if err := storageClient.TestConnection(context.Background()); err != nil {
		return fmt.Errorf("storage connection test failed: %v", err)
	}

	// Initialize database
	db := database.NewArangoDB(arangodbConfig, database.BackupOptions{
		IncludeSystem: restoreIncludeSystem,
		Overwrite:     restoreOverwrite,
		InputDir:      getRestoreInputDir(cfg, restoreInputDir),
	})

	// Test database connection
	if err := db.TestConnection(context.Background()); err != nil {
		return fmt.Errorf("database connection test failed: %v", err)
	}

	var selectedBackup *storage.BackupMetadata

	if restoreBackupKey != "" {
		// Use specified backup key
		backupInfo, err := storageClient.GetBackupInfo(context.Background(), restoreBackupKey)
		if err != nil {
			return fmt.Errorf("failed to get backup info: %v", err)
		}
		selectedBackup = backupInfo
	} else {
		// Interactive backup selection
		selectedBackup, err = selectBackupInteractively(storageClient, cfg.General.BackupPrefix)
		if err != nil {
			return fmt.Errorf("backup selection failed: %v", err)
		}
	}

	// Multiple confirmation prompts
	if err := confirmRestore(selectedBackup, arangodbConfig); err != nil {
		return fmt.Errorf("restore cancelled: %v", err)
	}

	// Determine target database name
	targetDatabase := restoreDatabase
	if targetDatabase == "" {
		targetDatabase = selectedBackup.DatabaseName
	}

	// Display target database connection information
	displayTargetDatabaseInfo(arangodbConfig, targetDatabase)

	// Final confirmation
	if err := finalConfirmation(selectedBackup, targetDatabase); err != nil {
		return fmt.Errorf("restore cancelled: %v", err)
	}

	// Download backup
	restoreDir := getRestoreInputDir(cfg, restoreInputDir)
	archivePath := filepath.Join(restoreDir, filepath.Base(selectedBackup.Key))
	logger.Infof("Downloading backup to: %s", archivePath)

	if err := storageClient.Download(context.Background(), selectedBackup.Key, archivePath); err != nil {
		return fmt.Errorf("failed to download backup: %v", err)
	}

	// Extract the archive
	extractDir := strings.TrimSuffix(archivePath, ".tar.gz")
	logger.Infof("Extracting backup to: %s", extractDir)

	if err := storageClient.ExtractArchive(archivePath, extractDir); err != nil {
		// Clean up downloaded file
		os.Remove(archivePath)
		return fmt.Errorf("failed to extract backup: %v", err)
	}

	// Restore database
	logger.Infof("Restoring database: %s", targetDatabase)
	if err := db.Restore(context.Background(), extractDir, targetDatabase); err != nil {
		// Clean up downloaded files
		os.Remove(archivePath)
		os.RemoveAll(extractDir)
		return fmt.Errorf("failed to restore database: %v", err)
	}

	logger.Infof("Database %s restored successfully from backup: %s", targetDatabase, selectedBackup.Key)

	// Clean up downloaded files
	if err := os.Remove(archivePath); err != nil {
		logger.Warnf("Failed to clean up archive file: %v", err)
	}
	if err := os.RemoveAll(extractDir); err != nil {
		logger.Warnf("Failed to clean up extracted directory: %v", err)
	}

	return nil
}

// selectBackupInteractively allows user to select a backup interactively
func selectBackupInteractively(storage storage.StorageInterface, prefix string) (*storage.BackupMetadata, error) {
	fmt.Println("Available backups:")
	fmt.Println("==================")

	// List backups
	backups, err := storage.ListBackups(context.Background(), prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %v", err)
	}

	if len(backups) == 0 {
		return nil, fmt.Errorf("no backups found with prefix: %s", prefix)
	}

	// Sort backups by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	// Display backups
	for i, backup := range backups {
		fmt.Printf("%d. Database: %s\n", i+1, backup.DatabaseName)
		fmt.Printf("   Timestamp: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Size: %s\n", formatBytes(backup.Size))
		fmt.Printf("   Remote Key: %s\n", backup.Key)
		fmt.Println()
	}

	// Get user selection
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nSelect backup number: ")

	selection, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %v", err)
	}

	selection = strings.TrimSpace(selection)
	index, err := strconv.Atoi(selection)
	if err != nil || index < 1 || index > len(backups) {
		return nil, fmt.Errorf("invalid selection: %s", selection)
	}

	return backups[index-1], nil
}

// confirmRestore asks for multiple confirmations
func confirmRestore(backup *storage.BackupMetadata, arangodbConfig config.ArangoDBConfig) error {
	fmt.Printf("\nRestore Details:\n")
	fmt.Printf("================\n")
	fmt.Printf("Database: %s\n", backup.DatabaseName)
	fmt.Printf("Timestamp: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Size: %s\n", formatBytes(backup.Size))
	fmt.Printf("Remote Key: %s\n", backup.Key)

	fmt.Printf("\nTarget Database Connection:\n")
	fmt.Printf("===========================\n")
	fmt.Printf("Host: %s\n", arangodbConfig.Host)
	fmt.Printf("Port: %d\n", arangodbConfig.Port)
	fmt.Printf("Username: %s\n", arangodbConfig.Username)
	fmt.Printf("Password: %s\n", strings.Repeat("*", len(arangodbConfig.Password)))

	reader := bufio.NewReader(os.Stdin)

	// First confirmation
	fmt.Print("\n⚠️  WARNING: This will overwrite the target database. Are you sure? [y/N]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return fmt.Errorf("restore cancelled by user")
	}

	// Second confirmation
	fmt.Print("⚠️  FINAL WARNING: This operation cannot be undone. Type 'RESTORE' to confirm: ")
	response, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	if strings.TrimSpace(response) != "RESTORE" {
		return fmt.Errorf("restore cancelled by user")
	}

	return nil
}

// finalConfirmation asks for final confirmation
func finalConfirmation(backup *storage.BackupMetadata, targetDatabase string) error {
	fmt.Printf("\nFinal Confirmation:\n")
	fmt.Printf("===================\n")
	fmt.Printf("Source Backup: %s\n", backup.Key)
	fmt.Printf("Target Database: %s\n", targetDatabase)
	fmt.Printf("Timestamp: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nProceed with restore? [y/N]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return fmt.Errorf("restore cancelled by user")
	}

	return nil
}

// getRestoreInputDir returns the restore input directory
func getRestoreInputDir(cfg *config.Config, customDir string) string {
	if customDir != "" {
		return customDir
	}

	// Use storage path from config
	storageConfig := cfg.GetStorageConfig()
	if s3Config, ok := storageConfig.(config.S3Config); ok {
		return s3Config.Path
	}

	// Fallback to default
	return "/tmp/apito-restore"
}

// displayTargetDatabaseInfo displays the target database connection information
func displayTargetDatabaseInfo(arangodbConfig config.ArangoDBConfig, targetDatabase string) {
	fmt.Printf("\nTarget Database Information:\n")
	fmt.Printf("============================\n")
	fmt.Printf("Database Name: %s\n", targetDatabase)
	fmt.Printf("Host: %s\n", arangodbConfig.Host)
	fmt.Printf("Port: %d\n", arangodbConfig.Port)
	fmt.Printf("Username: %s\n", arangodbConfig.Username)
	fmt.Printf("Password: %s\n", strings.Repeat("*", len(arangodbConfig.Password)))
	fmt.Printf("============================\n")
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
