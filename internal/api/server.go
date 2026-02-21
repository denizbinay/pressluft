package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/domains"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/migration"
	"pressluft/internal/promotion"
	"pressluft/internal/settings"
	"pressluft/internal/sites"
	"pressluft/internal/ssh"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

type Server struct {
	authService *auth.Service
	siteService *sites.Service
	envService  *environments.Service
	promoteSvc  *promotion.Service
	magicSvc    *ssh.Service
	settingsSvc *settings.Service
	jobsSvc     *jobs.Service
	metricsSvc  *metrics.Service
	backupSvc   *backups.Service
	domainSvc   *domains.Service
	migrateSvc  *migration.Service
	audit       *audit.Service
	mux         *http.ServeMux
}

func NewServer(authService *auth.Service, siteService *sites.Service, envService *environments.Service, promotionService *promotion.Service, magicLoginService *ssh.Service, settingsService *settings.Service, jobsService *jobs.Service, metricsService *metrics.Service, backupService *backups.Service, domainService *domains.Service, migrationService *migration.Service, auditService *audit.Service) *Server {
	s := &Server{
		authService: authService,
		siteService: siteService,
		envService:  envService,
		promoteSvc:  promotionService,
		magicSvc:    magicLoginService,
		settingsSvc: settingsService,
		jobsSvc:     jobsService,
		metricsSvc:  metricsService,
		backupSvc:   backupService,
		domainSvc:   domainService,
		migrateSvc:  migrationService,
		audit:       auditService,
		mux:         http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/logout", s.withAuth(s.handleLogout))
	s.mux.HandleFunc("/api/sites", s.withAuth(s.handleSites))
	s.mux.HandleFunc("/api/sites/", s.withAuth(s.handleSiteByID))
	s.mux.HandleFunc("/api/environments/", s.withAuth(s.handleEnvironmentByID))
	s.mux.HandleFunc("/api/domains/", s.withAuth(s.handleDomainByID))
	s.mux.HandleFunc("/api/jobs", s.withAuth(s.handleJobs))
	s.mux.HandleFunc("/api/jobs/", s.withAuth(s.handleJobByID))
	s.mux.HandleFunc("/api/metrics", s.withAuth(s.handleMetrics))
	s.mux.HandleFunc("/_admin/settings/domain-config", s.withAuth(s.handleDomainConfigSettings))
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	jobsList, err := s.jobsSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, jobsList)
}

func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if strings.HasSuffix(rest, "/cancel") {
		jobID := strings.TrimSuffix(rest, "/cancel")
		jobID = strings.TrimSpace(jobID)
		if jobID == "" || strings.Contains(jobID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleCancelJob(w, r, jobID)
		return
	}

	if strings.Contains(rest, "/") {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	job, err := s.jobsSvc.Get(r.Context(), rest)
	if err != nil {
		switch {
		case errors.Is(err, jobs.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, jobs.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid job id")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	job, err := s.jobsSvc.Cancel(r.Context(), jobID)
	if err != nil {
		switch {
		case errors.Is(err, jobs.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, jobs.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid job id")
		case errors.Is(err, jobs.ErrNotCancellable):
			writeError(w, http.StatusConflict, "job_not_cancellable", "job cannot be cancelled from current state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "job.cancel",
		ResourceType: "job",
		ResourceID:   jobID,
		Result:       "success",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": job.Status})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	snapshot, err := s.metricsSvc.Snapshot(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleDomainConfigSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.settingsSvc.GetDomainConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		input, err := decodeDomainConfigUpdateRequest(r)
		if err != nil {
			if errors.Is(err, settings.ErrInvalidInput) {
				writeError(w, http.StatusBadRequest, "invalid_request", "invalid settings payload")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
			return
		}

		cfg, err := s.settingsSvc.UpdateDomainConfig(r.Context(), input)
		if err != nil {
			if errors.Is(err, settings.ErrInvalidInput) {
				writeError(w, http.StatusBadRequest, "invalid_request", "invalid settings payload")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		userID, _ := r.Context().Value(userIDContextKey).(string)
		if err := s.audit.Record(r.Context(), audit.Entry{
			UserID:       userID,
			Action:       "settings.domain_config.update",
			ResourceType: "settings",
			ResourceID:   "domain-config",
			Result:       "success",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, cfg)
	default:
		methodNotAllowed(w)
	}
}

func decodeDomainConfigUpdateRequest(r *http.Request) (settings.UpdateDomainConfigInput, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return settings.UpdateDomainConfigInput{}, err
	}

	required := []string{"control_plane_domain", "preview_domain", "dns01_provider", "dns01_credentials_json"}
	for _, key := range required {
		if _, ok := raw[key]; !ok {
			return settings.UpdateDomainConfigInput{}, settings.ErrInvalidInput
		}
	}

	controlPlaneDomain, err := decodeOptionalString(raw["control_plane_domain"])
	if err != nil {
		return settings.UpdateDomainConfigInput{}, settings.ErrInvalidInput
	}
	previewDomain, err := decodeOptionalString(raw["preview_domain"])
	if err != nil {
		return settings.UpdateDomainConfigInput{}, settings.ErrInvalidInput
	}
	dns01Provider, err := decodeOptionalString(raw["dns01_provider"])
	if err != nil {
		return settings.UpdateDomainConfigInput{}, settings.ErrInvalidInput
	}

	credentials, err := decodeOptionalStringMap(raw["dns01_credentials_json"])
	if err != nil {
		return settings.UpdateDomainConfigInput{}, settings.ErrInvalidInput
	}

	return settings.UpdateDomainConfigInput{
		ControlPlaneDomain:   controlPlaneDomain,
		PreviewDomain:        previewDomain,
		DNS01Provider:        dns01Provider,
		DNS01CredentialsJSON: credentials,
	}, nil
}

func decodeOptionalString(raw json.RawMessage) (*string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

func decodeOptionalStringMap(raw json.RawMessage) (map[string]string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	var value map[string]string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, settings.ErrInvalidInput
	}
	return value, nil
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	loginResult, err := s.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "auth_invalid_credentials", "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    loginResult.SessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
	})

	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       loginResult.UserID,
		Action:       "auth.login",
		ResourceType: "auth_session",
		ResourceID:   "current",
		Result:       "success",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth_unauthorized", "authentication required")
		return
	}

	if err := s.authService.Logout(r.Context(), token); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "auth_unauthorized", "authentication required")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "auth.logout",
		ResourceType: "auth_session",
		ResourceID:   "current",
		Result:       "success",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSites(w, r)
	case http.MethodPost:
		s.handleCreateSite(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSiteByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if strings.HasSuffix(rest, "/environments") {
		siteID := strings.TrimSuffix(rest, "/environments")
		siteID = strings.TrimSpace(siteID)
		if siteID == "" || strings.Contains(siteID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleSiteEnvironments(w, r, siteID)
		return
	}

	if strings.HasSuffix(rest, "/import") {
		siteID := strings.TrimSuffix(rest, "/import")
		siteID = strings.TrimSpace(siteID)
		if siteID == "" || strings.Contains(siteID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleSiteImport(w, r, siteID)
		return
	}

	if strings.HasSuffix(rest, "/reset") {
		siteID := strings.TrimSuffix(rest, "/reset")
		siteID = strings.TrimSpace(siteID)
		if siteID == "" || strings.Contains(siteID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleSiteReset(w, r, siteID)
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id := rest
	if strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	site, err := s.siteService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sites.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	sitesList, err := s.siteService.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, sitesList)
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.siteService.Create(r.Context(), sites.CreateInput{Name: req.Name, Slug: req.Slug})
	if err != nil {
		switch {
		case errors.Is(err, sites.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid site create payload")
		case errors.Is(err, sites.ErrSlugConflict):
			writeError(w, http.StatusConflict, "conflict", "site slug already exists")
		case errors.Is(err, jobs.ErrConcurrencyConflict), errors.Is(err, sites.ErrNoAvailableNode), errors.Is(err, sites.ErrNodeMissingPublicIP):
			writeError(w, http.StatusConflict, "conflict", "site creation blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "site.create",
		ResourceType: "site",
		ResourceID:   result.SiteID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleSiteReset(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	site, err := s.siteService.ResetFailed(r.Context(), siteID)
	if err != nil {
		switch {
		case errors.Is(err, sites.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, sites.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid site id")
		case errors.Is(err, sites.ErrResourceNotFailed):
			writeError(w, http.StatusConflict, "resource_not_failed", "site must be in failed state to reset")
		case errors.Is(err, sites.ErrResetValidationFailed):
			writeError(w, http.StatusConflict, "reset_validation_failed", "site reset safety validation failed")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "site.reset",
		ResourceType: "site",
		ResourceID:   siteID,
		Result:       "success",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": site.Status})
}

func (s *Server) handleSiteEnvironments(w http.ResponseWriter, r *http.Request, siteID string) {
	switch r.Method {
	case http.MethodGet:
		envs, err := s.envService.ListBySite(r.Context(), siteID)
		if err != nil {
			if errors.Is(err, environments.ErrSiteNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, envs)
	case http.MethodPost:
		var req struct {
			Name                string  `json:"name"`
			Slug                string  `json:"slug"`
			Type                string  `json:"type"`
			SourceEnvironmentID *string `json:"source_environment_id"`
			PromotionPreset     string  `json:"promotion_preset"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
			return
		}

		result, err := s.envService.Create(r.Context(), environments.CreateInput{
			SiteID:              siteID,
			Name:                req.Name,
			Slug:                req.Slug,
			EnvironmentType:     req.Type,
			SourceEnvironmentID: req.SourceEnvironmentID,
			PromotionPreset:     req.PromotionPreset,
		})
		if err != nil {
			switch {
			case errors.Is(err, environments.ErrInvalidInput):
				writeError(w, http.StatusBadRequest, "invalid_request", "invalid environment create payload")
			case errors.Is(err, environments.ErrSiteNotFound), errors.Is(err, environments.ErrEnvironmentNotFound):
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
			case errors.Is(err, jobs.ErrConcurrencyConflict), errors.Is(err, environments.ErrNoAvailableNode), errors.Is(err, environments.ErrNodeMissingPublicIP):
				writeError(w, http.StatusConflict, "conflict", "environment creation blocked by current infrastructure state")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		userID, _ := r.Context().Value(userIDContextKey).(string)
		if err := s.audit.Record(r.Context(), audit.Entry{
			UserID:       userID,
			Action:       "environment.create",
			ResourceType: "environment",
			ResourceID:   result.EnvironmentID,
			Result:       "accepted",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSiteImport(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		ArchiveURL string `json:"archive_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.migrateSvc.ImportSite(r.Context(), migration.ImportInput{
		SiteID:     siteID,
		ArchiveURL: req.ArchiveURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, migration.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid site import payload")
		case errors.Is(err, migration.ErrSiteNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, migration.ErrEnvironmentNotFound):
			writeError(w, http.StatusConflict, "conflict", "site import blocked by current infrastructure state")
		case errors.Is(err, migration.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "site import blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "site.import",
		ResourceType: "site",
		ResourceID:   siteID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/environments/")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if strings.HasSuffix(rest, "/backups") {
		environmentID := strings.TrimSuffix(rest, "/backups")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentBackups(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/deploy") {
		environmentID := strings.TrimSuffix(rest, "/deploy")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentDeploy(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/updates") {
		environmentID := strings.TrimSuffix(rest, "/updates")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentUpdates(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/restore") {
		environmentID := strings.TrimSuffix(rest, "/restore")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentRestore(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/drift-check") {
		environmentID := strings.TrimSuffix(rest, "/drift-check")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentDriftCheck(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/promote") {
		environmentID := strings.TrimSuffix(rest, "/promote")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentPromote(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/domains") {
		environmentID := strings.TrimSuffix(rest, "/domains")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentDomains(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/cache/purge") {
		environmentID := strings.TrimSuffix(rest, "/cache/purge")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentCachePurge(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/cache") {
		environmentID := strings.TrimSuffix(rest, "/cache")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentCache(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/magic-login") {
		environmentID := strings.TrimSuffix(rest, "/magic-login")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentMagicLogin(w, r, environmentID)
		return
	}

	if strings.HasSuffix(rest, "/reset") {
		environmentID := strings.TrimSuffix(rest, "/reset")
		environmentID = strings.TrimSpace(environmentID)
		if environmentID == "" || strings.Contains(environmentID, "/") {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		s.handleEnvironmentReset(w, r, environmentID)
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id := rest
	if strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	environment, err := s.envService.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, environments.ErrEnvironmentNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, environment)
}

func (s *Server) handleEnvironmentBackups(w http.ResponseWriter, r *http.Request, environmentID string) {
	switch r.Method {
	case http.MethodGet:
		backupList, err := s.backupSvc.ListByEnvironment(r.Context(), environmentID)
		if err != nil {
			if errors.Is(err, backups.ErrEnvironmentNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, backupList)
	case http.MethodPost:
		var req struct {
			BackupScope string `json:"backup_scope"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
			return
		}

		result, err := s.backupSvc.Create(r.Context(), backups.CreateInput{
			EnvironmentID: environmentID,
			BackupScope:   req.BackupScope,
		})
		if err != nil {
			switch {
			case errors.Is(err, backups.ErrInvalidInput):
				writeError(w, http.StatusBadRequest, "bad_request", "invalid backup payload")
			case errors.Is(err, backups.ErrEnvironmentNotFound):
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
			case errors.Is(err, jobs.ErrConcurrencyConflict):
				writeError(w, http.StatusConflict, "conflict", "backup creation blocked by current infrastructure state")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		userID, _ := r.Context().Value(userIDContextKey).(string)
		if err := s.audit.Record(r.Context(), audit.Entry{
			UserID:       userID,
			Action:       "backup.create",
			ResourceType: "environment",
			ResourceID:   environmentID,
			Result:       "accepted",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleEnvironmentDeploy(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		SourceType string `json:"source_type"`
		SourceRef  string `json:"source_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.envService.Deploy(r.Context(), environments.DeployInput{
		EnvironmentID: environmentID,
		SourceType:    req.SourceType,
		SourceRef:     req.SourceRef,
	})
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid deploy payload")
		case errors.Is(err, environments.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "deploy blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.deploy",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentDomains(w http.ResponseWriter, r *http.Request, environmentID string) {
	switch r.Method {
	case http.MethodGet:
		domainsList, err := s.domainSvc.ListByEnvironment(r.Context(), environmentID)
		if err != nil {
			if errors.Is(err, domains.ErrEnvironmentNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, domainsList)
	case http.MethodPost:
		var req struct {
			Hostname string `json:"hostname"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
			return
		}

		result, err := s.domainSvc.Add(r.Context(), domains.AddInput{
			EnvironmentID: environmentID,
			Hostname:      req.Hostname,
		})
		if err != nil {
			switch {
			case errors.Is(err, domains.ErrInvalidInput):
				writeError(w, http.StatusBadRequest, "invalid_request", "invalid domain payload")
			case errors.Is(err, domains.ErrEnvironmentNotFound):
				writeError(w, http.StatusNotFound, "not_found", "resource not found")
			case errors.Is(err, domains.ErrDomainConflict), errors.Is(err, domains.ErrEnvironmentNotActive), errors.Is(err, domains.ErrNodeMissingPublicIP), errors.Is(err, jobs.ErrConcurrencyConflict):
				writeError(w, http.StatusConflict, "conflict", "domain add blocked by current infrastructure state")
			default:
				writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
			return
		}

		userID, _ := r.Context().Value(userIDContextKey).(string)
		if err := s.audit.Record(r.Context(), audit.Entry{
			UserID:       userID,
			Action:       "domain.add",
			ResourceType: "environment",
			ResourceID:   environmentID,
			Result:       "accepted",
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleDomainByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/domains/")
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.Contains(rest, "/") {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}

	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}

	result, err := s.domainSvc.Remove(r.Context(), rest)
	if err != nil {
		switch {
		case errors.Is(err, domains.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid domain id")
		case errors.Is(err, domains.ErrDomainNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, domains.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "domain removal blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "domain.remove",
		ResourceType: "domain",
		ResourceID:   rest,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentUpdates(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		Scope string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.envService.Updates(r.Context(), environments.UpdatesInput{
		EnvironmentID: environmentID,
		Scope:         req.Scope,
	})
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid updates payload")
		case errors.Is(err, environments.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "updates blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.update",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentCache(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w)
		return
	}

	var req struct {
		FastCGICacheEnabled *bool `json:"fastcgi_cache_enabled"`
		RedisCacheEnabled   *bool `json:"redis_cache_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.envService.ToggleCache(r.Context(), environments.CacheToggleInput{
		EnvironmentID:       environmentID,
		FastCGICacheEnabled: req.FastCGICacheEnabled,
		RedisCacheEnabled:   req.RedisCacheEnabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid cache payload")
		case errors.Is(err, environments.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "cache toggle blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.cache_toggle",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentCachePurge(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	result, err := s.envService.PurgeCache(r.Context(), environments.CachePurgeInput{EnvironmentID: environmentID})
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid cache purge payload")
		case errors.Is(err, environments.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "cache purge blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.cache_purge",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentMagicLogin(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	result, err := s.magicSvc.CreateMagicLogin(r.Context(), environmentID)
	if err != nil {
		if auditErr := s.recordMagicLoginAudit(r.Context(), environmentID, "failed"); auditErr != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		switch {
		case errors.Is(err, ssh.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, ssh.ErrEnvironmentNotActive):
			writeError(w, http.StatusConflict, "environment_not_active", "Environment is not in active state")
		case errors.Is(err, ssh.ErrNodeUnreachable):
			writeError(w, http.StatusBadGateway, "node_unreachable", "SSH connection to node failed or timed out")
		case errors.Is(err, ssh.ErrWPCliError):
			writeError(w, http.StatusBadGateway, "wp_cli_error", "WP-CLI command failed")
		case errors.Is(err, ssh.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid magic login request")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	if err := s.recordMagicLoginAudit(r.Context(), environmentID, "success"); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleEnvironmentReset(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	env, err := s.envService.ResetFailed(r.Context(), environmentID)
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid environment id")
		case errors.Is(err, environments.ErrResourceNotFailed):
			writeError(w, http.StatusConflict, "resource_not_failed", "environment must be in failed state to reset")
		case errors.Is(err, environments.ErrResetValidationFailed):
			writeError(w, http.StatusConflict, "reset_validation_failed", "environment reset safety validation failed")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.reset",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "success",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": env.Status})
}

func (s *Server) recordMagicLoginAudit(ctx context.Context, environmentID, result string) error {
	userID, _ := ctx.Value(userIDContextKey).(string)
	return s.audit.Record(ctx, audit.Entry{
		UserID:       userID,
		Action:       "magic_login",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       result,
	})
}

func (s *Server) handleEnvironmentRestore(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		BackupID string `json:"backup_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.envService.Restore(r.Context(), environments.RestoreInput{
		EnvironmentID: environmentID,
		BackupID:      req.BackupID,
	})
	if err != nil {
		switch {
		case errors.Is(err, environments.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid restore payload")
		case errors.Is(err, environments.ErrEnvironmentNotFound), errors.Is(err, environments.ErrBackupNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, environments.ErrBackupNotCompleted), errors.Is(err, environments.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "restore blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.restore",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentDriftCheck(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	result, err := s.promoteSvc.DriftCheck(r.Context(), promotion.DriftCheckInput{EnvironmentID: environmentID})
	if err != nil {
		switch {
		case errors.Is(err, promotion.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid drift-check payload")
		case errors.Is(err, promotion.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, promotion.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "drift check blocked by current infrastructure state")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.drift_check",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) handleEnvironmentPromote(w http.ResponseWriter, r *http.Request, environmentID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		TargetEnvironmentID string `json:"target_environment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	result, err := s.promoteSvc.Promote(r.Context(), promotion.PromoteInput{
		EnvironmentID:       environmentID,
		TargetEnvironmentID: req.TargetEnvironmentID,
	})
	if err != nil {
		switch {
		case errors.Is(err, promotion.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid promote payload")
		case errors.Is(err, promotion.ErrEnvironmentNotFound):
			writeError(w, http.StatusNotFound, "not_found", "resource not found")
		case errors.Is(err, promotion.ErrDriftGateNotMet), errors.Is(err, promotion.ErrBackupGateNotMet), errors.Is(err, promotion.ErrEnvironmentNotActive), errors.Is(err, jobs.ErrConcurrencyConflict):
			writeError(w, http.StatusConflict, "conflict", "promotion blocked by safety gates")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	userID, _ := r.Context().Value(userIDContextKey).(string)
	if err := s.audit.Record(r.Context(), audit.Entry{
		UserID:       userID,
		Action:       "environment.promote",
		ResourceType: "environment",
		ResourceID:   environmentID,
		Result:       "accepted",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": result.JobID})
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := sessionToken(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "auth_unauthorized", "authentication required")
			return
		}

		userID, err := s.authService.ValidateSession(r.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, "auth_unauthorized", "authentication required")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func sessionToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", errors.New("empty session token")
	}
	return value, nil
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
