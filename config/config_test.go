package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
general:
  mode: "manual"
  default_database: "arangodb"
  default_storage: "s3"
  backup_prefix: "test"

database:
  arangodb:
    host: "localhost"
    port: 8529
    username: "root"
    password: "test"
    database: ["_system", "testdb"]

storage:
  s3:
    endpoint: "https://s3.amazonaws.com"
    bucket: "test-bucket"
    access_key: "test-key"
    secret_key: "test-secret"
    region: "us-east-1"
    path: "/tmp/test-backup"
`

	// Write config to temporary file
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	// Test loading config
	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify general config
	if cfg.General.Mode != "manual" {
		t.Errorf("Expected mode 'manual', got '%s'", cfg.General.Mode)
	}

	if cfg.General.BackupPrefix != "test" {
		t.Errorf("Expected backup prefix 'test', got '%s'", cfg.General.BackupPrefix)
	}

	// Verify database config
	if cfg.Database.ArangoDB.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", cfg.Database.ArangoDB.Host)
	}

	if len(cfg.Database.ArangoDB.Database) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(cfg.Database.ArangoDB.Database))
	}

	// Verify storage config
	if cfg.Storage.S3.Bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", cfg.Storage.S3.Bucket)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := &Config{
		General: GeneralConfig{
			Mode:         "manual",
			BackupPrefix: "test",
		},
		Database: DatabaseConfig{
			ArangoDB: ArangoDBConfig{
				Host:     "localhost",
				Port:     8529,
				Username: "root",
				Password: "test",
				Database: []string{"testdb"},
			},
		},
		Storage: StorageConfig{
			S3: S3Config{
				Endpoint:  "https://s3.amazonaws.com",
				Bucket:    "test-bucket",
				AccessKey: "test-key",
				SecretKey: "test-secret",
				Region:    "us-east-1",
			},
		},
	}

	if err := validConfig.Validate(); err != nil {
		t.Errorf("Valid config should not have validation errors: %v", err)
	}

	// Test invalid mode
	invalidModeConfig := *validConfig
	invalidModeConfig.General.Mode = "invalid"
	if err := invalidModeConfig.Validate(); err == nil {
		t.Error("Invalid mode should cause validation error")
	}

	// Test missing backup prefix
	missingPrefixConfig := *validConfig
	missingPrefixConfig.General.BackupPrefix = ""
	if err := missingPrefixConfig.Validate(); err == nil {
		t.Error("Missing backup prefix should cause validation error")
	}
}
