package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: go run ./migrations/migrate.go <up|down>")
	}

	action := os.Args[1]
	if action != "up" && action != "down" {
		fatal("invalid action %q (expected up or down)", action)
	}

	dbPath := os.Getenv("PRESSLUFT_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join(".", "pressluft.db")
	}

	wd, err := os.Getwd()
	if err != nil {
		fatal("resolve working directory: %v", err)
	}

	migrationsDir := filepath.Join(wd, "migrations")
	if err := runMigrations(action, migrationsDir, dbPath); err != nil {
		fatal("migration %s failed: %v", action, err)
	}

	fmt.Printf("migration %s completed on %s\n", action, dbPath)
}

func runMigrations(action, migrationsDir, dbPath string) error {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return errors.New("sqlite3 CLI not found in PATH")
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

	sort.Strings(files)
	if action == "down" {
		reverse(files)
	}

	for _, file := range files {
		if err := applyFile(dbPath, file); err != nil {
			return fmt.Errorf("apply %s: %w", filepath.Base(file), err)
		}
	}

	return nil
}

func applyFile(dbPath, sqlFile string) error {
	cmd := exec.Command("sqlite3", dbPath, ".read "+sqlFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlite3 run: %w", err)
	}
	return nil
}

func reverse(items []string) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
