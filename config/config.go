package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the main configuration structure
type Config struct {
	General  GeneralConfig  `mapstructure:"general"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

// GeneralConfig represents general configuration options
type GeneralConfig struct {
	Mode            string `mapstructure:"mode"`             // auto or manual
	DefaultDatabase string `mapstructure:"default_database"` // default database type
	DefaultStorage  string `mapstructure:"default_storage"`  // default storage type
	BackupPrefix    string `mapstructure:"backup_prefix"`    // backup file prefix
}

// DatabaseConfig represents database configuration options
type DatabaseConfig struct {
	ArangoDB   ArangoDBConfig   `mapstructure:"arangodb"`
	SQLite     SQLiteConfig     `mapstructure:"sqlite"`
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`
}

// ArangoDBConfig represents ArangoDB specific configuration
type ArangoDBConfig struct {
	Host     string   `mapstructure:"host"`
	Port     int      `mapstructure:"port"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	Database []string `mapstructure:"database"` // Array of database names
}

// SQLiteConfig represents SQLite specific configuration
type SQLiteConfig struct {
	Path      string   `mapstructure:"path"`
	Databases []string `mapstructure:"databases"` // Array of database file names
}

// PostgreSQLConfig represents PostgreSQL specific configuration
type PostgreSQLConfig struct {
	Host     string   `mapstructure:"host"`
	Port     int      `mapstructure:"port"`
	Username string   `mapstructure:"username"`
	Password string   `mapstructure:"password"`
	Databases []string `mapstructure:"databases"` // Array of database names
}

// StorageConfig represents storage configuration options
type StorageConfig struct {
	S3 S3Config `mapstructure:"s3"`
	// Future: R2, MinIO, etc.
}

// S3Config represents S3-compatible storage configuration
type S3Config struct {
	Endpoint  string `mapstructure:"endpoint"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Region    string `mapstructure:"region"`
	Path      string `mapstructure:"path"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set default config file name
	if configPath == "" {
		configPath = "config.yml"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yml")

	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Create config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("general.mode", "manual")
	viper.SetDefault("general.default_database", "arangodb")
	viper.SetDefault("general.default_storage", "s3")
	viper.SetDefault("general.backup_prefix", "apito")
	viper.SetDefault("database.arangodb.port", 8529)
	viper.SetDefault("database.postgresql.port", 5432)
	viper.SetDefault("storage.s3.region", "us-east-1")
	viper.SetDefault("storage.s3.path", "/tmp/apito-backup")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate general config
	if c.General.Mode != "auto" && c.General.Mode != "manual" {
		return fmt.Errorf("invalid mode: %s (must be 'auto' or 'manual')", c.General.Mode)
	}

	if c.General.BackupPrefix == "" {
		return fmt.Errorf("backup_prefix is required")
	}

	// Validate database config based on selected engine
	switch c.General.DefaultDatabase {
	case "arangodb":
		if err := c.Database.ArangoDB.Validate(); err != nil {
			return fmt.Errorf("arangodb config validation failed: %v", err)
		}
	case "sqlite":
		if err := c.Database.SQLite.Validate(); err != nil {
			return fmt.Errorf("sqlite config validation failed: %v", err)
		}
	case "postgresql":
		if err := c.Database.PostgreSQL.Validate(); err != nil {
			return fmt.Errorf("postgresql config validation failed: %v", err)
		}
	default:
		return fmt.Errorf("unsupported database engine: %s", c.General.DefaultDatabase)
	}

	// Validate storage config
	if err := c.Storage.S3.Validate(); err != nil {
		return fmt.Errorf("s3 config validation failed: %v", err)
	}

	return nil
}

// Validate validates ArangoDB configuration
func (a *ArangoDBConfig) Validate() error {
	if a.Host == "" {
		return fmt.Errorf("host is required")
	}

	if a.Port <= 0 || a.Port > 65535 {
		return fmt.Errorf("invalid port: %d", a.Port)
	}

	if a.Username == "" {
		return fmt.Errorf("username is required")
	}

	if a.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(a.Database) == 0 {
		return fmt.Errorf("at least one database must be specified")
	}

	return nil
}

// Validate validates SQLite configuration
func (s *SQLiteConfig) Validate() error {
	if s.Path == "" && len(s.Databases) == 0 {
		return fmt.Errorf("either path or databases must be specified")
	}
	return nil
}

// Validate validates PostgreSQL configuration
func (p *PostgreSQLConfig) Validate() error {
	if p.Host == "" {
		return fmt.Errorf("host is required")
	}

	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("invalid port: %d", p.Port)
	}

	if p.Username == "" {
		return fmt.Errorf("username is required")
	}

	if p.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(p.Databases) == 0 {
		return fmt.Errorf("at least one database must be specified")
	}

	return nil
}

// Validate validates S3 configuration
func (s *S3Config) Validate() error {
	if s.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	if s.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	if s.AccessKey == "" {
		return fmt.Errorf("access_key is required")
	}

	if s.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}

	if s.Region == "" {
		return fmt.Errorf("region is required")
	}

	return nil
}

// GetDatabaseConfig returns the database configuration based on mode
func (c *Config) GetDatabaseConfig() interface{} {
	if c.General.Mode == "auto" {
		switch c.General.DefaultDatabase {
		case "arangodb":
			return c.Database.ArangoDB
		default:
			return c.Database.ArangoDB // fallback to arangodb
		}
	}
	return c.Database.ArangoDB
}

// GetStorageConfig returns the storage configuration based on mode
func (c *Config) GetStorageConfig() interface{} {
	if c.General.Mode == "auto" {
		switch c.General.DefaultStorage {
		case "s3":
			return c.Storage.S3
		default:
			return c.Storage.S3 // fallback to s3
		}
	}
	return c.Storage.S3
}
