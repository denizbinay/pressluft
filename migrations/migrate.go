package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pressluft/internal/migrations"
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
	if err := migrations.Run(action, migrationsDir, dbPath); err != nil {
		fatal("migration %s failed: %v", action, err)
	}

	fmt.Printf("migration %s completed on %s\n", action, dbPath)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
