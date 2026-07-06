package database

import (
	"errors"

	"arangodb-bk-restore/config"
)

// DatabaseEngine represents supported database engines
type DatabaseEngine string

const (
	// EngineArangoDB represents ArangoDB database engine
	EngineArangoDB DatabaseEngine = "arangodb"
	// EngineSQLite represents SQLite database engine
	EngineSQLite DatabaseEngine = "sqlite"
	// EnginePostgreSQL represents PostgreSQL database engine
	EnginePostgreSQL DatabaseEngine = "postgresql"
)

// NewDatabase creates a new database instance based on the configured engine
func NewDatabase(cfg *config.Config, options BackupOptions) (DatabaseInterface, error) {
	engine := DatabaseEngine(cfg.General.DefaultDatabase)

	switch engine {
	case EngineArangoDB:
		return NewArangoDB(cfg.Database.ArangoDB, options), nil
	case EngineSQLite:
		return NewSQLite(cfg.Database.SQLite, options), nil
	case EnginePostgreSQL:
		return NewPostgreSQL(cfg.Database.PostgreSQL, options), nil
	default:
		return nil, errors.New("unsupported database engine: " + cfg.General.DefaultDatabase)
	}
}

// GetDatabasesToBackup returns the list of databases to backup for the configured engine
func GetDatabasesToBackup(cfg *config.Config) ([]string, error) {
	engine := DatabaseEngine(cfg.General.DefaultDatabase)

	switch engine {
	case EngineArangoDB:
		return cfg.Database.ArangoDB.Database, nil
	case EngineSQLite:
		dbs := cfg.Database.SQLite.Databases
		if len(dbs) == 0 && cfg.Database.SQLite.Path != "" {
			dbs = []string{cfg.Database.SQLite.Path}
		}
		return dbs, nil
	case EnginePostgreSQL:
		return cfg.Database.PostgreSQL.Databases, nil
	default:
		return nil, errors.New("unsupported database engine: " + cfg.General.DefaultDatabase)
	}
}

// GetDatabaseType returns the type of database engine as a string
func GetDatabaseType(cfg *config.Config) string {
	return cfg.General.DefaultDatabase
}
