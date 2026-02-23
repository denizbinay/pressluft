package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// DB wraps *sql.DB with application-specific configuration.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at dbPath and runs all
// pending migrations. The parent directory is created if it does not exist.
func Open(dbPath string, logger *slog.Logger) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite concurrency: one writer at a time.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Apply PRAGMAs for performance and correctness.
	if err := applyPragmas(sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}

	if err := runMigrations(sqlDB, logger); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &DB{DB: sqlDB}, nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return nil
}

func runMigrations(db *sql.DB, logger *slog.Logger) error {
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		migrationsFS,
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(context.Background())
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	for _, r := range results {
		logger.Info("migration applied", "path", r.Source.Path, "duration", r.Duration)
	}
	return nil
}
