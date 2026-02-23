package server

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"pressluft/internal/provider"
)

const assetDir = "dist"

//go:embed all:dist
var embeddedDist embed.FS

// NewHandler creates the root HTTP handler. When db is nil the provider
// endpoints are not registered (useful for tests that only need static assets).
func NewHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/api/health", handleHealth)

	// Provider endpoints (only when database is available)
	if db != nil {
		ph := &providerHandler{store: provider.NewStore(db)}
		mux.HandleFunc("/api/providers", ph.route)
		mux.HandleFunc("/api/providers/", ph.routeWithID)
		mux.HandleFunc("/api/providers/validate", ph.handleValidate)
		mux.HandleFunc("/api/providers/types", ph.handleTypes)
	}

	// Dashboard SPA (catch-all)
	mux.Handle("/", newDashboardHandler())
	return mux
}

func newDashboardHandler() http.Handler {
	distFS, err := fs.Sub(embeddedDist, assetDir)
	if err != nil {
		return missingDashboardHandler()
	}

	if _, err := fs.Stat(distFS, "index.html"); err != nil {
		return missingDashboardHandler()
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, statErr := fs.Stat(distFS, path); statErr == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexRequest := r.Clone(r.Context())
		indexRequest.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, indexRequest)
	})
}

func missingDashboardHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><title>Dashboard not generated</title></head><body><h1>Dashboard assets not found</h1><p>Run <code>make build</code> and restart the binary.</p></body></html>"))
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
