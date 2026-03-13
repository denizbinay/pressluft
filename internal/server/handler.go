package server

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
	"pressluft/internal/auth"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/ws"
)

const assetDir = "dist"

//go:embed all:dist
var embeddedDist embed.FS

// NewHandler creates the root HTTP handler. When db is nil the provider
// endpoints are not registered (useful for tests that only need static assets).
func NewHandler(db *sql.DB) http.Handler {
	return NewHandlerWithHub(db, nil, nil, nil)
}

// NewHandlerWithHub creates the root HTTP handler with an optional WebSocket hub
// for real-time agent status. When hub is nil, agent status endpoints will return
// stored database values only.
func NewHandlerWithHub(db *sql.DB, hub *ws.Hub, wsHandler *WSHandler, nodeHandler *NodeHandler) http.Handler {
	return NewHandlerWithOptions(db, hub, wsHandler, nodeHandler, HandlerOptions{})
}

func NewHandlerWithOptions(db *sql.DB, hub *ws.Hub, wsHandler *WSHandler, nodeHandler *NodeHandler, options HandlerOptions) http.Handler {
	mux := http.NewServeMux()
	operatorMux := http.NewServeMux()
	authorize := func(handler http.Handler, allow func(auth.Actor) bool) http.Handler {
		if options.Authenticator == nil {
			return handler
		}
		return withAuthorization(handler, allow)
	}

	// Health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		handleHealth(w, r, options)
	})

	// Agent WebSocket
	if wsHandler != nil {
		mux.HandleFunc("/ws/agent", wsHandler.handleAgentWebSocket)
	}

	// Node registration
	if nodeHandler != nil {
		nodeRegister := http.HandlerFunc(nodeHandler.handleNodeRegister)
		mux.Handle("/api/nodes/", withRateLimit(nodeRegister, newRateLimiter(10, time.Minute), "node-register"))
	}

	var authActivityStore *activity.Store
	if db != nil {
		authActivityStore = activity.NewStore(db)
	}
	authHandler := &authHandler{
		service:       options.AuthService,
		activityStore: authActivityStore,
		logger:        slog.Default(),
	}
	if options.AuthService != nil {
		mux.Handle("/api/auth/me", withOptionalActor(http.HandlerFunc(authHandler.handleMe), options.Authenticator))
		mux.Handle("/api/auth/logout", withOptionalActor(http.HandlerFunc(authHandler.handleLogout), options.Authenticator))
		mux.Handle("/api/auth/login", withRateLimit(http.HandlerFunc(authHandler.handleLogin), newRateLimiter(10, time.Minute), "auth-login"))
	}

	// Provider endpoints (only when database is available)
	if db != nil {
		// Create activity store once and share it across handlers
		activityStore := activity.NewStore(db)

		ph := &providerHandler{
			store:         provider.NewStore(db),
			activityStore: activityStore,
		}
		operatorMux.Handle("/api/providers", authorize(http.HandlerFunc(ph.route), auth.RequireCapability(auth.CapabilityManageProviders)))
		operatorMux.Handle("/api/providers/", authorize(http.HandlerFunc(ph.routeWithID), auth.RequireCapability(auth.CapabilityManageProviders)))
		operatorMux.Handle("/api/providers/validate", authorize(withRateLimit(http.HandlerFunc(ph.handleValidate), newRateLimiter(15, time.Minute), "provider-validate"), auth.RequireCapability(auth.CapabilityManageProviders)))
		operatorMux.Handle("/api/providers/types", authorize(http.HandlerFunc(ph.handleTypes), auth.RequireCapability(auth.CapabilityManageProviders)))

		jobStore := orchestrator.NewStore(db)
		serverStore := NewServerStore(db)
		siteStore := NewSiteStore(db)
		domainStore := NewDomainStore(db)
		_ = domainStore.BackfillLegacyPrimaryDomains(context.Background())

		sh := &serversHandler{
			providerStore: provider.NewStore(db),
			serverStore:   serverStore,
			siteStore:     siteStore,
			jobStore:      jobStore,
			activityStore: activityStore,
			hub:           hub,
		}
		operatorMux.Handle("/api/servers", authorize(withRateLimit(http.HandlerFunc(sh.route), newRateLimiter(30, time.Minute), "servers"), auth.RequireCapability(auth.CapabilityManageServers)))
		operatorMux.Handle("/api/servers/", authorize(withRateLimit(http.HandlerFunc(sh.routeWithPath), newRateLimiter(60, time.Minute), "servers-path"), auth.RequireCapability(auth.CapabilityManageServers)))

		sih := &sitesHandler{
			store:         siteStore,
			serverStore:   serverStore,
			jobStore:      jobStore,
			domainStore:   domainStore,
			activityStore: activityStore,
			hub:           hub,
		}
		operatorMux.Handle("/api/sites", authorize(withRateLimit(http.HandlerFunc(sih.route), newRateLimiter(30, time.Minute), "sites"), auth.RequireCapability(auth.CapabilityManageSites)))
		operatorMux.Handle("/api/sites/", authorize(withRateLimit(http.HandlerFunc(sih.routeWithID), newRateLimiter(60, time.Minute), "sites-path"), auth.RequireCapability(auth.CapabilityManageSites)))

		dh := &domainsHandler{store: domainStore, activityStore: activityStore}
		operatorMux.Handle("/api/domains", authorize(withRateLimit(http.HandlerFunc(dh.route), newRateLimiter(30, time.Minute), "domains"), auth.RequireCapability(auth.CapabilityManageSites)))
		operatorMux.Handle("/api/domains/", authorize(withRateLimit(http.HandlerFunc(dh.routeWithID), newRateLimiter(60, time.Minute), "domains-path"), auth.RequireCapability(auth.CapabilityManageSites)))

		jh := &jobsHandler{
			store:         jobStore,
			serverStore:   serverStore,
			activityStore: activityStore,
		}
		operatorMux.Handle("/api/jobs", authorize(withRateLimit(http.HandlerFunc(jh.route), newRateLimiter(30, time.Minute), "jobs"), auth.RequireCapability(auth.CapabilityQueueJobs)))
		operatorMux.Handle("/api/jobs/", authorize(http.HandlerFunc(jh.routeWithID), auth.RequireCapability(auth.CapabilityQueueJobs)))

		ah := &activityHandler{store: activityStore}
		operatorMux.Handle("/api/activity", authorize(http.HandlerFunc(ah.route), auth.RequireCapability(auth.CapabilityReadActivity)))
		operatorMux.Handle("/api/activity/", authorize(http.HandlerFunc(ah.routeWithID), auth.RequireCapability(auth.CapabilityReadActivity)))

		// Inject activity handler into servers handler for /api/servers/{id}/activity
		sh.activityHandler = ah
		sih.activityHandler = ah
	}
	mux.Handle("/api/", withOperatorAuth(operatorMux, options.Authenticator))

	// Dashboard SPA (catch-all)
	dashboard := newDashboardHandler(options.Authenticator)
	mux.Handle("/", dashboard)
	return withSecurityHeaders(mux, options.IsDev)
}

func newDashboardHandler(authenticator auth.Authenticator) http.Handler {
	distFS, err := fs.Sub(embeddedDist, assetDir)
	if err != nil {
		return missingDashboardHandler()
	}

	if _, err := fs.Stat(distFS, "index.html"); err != nil {
		return missingDashboardHandler()
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldAllowPublicDashboardPath(r.URL.Path) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			if _, statErr := fs.Stat(distFS, path); statErr == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		if authenticator != nil {
			actor, err := authenticator.Authenticate(r)
			if err != nil || !actor.IsAuthenticated() {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			r = r.WithContext(auth.ContextWithActor(r.Context(), actor))
		}
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

func shouldAllowPublicDashboardPath(path string) bool {
	switch path {
	case "/login", "/favicon.ico", "/robots.txt":
		return true
	}
	return strings.HasPrefix(path, "/_nuxt/")
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

func handleHealth(w http.ResponseWriter, _ *http.Request, options HandlerOptions) {
	payload := apitypes.HealthResponse{Status: "healthy"}
	if options.IsDev {
		mode := platform.DetectCallbackURLMode(options.ControlPlaneURL)
		payload.CallbackURLMode = mode
		if mode == platform.CallbackURLModeEphemeral {
			payload.CallbackURLWarning = "Cloudflare quick tunnels are session-scoped. Remote agents configured against this URL will not reconnect after control-plane restart."
		}
	}
	respondJSON(w, http.StatusOK, payload)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	// Prevent caching of API responses
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
