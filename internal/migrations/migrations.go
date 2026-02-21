package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func Run(action, migrationsDir, dbPath string) error {
	action = strings.TrimSpace(action)
	if action != "up" && action != "down" {
		return fmt.Errorf("invalid action %q (expected up or down)", action)
	}
	migrationsDir = strings.TrimSpace(migrationsDir)
	if migrationsDir == "" {
		return fmt.Errorf("migrations dir required")
	}
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return fmt.Errorf("db path required")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite db: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping sqlite db: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	suffix := "." + action + ".sql"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, suffix) {
			files = append(files, filepath.Join(migrationsDir, name))
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files ending with %s", suffix)
	}

	if action == "down" {
		if err := ensureMigrationTable(db); err != nil {
			return err
		}
	}

	sort.Strings(files)
	if action == "down" {
		reverse(files)
	}

	for _, file := range files {
		if action == "down" {
			applied, err := migrationApplied(db, filepath.Base(file))
			if err != nil {
				return err
			}
			if !applied {
				continue
			}
		}

		if err := applyFile(db, file); err != nil {
			return fmt.Errorf("apply %s: %w", filepath.Base(file), err)
		}

		if action == "up" {
			if err := markMigrationApplied(db, filepath.Base(file)); err != nil {
				return err
			}
		} else {
			if err := markMigrationReverted(db, filepath.Base(file)); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyFile(db *sql.DB, sqlFile string) error {
	content, err := os.ReadFile(sqlFile)
	if err != nil {
		return fmt.Errorf("read sql file: %w", err)
	}

	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("exec sql file: %w", err)
	}

	return nil
}

func ensureMigrationTable(db *sql.DB) error {
	if _, err := db.Exec(`
 		CREATE TABLE IF NOT EXISTS schema_migrations (
 			version TEXT PRIMARY KEY,
 			applied_at TEXT NOT NULL
 		)
 	`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func migrationApplied(db *sql.DB, name string) (bool, error) {
	if err := ensureMigrationTable(db); err != nil {
		return false, err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, name).Scan(&count); err != nil {
		return false, fmt.Errorf("query schema_migrations: %w", err)
	}

	return count > 0, nil
}

func markMigrationApplied(db *sql.DB, name string) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	if _, err := db.Exec(`
 		INSERT INTO schema_migrations(version, applied_at)
 		VALUES (?, datetime('now'))
 		ON CONFLICT(version) DO NOTHING
 	`, name); err != nil {
		return fmt.Errorf("insert schema_migrations: %w", err)
	}

	return nil
}

func markMigrationReverted(db *sql.DB, name string) error {
	if _, err := db.Exec(`DELETE FROM schema_migrations WHERE version = ?`, name); err != nil {
		return fmt.Errorf("delete schema_migrations row: %w", err)
	}
	return nil
}

func reverse(items []string) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
