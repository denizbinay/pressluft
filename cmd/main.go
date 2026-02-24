package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pressluft/internal/database"
	"pressluft/internal/orchestrator"
	"pressluft/internal/provider"
	"pressluft/internal/server"
	"pressluft/internal/worker"

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

	// Create stores for worker
	jobStore := orchestrator.NewStore(db.DB)
	serverStore := server.NewServerStore(db.DB)
	providerStore := provider.NewStore(db.DB)

	// Create worker with executor
	executor := worker.NewExecutor(
		jobStore,
		worker.NewServerStoreAdapter(serverStore),
		worker.NewProviderStoreAdapter(providerStore),
		logger,
	)
	w := worker.New(jobStore, executor, logger, worker.DefaultConfig())

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in background
	go w.Run(ctx)

	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandler(db.DB), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info("shutdown signal received")
		cancel() // Stop worker

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", "error", err)
		}
	}()

	logger.Info("pressluft listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}

	logger.Info("pressluft stopped")
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
