package server

import (
	"log/slog"
	"net/http"

	"pressluft/internal/controlplane/server/middleware"
)

// WithRequestLogging wraps an HTTP handler with request logging.
func WithRequestLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return middleware.WithRequestLogging(next, logger)
}
