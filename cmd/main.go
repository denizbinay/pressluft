package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"pressluft/internal/server"
)

const defaultAddr = ":8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandler(), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("pressluft bootstrap listening", "addr", httpServer.Addr)
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
