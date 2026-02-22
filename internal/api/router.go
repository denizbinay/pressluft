package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"pressluft/internal/auth"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
)

type contextKey string

const sessionTokenContextKey contextKey = "session_token"

type Router struct {
	logger      *log.Logger
	authService *auth.Service
	jobs        jobs.Reader
	metrics     *metrics.Service
}

func NewRouter(logger *log.Logger, authService *auth.Service, jobsReader jobs.Reader, metricsService *metrics.Service) http.Handler {
	router := &Router{logger: logger, authService: authService, jobs: jobsReader, metrics: metricsService}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", router.handleLogin)
	mux.HandleFunc("/logout", router.handleLogout)
	mux.HandleFunc("/jobs", router.handleJobsList)
	mux.HandleFunc("/jobs/", router.handleJobDetail)
	mux.HandleFunc("/metrics", router.handleMetrics)

	return router.withAuth(mux)
}

func (r *Router) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/login" {
			next.ServeHTTP(w, req)
			return
		}

		tokenCookie, err := req.Cookie(auth.CookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		if _, err := r.authService.Validate(tokenCookie.Value, time.Now().UTC()); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		ctx := context.WithValue(req.Context(), sessionTokenContextKey, tokenCookie.Value)
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if payload.Email == "" || payload.Password == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "email and password are required")
		return
	}

	session, err := r.authService.Login(payload.Email, payload.Password, time.Now().UTC())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			r.logger.Printf("event=auth_login result=failure email=%s reason=invalid_credentials", payload.Email)
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
			return
		}
		r.logger.Printf("event=auth_login result=failure email=%s reason=auth_system_error", payload.Email)
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication failed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    session.SessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
	})

	r.logger.Printf("event=auth_login result=success email=%s", payload.Email)
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	token, _ := req.Context().Value(sessionTokenContextKey).(string)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	if err := r.authService.Revoke(token, time.Now().UTC()); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Path:     "/",
		Value:    "",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})

	r.logger.Printf("event=auth_logout result=success")
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (r *Router) handleJobsList(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	if r.jobs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "jobs service unavailable")
		return
	}

	allJobs, err := r.jobs.List(req.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list jobs")
		return
	}

	writeJSON(w, http.StatusOK, allJobs)
}

func (r *Router) handleJobDetail(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	if r.jobs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "jobs service unavailable")
		return
	}

	id := strings.TrimPrefix(req.URL.Path, "/jobs/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "not_found", "job not found")
		return
	}

	job, err := r.jobs.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load job")
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (r *Router) handleMetrics(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
		return
	}

	if r.metrics == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "metrics service unavailable")
		return
	}

	snapshot, err := r.metrics.GetSnapshot(req.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to load metrics")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}
