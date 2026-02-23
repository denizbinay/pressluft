package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/database"
	"pressluft/internal/server"

	// Register provider implementations.
	_ "pressluft/internal/provider/hetzner"
)

const defaultAddr = ":8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := database.Open(resolveDBPath(), logger)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandler(db.DB), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("pressluft listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func resolveAddr() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return defaultAddr
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

func resolveDBPath() string {
	if p := os.Getenv("PRESSLUFT_DB"); p != "" {
		return p
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "pressluft", "pressluft.db")
}
