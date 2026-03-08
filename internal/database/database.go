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
	"strings"

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
	if err := reconcileLegacySchema(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("reconcile legacy schema: %w", err)
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

func reconcileLegacySchema(db *sql.DB) error {
	if err := ensureJobsColumns(db, []string{
		"payload TEXT",
		"command_id TEXT",
		"started_at TEXT",
		"finished_at TEXT",
		"timeout_at TEXT",
	}); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_jobs_command_id ON jobs(command_id) WHERE command_id IS NOT NULL`); err != nil {
		return fmt.Errorf("ensure jobs command_id index: %w", err)
	}
	if err := ensureProvidersColumns(db, []string{
		"api_token_encrypted TEXT",
		"api_token_key_id TEXT",
		"api_token_version INTEGER NOT NULL DEFAULT 0",
	}); err != nil {
		return err
	}
	if err := ensureServersColumns(db, []string{
		"setup_state TEXT NOT NULL DEFAULT 'not_started'",
		"setup_last_error TEXT",
	}); err != nil {
		return err
	}
	return nil
}

func ensureJobsColumns(db *sql.DB, definitions []string) error {
	rows, err := db.Query(`PRAGMA table_info(jobs)`)
	if err != nil {
		return fmt.Errorf("inspect jobs schema: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan jobs schema: %w", err)
		}
		existing[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate jobs schema: %w", err)
	}

	for _, definition := range definitions {
		parts := strings.Fields(definition)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToLower(parts[0])
		if _, ok := existing[name]; ok {
			continue
		}
		if _, err := db.Exec(`ALTER TABLE jobs ADD COLUMN ` + definition); err != nil {
			return fmt.Errorf("add jobs.%s: %w", name, err)
		}
		existing[name] = struct{}{}
	}
	return nil
}

func ensureProvidersColumns(db *sql.DB, definitions []string) error {
	rows, err := db.Query(`PRAGMA table_info(providers)`)
	if err != nil {
		return fmt.Errorf("inspect providers schema: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	foundTable := false
	for rows.Next() {
		foundTable = true
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan providers schema: %w", err)
		}
		existing[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate providers schema: %w", err)
	}
	if !foundTable {
		return nil
	}

	for _, definition := range definitions {
		parts := strings.Fields(definition)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToLower(parts[0])
		if _, ok := existing[name]; ok {
			continue
		}
		if _, err := db.Exec(`ALTER TABLE providers ADD COLUMN ` + definition); err != nil {
			return fmt.Errorf("add providers.%s: %w", name, err)
		}
	}
	return nil
}

func ensureServersColumns(db *sql.DB, definitions []string) error {
	rows, err := db.Query(`PRAGMA table_info(servers)`)
	if err != nil {
		return fmt.Errorf("inspect servers schema: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	foundTable := false
	for rows.Next() {
		foundTable = true
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan servers schema: %w", err)
		}
		existing[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate servers schema: %w", err)
	}
	if !foundTable {
		return nil
	}

	for _, definition := range definitions {
		parts := strings.Fields(definition)
		if len(parts) == 0 {
			continue
		}
		name := strings.ToLower(parts[0])
		if _, ok := existing[name]; ok {
			continue
		}
		if _, err := db.Exec(`ALTER TABLE servers ADD COLUMN ` + definition); err != nil {
			return fmt.Errorf("add servers.%s: %w", name, err)
		}
	}
	return nil
}
