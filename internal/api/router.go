package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/sites"
	"pressluft/internal/store"
)

type contextKey string

const sessionTokenContextKey contextKey = "session_token"
const userIDContextKey contextKey = "user_id"

type Router struct {
	logger       *log.Logger
	authService  *auth.Service
	jobs         jobs.Reader
	metrics      *metrics.Service
	auditor      audit.Recorder
	sites        *sites.Service
	environments *environments.Service
	backups      *backups.Service
}

func NewRouter(logger *log.Logger, authService *auth.Service, jobsReader jobs.Reader, metricsService *metrics.Service, auditRecorder audit.Recorder) http.Handler {
	var siteService *sites.Service
	var environmentService *environments.Service
	var backupService *backups.Service
	queue, ok := jobsReader.(sites.JobQueue)
	if ok {
		siteStore := store.DefaultSiteStore()
		siteService = sites.NewService(siteStore, queue)
		environmentService = environments.NewService(siteStore, queue)
		backupService = backups.NewService(siteStore, store.DefaultBackupStore(), queue)
	}

	router := &Router{logger: logger, authService: authService, jobs: jobsReader, metrics: metricsService, auditor: auditRecorder, sites: siteService, environments: environmentService, backups: backupService}
	mux := http.NewServeMux()
	mux.HandleFunc("/login", router.handleLogin)
	mux.HandleFunc("/logout", router.handleLogout)
	mux.HandleFunc("/jobs", router.handleJobsList)
	mux.HandleFunc("/jobs/", router.handleJobDetail)
	mux.HandleFunc("/metrics", router.handleMetrics)
	mux.HandleFunc("/sites", router.handleSites)
	mux.HandleFunc("/sites/", router.handleSiteByID)
	mux.HandleFunc("/environments/", router.handleEnvironmentByID)

	return router.withAuth(mux)
}

func (r *Router) handleSites(w http.ResponseWriter, req *http.Request) {
	if r.sites == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "sites service unavailable")
		return
	}

	switch req.Method {
	case http.MethodGet:
		sitesList, err := r.sites.List(req.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list sites")
			return
		}
		writeJSON(w, http.StatusOK, sitesList)
	case http.MethodPost:
		var payload struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.Slug) == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name and slug are required")
			return
		}

		jobID, err := r.sites.Create(req.Context(), payload.Name, payload.Slug)
		if err != nil {
			if errors.Is(err, store.ErrSiteSlugConflict) || errors.Is(err, sites.ErrMutationConflict) {
				writeError(w, http.StatusConflict, "conflict", "conflicting site mutation")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create site")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
	}
}

func (r *Router) handleSiteByID(w http.ResponseWriter, req *http.Request) {
	if r.sites == nil || r.environments == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "sites service unavailable")
		return
	}

	trimmed := strings.TrimPrefix(req.URL.Path, "/sites/")
	if trimmed == "" {
		writeError(w, http.StatusNotFound, "not_found", "site not found")
		return
	}
	parts := strings.Split(trimmed, "/")
	siteID := parts[0]
	if siteID == "" {
		writeError(w, http.StatusNotFound, "not_found", "site not found")
		return
	}

	if len(parts) == 1 {
		if req.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
			return
		}
		r.handleSiteDetail(w, req, siteID)
		return
	}

	if len(parts) == 2 && parts[1] == "environments" {
		r.handleSiteEnvironments(w, req, siteID)
		return
	}

	writeError(w, http.StatusNotFound, "not_found", "site not found")
}

func (r *Router) handleSiteDetail(w http.ResponseWriter, req *http.Request, id string) {
	site, err := r.sites.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrSiteNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "site not found")
			return
		}
		r.logger.Printf("event=site_lookup_failed id=%s err=%v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("failed to load site %s", id))
		return
	}

	writeJSON(w, http.StatusOK, site)
}

func (r *Router) handleSiteEnvironments(w http.ResponseWriter, req *http.Request, siteID string) {
	switch req.Method {
	case http.MethodGet:
		environmentsList, err := r.environments.ListBySiteID(req.Context(), siteID)
		if err != nil {
			if errors.Is(err, store.ErrSiteNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "site not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list site environments")
			return
		}
		writeJSON(w, http.StatusOK, environmentsList)
	case http.MethodPost:
		var payload struct {
			Name                string  `json:"name"`
			Slug                string  `json:"slug"`
			Type                string  `json:"type"`
			SourceEnvironmentID *string `json:"source_environment_id"`
			PromotionPreset     string  `json:"promotion_preset"`
		}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.Slug) == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "name and slug are required")
			return
		}

		if payload.Type != "staging" && payload.Type != "clone" {
			writeError(w, http.StatusBadRequest, "bad_request", "type must be staging or clone")
			return
		}

		if payload.PromotionPreset != "content-protect" && payload.PromotionPreset != "commerce-protect" {
			writeError(w, http.StatusBadRequest, "bad_request", "promotion_preset is invalid")
			return
		}

		jobID, err := r.environments.Create(req.Context(), environments.CreateInput{
			SiteID:              siteID,
			Name:                payload.Name,
			Slug:                payload.Slug,
			EnvironmentType:     payload.Type,
			SourceEnvironmentID: payload.SourceEnvironmentID,
			PromotionPreset:     payload.PromotionPreset,
		})
		if err != nil {
			switch {
			case errors.Is(err, store.ErrSiteNotFound):
				writeError(w, http.StatusNotFound, "not_found", "site not found")
			case errors.Is(err, environments.ErrMutationConflict) || errors.Is(err, store.ErrEnvironmentSlugConflict):
				writeError(w, http.StatusConflict, "conflict", "conflicting environment mutation")
			case errors.Is(err, store.ErrInvalidEnvironmentCreate):
				writeError(w, http.StatusBadRequest, "bad_request", "invalid environment create request")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to create environment")
			}
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
	}
}

func (r *Router) handleEnvironmentByID(w http.ResponseWriter, req *http.Request) {
	if r.environments == nil || r.backups == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "environments service unavailable")
		return
	}

	trimmed := strings.TrimPrefix(req.URL.Path, "/environments/")
	if trimmed == "" {
		writeError(w, http.StatusNotFound, "not_found", "environment not found")
		return
	}
	parts := strings.Split(trimmed, "/")
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "environment not found")
		return
	}

	if len(parts) == 1 {
		if req.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
			return
		}
		r.handleEnvironmentDetail(w, req, id)
		return
	}

	if len(parts) == 2 && parts[1] == "backups" {
		r.handleEnvironmentBackups(w, req, id)
		return
	}

	writeError(w, http.StatusNotFound, "not_found", "environment not found")
}

func (r *Router) handleEnvironmentDetail(w http.ResponseWriter, req *http.Request, id string) {
	environment, err := r.environments.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrEnvironmentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "environment not found")
			return
		}
		r.logger.Printf("event=environment_lookup_failed id=%s err=%v", id, err)
		writeError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("failed to load environment %s", id))
		return
	}

	writeJSON(w, http.StatusOK, environment)
}

func (r *Router) handleEnvironmentBackups(w http.ResponseWriter, req *http.Request, environmentID string) {
	switch req.Method {
	case http.MethodGet:
		backupList, err := r.backups.ListByEnvironmentID(req.Context(), environmentID)
		if err != nil {
			if errors.Is(err, store.ErrEnvironmentNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "environment not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list environment backups")
			return
		}
		writeJSON(w, http.StatusOK, backupList)
	case http.MethodPost:
		var payload struct {
			BackupScope string `json:"backup_scope"`
		}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}

		if payload.BackupScope != "db" && payload.BackupScope != "files" && payload.BackupScope != "full" {
			writeError(w, http.StatusBadRequest, "bad_request", "backup_scope must be db, files, or full")
			return
		}

		jobID, err := r.backups.Create(req.Context(), backups.CreateInput{EnvironmentID: environmentID, BackupScope: payload.BackupScope})
		if err != nil {
			switch {
			case errors.Is(err, store.ErrEnvironmentNotFound):
				writeError(w, http.StatusNotFound, "not_found", "environment not found")
			case errors.Is(err, backups.ErrMutationConflict):
				writeError(w, http.StatusConflict, "conflict", "conflicting backup mutation")
			case errors.Is(err, store.ErrInvalidBackupScope):
				writeError(w, http.StatusBadRequest, "bad_request", "backup_scope must be db, files, or full")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to create environment backup")
			}
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "bad_request", "method not allowed")
	}
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

		session, err := r.authService.Validate(tokenCookie.Value, time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		ctx := context.WithValue(req.Context(), sessionTokenContextKey, tokenCookie.Value)
		ctx = context.WithValue(ctx, userIDContextKey, session.UserID)
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
			if !r.recordAudit(req.Context(), auth.DefaultUserID, "auth_login", "session", auth.DefaultUserID, "failed") {
				writeError(w, http.StatusInternalServerError, "internal_error", "request could not be audited")
				return
			}
			r.logger.Printf("event=auth_login result=failure email=%s reason=invalid_credentials", payload.Email)
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
			return
		}
		if !r.recordAudit(req.Context(), auth.DefaultUserID, "auth_login", "session", auth.DefaultUserID, "failed") {
			writeError(w, http.StatusInternalServerError, "internal_error", "request could not be audited")
			return
		}
		r.logger.Printf("event=auth_login result=failure email=%s reason=auth_system_error", payload.Email)
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication failed")
		return
	}

	if !r.recordAudit(req.Context(), session.UserID, "auth_login", "session", session.UserID, "succeeded") {
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be audited")
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
	userID, _ := req.Context().Value(userIDContextKey).(string)
	if userID == "" {
		userID = auth.DefaultUserID
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	if err := r.authService.Revoke(token, time.Now().UTC()); err != nil {
		_ = r.recordAudit(req.Context(), userID, "auth_logout", "session", userID, "failed")
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	if !r.recordAudit(req.Context(), userID, "auth_logout", "session", userID, "succeeded") {
		writeError(w, http.StatusInternalServerError, "internal_error", "request could not be audited")
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

func (r *Router) recordAudit(ctx context.Context, userID string, action string, resourceType string, resourceID string, result string) bool {
	if r.auditor == nil {
		return true
	}

	err := r.auditor.Record(ctx, audit.Entry{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
	})
	if err != nil {
		r.logger.Printf("event=audit_write_failed action=%s resource_type=%s", action, resourceType)
		return false
	}

	return true
}
