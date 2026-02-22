package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/nodes"
	"pressluft/internal/providers"
	"pressluft/internal/sites"
	"pressluft/internal/store"
)

func TestLoginSuccessSetsSessionCookie(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)

	body := strings.NewReader(`{"email":"admin@pressluft.local","password":"pressluft-dev-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	if got := rr.Header().Get("Set-Cookie"); !strings.Contains(got, "session_token=") {
		t.Fatalf("Set-Cookie = %q, expected session_token cookie", got)
	}

	var payload map[string]bool
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if !payload["success"] {
		t.Fatalf("success = %v, want true", payload["success"])
	}
}

func TestLoginFailureReturnsUnauthorizedShape(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)

	body := strings.NewReader(`{"email":"admin@pressluft.local","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}

	assertErrorShape(t, rr.Body.Bytes(), "unauthorized")
}

func TestLogoutRevokesSessionAndClearsCookie(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRR := httptest.NewRecorder()
	router.ServeHTTP(logoutRR, logoutReq)

	if logoutRR.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRR.Code, http.StatusOK)
	}

	clearCookie := logoutRR.Header().Get("Set-Cookie")
	if !strings.Contains(clearCookie, "session_token=") || !strings.Contains(clearCookie, "Max-Age=0") {
		t.Fatalf("logout Set-Cookie = %q, expected clearing cookie", clearCookie)
	}

	reuseReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	reuseReq.AddCookie(sessionCookie)
	reuseRR := httptest.NewRecorder()
	router.ServeHTTP(reuseRR, reuseReq)

	if reuseRR.Code != http.StatusUnauthorized {
		t.Fatalf("reuse status = %d, want %d", reuseRR.Code, http.StatusUnauthorized)
	}

	assertErrorShape(t, reuseRR.Body.Bytes(), "unauthorized")
}

func TestMutatingAuthEndpointsWriteAuditEntries(t *testing.T) {
	router, auditStore := newTestRouterWithAudit(t, 24*time.Hour)

	loginBody := strings.NewReader(`{"email":"admin@pressluft.local","password":"pressluft-dev-password"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRR.Code, http.StatusOK)
	}

	cookie := loginRR.Result().Cookies()[0]
	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRR := httptest.NewRecorder()
	router.ServeHTTP(logoutRR, logoutReq)

	if logoutRR.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", logoutRR.Code, http.StatusOK)
	}

	entries, err := auditStore.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("audit entries len = %d, want 2", len(entries))
	}

	if entries[0].Action != "auth_login" || entries[0].Result != "succeeded" {
		t.Fatalf("login audit entry = %#v", entries[0])
	}
	if entries[1].Action != "auth_logout" || entries[1].Result != "succeeded" {
		t.Fatalf("logout audit entry = %#v", entries[1])
	}
}

func TestLoginReturnsInternalErrorWhenAuditWriteFails(t *testing.T) {
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", 24*time.Hour)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	nodeStore := nodes.NewInMemoryStore(nil)
	_, _ = nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.20",
		PublicIP:   "192.0.2.20",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
	})
	siteStore := store.NewInMemorySiteStore(0)
	_ = store.NewInMemoryBackupStore()
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	if err := os.Setenv("PRESSLUFT_DISABLE_RUNTIME_PROBES", "1"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("PRESSLUFT_DISABLE_RUNTIME_PROBES")
	})
	providerStore := providers.NewInMemoryStore(nil)
	router := NewRouter(logger, authService, jobStore, metricsService, failingAuditRecorder{}, nodeStore, providerStore)

	body := strings.NewReader(`{"email":"admin@pressluft.local","password":"pressluft-dev-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	assertErrorShape(t, rr.Body.Bytes(), "internal_error")
	if got := rr.Header().Get("Set-Cookie"); strings.Contains(got, "session_token=") {
		t.Fatalf("Set-Cookie = %q, expected no session cookie", got)
	}
}

func TestExpiredSessionRejectedOnProtectedEndpoint(t *testing.T) {
	router := newTestRouter(t, 10*time.Millisecond)
	sessionCookie := loginAndGetCookie(t, router)

	time.Sleep(25 * time.Millisecond)

	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(sessionCookie)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}

	assertErrorShape(t, recorder.Body.Bytes(), "unauthorized")
}

func TestJobsListReturnsStablePayloadShape(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode jobs list: %v", err)
	}

	if len(payload) != 3 {
		t.Fatalf("jobs count = %d, want 3", len(payload))
	}

	for _, job := range payload {
		if _, ok := job["id"]; !ok {
			t.Fatalf("job missing id: %#v", job)
		}
		if _, ok := job["status"]; !ok {
			t.Fatalf("job missing status: %#v", job)
		}
		if _, ok := job["attempt_count"]; !ok {
			t.Fatalf("job missing attempt_count: %#v", job)
		}
		if _, ok := job["max_attempts"]; !ok {
			t.Fatalf("job missing max_attempts: %#v", job)
		}
		if _, ok := job["created_at"]; !ok {
			t.Fatalf("job missing created_at: %#v", job)
		}
		if _, ok := job["updated_at"]; !ok {
			t.Fatalf("job missing updated_at: %#v", job)
		}
	}
}

func TestJobDetailIncludesAttemptsAndErrorFields(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/jobs/job-failed", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode job detail: %v", err)
	}

	if payload["id"] != "job-failed" {
		t.Fatalf("id = %v, want job-failed", payload["id"])
	}
	if payload["attempt_count"] != float64(3) {
		t.Fatalf("attempt_count = %v, want 3", payload["attempt_count"])
	}
	if payload["max_attempts"] != float64(3) {
		t.Fatalf("max_attempts = %v, want 3", payload["max_attempts"])
	}
	if payload["error_code"] != "node_unreachable" {
		t.Fatalf("error_code = %v, want node_unreachable", payload["error_code"])
	}
	if payload["error_message"] == "" {
		t.Fatal("error_message is empty")
	}
}

func TestMetricsReturnsNonNegativeCounters(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload struct {
		JobsRunning int `json:"jobs_running"`
		JobsQueued  int `json:"jobs_queued"`
		NodesActive int `json:"nodes_active"`
		SitesTotal  int `json:"sites_total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}

	if payload.JobsRunning < 0 || payload.JobsQueued < 0 || payload.NodesActive < 0 || payload.SitesTotal < 0 {
		t.Fatalf("metrics contain negative value: %+v", payload)
	}
}

func TestJobsAndMetricsRequireAuth(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)

	for _, path := range []string{"/jobs", "/jobs/job-running", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("path %s status = %d, want %d", path, rr.Code, http.StatusUnauthorized)
		}
		assertErrorShape(t, rr.Body.Bytes(), "unauthorized")
	}
}

func TestCreateSiteReturnsAcceptedJobAndPersistsSite(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Acme Co","slug":"acme"}`))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusAccepted, rr.Body.String())
	}

	var accepted map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode create site response: %v", err)
	}
	if accepted["job_id"] == "" {
		t.Fatal("job_id empty")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/sites", nil)
	listReq.AddCookie(sessionCookie)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRR.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list sites: %v", err)
	}
	if len(payload) < 1 {
		t.Fatal("sites list is empty")
	}

	id, _ := payload[0]["id"].(string)
	if id == "" {
		t.Fatal("site id missing")
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/sites/"+id, nil)
	detailReq.AddCookie(sessionCookie)
	detailRR := httptest.NewRecorder()
	router.ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detailRR.Code, http.StatusOK)
	}
}

func TestCreateSiteValidationAndConflict(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	invalidReq := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"","slug":""}`))
	invalidReq.AddCookie(sessionCookie)
	invalidRR := httptest.NewRecorder()
	router.ServeHTTP(invalidRR, invalidReq)

	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want %d", invalidRR.Code, http.StatusBadRequest)
	}
	assertErrorShape(t, invalidRR.Body.Bytes(), "bad_request")

	firstReq := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Acme Co","slug":"acme"}`))
	firstReq.AddCookie(sessionCookie)
	firstRR := httptest.NewRecorder()
	router.ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusAccepted {
		t.Fatalf("first create status = %d, want %d; body=%s", firstRR.Code, http.StatusAccepted, firstRR.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Acme Co","slug":"acme"}`))
	secondReq.AddCookie(sessionCookie)
	secondRR := httptest.NewRecorder()
	router.ServeHTTP(secondRR, secondReq)

	if secondRR.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want %d", secondRR.Code, http.StatusConflict)
	}
	assertErrorShape(t, secondRR.Body.Bytes(), "conflict")
}

func TestCreateSiteReturnsNodeNotReadyWhenPreflightFails(t *testing.T) {
	router := newTestRouterWithForcedReadiness(t, 24*time.Hour, nodes.ReadinessReport{
		IsReady:     false,
		ReasonCodes: []string{nodes.ReasonSudoUnavailable},
		Guidance:    []string{"configure passwordless sudo"},
	})
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodPost, "/sites", strings.NewReader(`{"name":"Acme Co","slug":"acme"}`))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	assertErrorShape(t, rr.Body.Bytes(), "node_not_ready")
}

func TestGetSiteByIDReturnsNotFoundForUnknownSite(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/sites/00000000-0000-0000-0000-000000000000", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	assertErrorShape(t, rr.Body.Bytes(), "not_found")
}

func TestCreateEnvironmentReturnsAcceptedAndSupportsListAndGet(t *testing.T) {
	router, siteID, primaryEnvironmentID := newTestRouterWithSeedSite(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	createReq := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Staging","slug":"staging","type":"staging","source_environment_id":"`+primaryEnvironmentID+`","promotion_preset":"content-protect"}`))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, want %d", createRR.Code, http.StatusAccepted)
	}

	var accepted map[string]string
	if err := json.Unmarshal(createRR.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode create environment response: %v", err)
	}
	if accepted["job_id"] == "" {
		t.Fatal("job_id empty")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/sites/"+siteID+"/environments", nil)
	listReq.AddCookie(sessionCookie)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRR.Code, http.StatusOK)
	}

	var environmentsPayload []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &environmentsPayload); err != nil {
		t.Fatalf("decode list environments: %v", err)
	}
	if len(environmentsPayload) != 2 {
		t.Fatalf("environments count = %d, want 2", len(environmentsPayload))
	}

	createdID, _ := environmentsPayload[1]["id"].(string)
	if createdID == "" {
		t.Fatal("created environment id missing")
	}
	if environmentsPayload[1]["status"] != "cloning" {
		t.Fatalf("created status = %v, want cloning", environmentsPayload[1]["status"])
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/environments/"+createdID, nil)
	detailReq.AddCookie(sessionCookie)
	detailRR := httptest.NewRecorder()
	router.ServeHTTP(detailRR, detailReq)

	if detailRR.Code != http.StatusOK {
		t.Fatalf("detail status = %d, want %d", detailRR.Code, http.StatusOK)
	}
}

func TestCreateEnvironmentValidationConflictAndNotFound(t *testing.T) {
	router, siteID, primaryEnvironmentID := newTestRouterWithSeedSite(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	invalidTypeReq := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Bad","slug":"bad","type":"production","source_environment_id":"`+primaryEnvironmentID+`","promotion_preset":"content-protect"}`))
	invalidTypeReq.AddCookie(sessionCookie)
	invalidTypeRR := httptest.NewRecorder()
	router.ServeHTTP(invalidTypeRR, invalidTypeReq)
	if invalidTypeRR.Code != http.StatusBadRequest {
		t.Fatalf("invalid type status = %d, want %d", invalidTypeRR.Code, http.StatusBadRequest)
	}

	missingSourceReq := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Bad","slug":"bad","type":"clone","promotion_preset":"content-protect"}`))
	missingSourceReq.AddCookie(sessionCookie)
	missingSourceRR := httptest.NewRecorder()
	router.ServeHTTP(missingSourceRR, missingSourceReq)
	if missingSourceRR.Code != http.StatusBadRequest {
		t.Fatalf("missing source status = %d, want %d", missingSourceRR.Code, http.StatusBadRequest)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Staging","slug":"staging","type":"staging","source_environment_id":"`+primaryEnvironmentID+`","promotion_preset":"content-protect"}`))
	firstReq.AddCookie(sessionCookie)
	firstRR := httptest.NewRecorder()
	router.ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusAccepted {
		t.Fatalf("first create status = %d, want %d", firstRR.Code, http.StatusAccepted)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Clone","slug":"clone-1","type":"clone","source_environment_id":"`+primaryEnvironmentID+`","promotion_preset":"content-protect"}`))
	secondReq.AddCookie(sessionCookie)
	secondRR := httptest.NewRecorder()
	router.ServeHTTP(secondRR, secondReq)
	if secondRR.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want %d", secondRR.Code, http.StatusConflict)
	}

	notFoundSiteReq := httptest.NewRequest(http.MethodGet, "/sites/00000000-0000-0000-0000-000000000000/environments", nil)
	notFoundSiteReq.AddCookie(sessionCookie)
	notFoundSiteRR := httptest.NewRecorder()
	router.ServeHTTP(notFoundSiteRR, notFoundSiteReq)
	if notFoundSiteRR.Code != http.StatusNotFound {
		t.Fatalf("site env list status = %d, want %d", notFoundSiteRR.Code, http.StatusNotFound)
	}

	notFoundEnvironmentReq := httptest.NewRequest(http.MethodGet, "/environments/00000000-0000-0000-0000-000000000000", nil)
	notFoundEnvironmentReq.AddCookie(sessionCookie)
	notFoundEnvironmentRR := httptest.NewRecorder()
	router.ServeHTTP(notFoundEnvironmentRR, notFoundEnvironmentReq)
	if notFoundEnvironmentRR.Code != http.StatusNotFound {
		t.Fatalf("environment detail status = %d, want %d", notFoundEnvironmentRR.Code, http.StatusNotFound)
	}
}

func TestCreateEnvironmentReturnsNodeNotReadyWhenPreflightFails(t *testing.T) {
	router, siteID, primaryEnvironmentID := newTestRouterWithForcedReadinessAndSeedSite(t, 24*time.Hour, nodes.ReadinessReport{
		IsReady:     false,
		ReasonCodes: []string{nodes.ReasonRuntimeMissing},
		Guidance:    []string{"install wp CLI"},
	})
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodPost, "/sites/"+siteID+"/environments", strings.NewReader(`{"name":"Staging","slug":"staging","type":"staging","source_environment_id":"`+primaryEnvironmentID+`","promotion_preset":"content-protect"}`))
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	assertErrorShape(t, rr.Body.Bytes(), "node_not_ready")
}

func TestCreateAndListEnvironmentBackups(t *testing.T) {
	router, _, environmentID := newTestRouterWithSeedSite(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	createReq := httptest.NewRequest(http.MethodPost, "/environments/"+environmentID+"/backups", strings.NewReader(`{"backup_scope":"full"}`))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, want %d", createRR.Code, http.StatusAccepted)
	}

	var accepted map[string]string
	if err := json.Unmarshal(createRR.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode create backup response: %v", err)
	}
	if accepted["job_id"] == "" {
		t.Fatal("job_id empty")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/environments/"+environmentID+"/backups", nil)
	listReq.AddCookie(sessionCookie)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRR.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list backups: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("backups count = %d, want 1", len(payload))
	}
	if payload[0]["status"] != "pending" {
		t.Fatalf("backup status = %v, want pending", payload[0]["status"])
	}
	if payload[0]["backup_scope"] != "full" {
		t.Fatalf("backup_scope = %v, want full", payload[0]["backup_scope"])
	}
	if payload[0]["retention_until"] == "" {
		t.Fatal("retention_until is empty")
	}
}

func TestListNodesReturnsAllRegisteredNodes(t *testing.T) {
	router := newTestRouterWithNodes(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode nodes list: %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("nodes count = %d, want 1", len(payload))
	}

	node := payload[0]
	if node["id"] == "" {
		t.Fatal("node id is empty")
	}
	if node["name"] == "" {
		t.Fatal("node name is empty")
	}
	if node["hostname"] == "" {
		t.Fatal("node hostname is empty")
	}
	if _, ok := node["status"]; !ok {
		t.Fatal("node status missing")
	}
	if _, ok := node["is_local"]; !ok {
		t.Fatal("node is_local missing")
	}
	if _, ok := node["created_at"]; !ok {
		t.Fatal("node created_at missing")
	}
	if _, ok := node["updated_at"]; !ok {
		t.Fatal("node updated_at missing")
	}
	if _, ok := node["readiness"]; !ok {
		t.Fatal("node readiness missing")
	}
}

func TestListNodesRequiresAuth(t *testing.T) {
	router := newTestRouterWithNodes(t, 24*time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	assertErrorShape(t, rr.Body.Bytes(), "unauthorized")
}

func TestListProvidersReturnsDisconnectedCatalog(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode providers list: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("providers list is empty")
	}

	provider := payload[0]
	if provider["provider_id"] != "hetzner" {
		t.Fatalf("provider_id = %v, want hetzner", provider["provider_id"])
	}
	if provider["status"] != "disconnected" {
		t.Fatalf("status = %v, want disconnected", provider["status"])
	}
	if provider["secret_configured"] != false {
		t.Fatalf("secret_configured = %v, want false", provider["secret_configured"])
	}
}

func TestConnectProviderPersistsMaskedConnection(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	connectReq := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(`{"provider_id":"hetzner","api_token":"bearer-token"}`))
	connectReq.AddCookie(sessionCookie)
	connectRR := httptest.NewRecorder()
	router.ServeHTTP(connectRR, connectReq)

	if connectRR.Code != http.StatusOK {
		t.Fatalf("connect status = %d, want %d", connectRR.Code, http.StatusOK)
	}

	var connectPayload map[string]any
	if err := json.Unmarshal(connectRR.Body.Bytes(), &connectPayload); err != nil {
		t.Fatalf("decode connect response: %v", err)
	}
	if connectPayload["status"] != "connected" {
		t.Fatalf("status = %v, want connected", connectPayload["status"])
	}
	if connectPayload["secret_configured"] != true {
		t.Fatalf("secret_configured = %v, want true", connectPayload["secret_configured"])
	}
	if _, ok := connectPayload["api_token"]; ok {
		t.Fatalf("response unexpectedly includes api_token")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/providers", nil)
	listReq.AddCookie(sessionCookie)
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRR.Code, http.StatusOK)
	}

	var listPayload []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode providers list: %v", err)
	}
	if len(listPayload) == 0 {
		t.Fatal("providers list is empty")
	}
	if listPayload[0]["secret_configured"] != true {
		t.Fatalf("secret_configured = %v, want true", listPayload[0]["secret_configured"])
	}
}

func TestCreateNodeReturnsConflictWhenProviderNotConnected(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	createReq := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(`{"provider_id":"hetzner","name":"edge-1"}`))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", createRR.Code, http.StatusConflict)
	}
	assertErrorShape(t, createRR.Body.Bytes(), "conflict")
}

func TestCreateNodeReturnsAcceptedAndEnqueuesProvisionJob(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	connectReq := httptest.NewRequest(http.MethodPost, "/providers", strings.NewReader(`{"provider_id":"hetzner","api_token":"bearer-token"}`))
	connectReq.AddCookie(sessionCookie)
	connectRR := httptest.NewRecorder()
	router.ServeHTTP(connectRR, connectReq)
	if connectRR.Code != http.StatusOK {
		t.Fatalf("connect status = %d, want %d", connectRR.Code, http.StatusOK)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(`{"provider_id":"hetzner","name":"edge-1"}`))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", createRR.Code, http.StatusAccepted)
	}

	var accepted map[string]string
	if err := json.Unmarshal(createRR.Body.Bytes(), &accepted); err != nil {
		t.Fatalf("decode accepted response: %v", err)
	}
	if accepted["job_id"] == "" {
		t.Fatal("job_id is empty")
	}

	nodesReq := httptest.NewRequest(http.MethodGet, "/nodes", nil)
	nodesReq.AddCookie(sessionCookie)
	nodesRR := httptest.NewRecorder()
	router.ServeHTTP(nodesRR, nodesReq)

	if nodesRR.Code != http.StatusOK {
		t.Fatalf("nodes status = %d, want %d", nodesRR.Code, http.StatusOK)
	}

	var nodesPayload []map[string]any
	if err := json.Unmarshal(nodesRR.Body.Bytes(), &nodesPayload); err != nil {
		t.Fatalf("decode nodes response: %v", err)
	}
	if len(nodesPayload) != 2 {
		t.Fatalf("nodes count = %d, want 2", len(nodesPayload))
	}

	var createdNode map[string]any
	for _, node := range nodesPayload {
		if node["name"] == "edge-1" {
			createdNode = node
			break
		}
	}
	if createdNode == nil {
		t.Fatalf("created provider node not found in payload: %+v", nodesPayload)
	}

	nodeID, _ := createdNode["id"].(string)
	if nodeID == "" {
		t.Fatal("node id is empty")
	}
	if createdNode["is_local"] != false {
		t.Fatalf("is_local = %v, want false", createdNode["is_local"])
	}
	if createdNode["hostname"] != "pending.provider" {
		t.Fatalf("hostname = %v, want pending.provider", createdNode["hostname"])
	}

	jobReq := httptest.NewRequest(http.MethodGet, "/jobs/"+accepted["job_id"], nil)
	jobReq.AddCookie(sessionCookie)
	jobRR := httptest.NewRecorder()
	router.ServeHTTP(jobRR, jobReq)

	if jobRR.Code != http.StatusOK {
		t.Fatalf("job status = %d, want %d", jobRR.Code, http.StatusOK)
	}

	var jobPayload map[string]any
	if err := json.Unmarshal(jobRR.Body.Bytes(), &jobPayload); err != nil {
		t.Fatalf("decode job response: %v", err)
	}
	if jobPayload["job_type"] != "node_provision" {
		t.Fatalf("job_type = %v, want node_provision", jobPayload["job_type"])
	}
	if jobPayload["node_id"] != nodeID {
		t.Fatalf("node_id = %v, want %s", jobPayload["node_id"], nodeID)
	}
}

func TestCreateNodeRejectsMissingProviderID(t *testing.T) {
	router := newTestRouter(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	createReq := httptest.NewRequest(http.MethodPost, "/nodes", strings.NewReader(`{"name":"edge-1"}`))
	createReq.AddCookie(sessionCookie)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", createRR.Code, http.StatusBadRequest)
	}
	assertErrorShape(t, createRR.Body.Bytes(), "bad_request")

}

func TestWordPressVersionQuerySuccess(t *testing.T) {
	router, _, environmentID := newTestRouterWithWordPressVersionAndIDs(t, 24*time.Hour, "6.4.3", nil)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/environments/"+environmentID+"/wordpress-version", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode wp version response: %v", err)
	}

	if payload["environment_id"] != environmentID {
		t.Fatalf("environment_id = %v, want %s", payload["environment_id"], environmentID)
	}
	if payload["wordpress_version"] != "6.4.3" {
		t.Fatalf("wordpress_version = %v", payload["wordpress_version"])
	}
	if payload["queried_at"] == "" {
		t.Fatal("queried_at is empty")
	}
}

func TestWordPressVersionQueryEnvironmentNotFound(t *testing.T) {
	router, _, _ := newTestRouterWithWordPressVersionAndIDs(t, 24*time.Hour, "", nil)
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/environments/00000000-0000-0000-0000-000000000000/wordpress-version", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	assertErrorShape(t, rr.Body.Bytes(), "not_found")
}

func TestWordPressVersionQueryNodeUnreachable(t *testing.T) {
	router, _, environmentID := newTestRouterWithWordPressVersionAndIDs(t, 24*time.Hour, "", errors.New("ssh timeout"))
	sessionCookie := loginAndGetCookie(t, router)

	req := httptest.NewRequest(http.MethodGet, "/environments/"+environmentID+"/wordpress-version", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
	assertErrorShape(t, rr.Body.Bytes(), "node_unreachable")
}

func TestCreateEnvironmentBackupValidationNotFoundAndConflict(t *testing.T) {
	router, _, environmentID := newTestRouterWithSeedSite(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)

	invalidReq := httptest.NewRequest(http.MethodPost, "/environments/"+environmentID+"/backups", strings.NewReader(`{"backup_scope":"invalid"}`))
	invalidReq.AddCookie(sessionCookie)
	invalidRR := httptest.NewRecorder()
	router.ServeHTTP(invalidRR, invalidReq)
	if invalidRR.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want %d", invalidRR.Code, http.StatusBadRequest)
	}

	notFoundReq := httptest.NewRequest(http.MethodPost, "/environments/00000000-0000-0000-0000-000000000000/backups", strings.NewReader(`{"backup_scope":"full"}`))
	notFoundReq.AddCookie(sessionCookie)
	notFoundRR := httptest.NewRecorder()
	router.ServeHTTP(notFoundRR, notFoundReq)
	if notFoundRR.Code != http.StatusNotFound {
		t.Fatalf("not found status = %d, want %d", notFoundRR.Code, http.StatusNotFound)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/environments/"+environmentID+"/backups", strings.NewReader(`{"backup_scope":"full"}`))
	firstReq.AddCookie(sessionCookie)
	firstRR := httptest.NewRecorder()
	router.ServeHTTP(firstRR, firstReq)
	if firstRR.Code != http.StatusAccepted {
		t.Fatalf("first create status = %d, want %d", firstRR.Code, http.StatusAccepted)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/environments/"+environmentID+"/backups", strings.NewReader(`{"backup_scope":"db"}`))
	secondReq.AddCookie(sessionCookie)
	secondRR := httptest.NewRecorder()
	router.ServeHTTP(secondRR, secondReq)
	if secondRR.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want %d", secondRR.Code, http.StatusConflict)
	}
}

func TestRestoreEnvironmentReturnsAcceptedJob(t *testing.T) {
	router, _, environmentID := newTestRouterWithSeedSite(t, 24*time.Hour)
	sessionCookie := loginAndGetCookie(t, router)
	backupID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	backupStore, ok := store.DefaultBackupStore().(*store.InMemoryBackupStore)
	if !ok {
		t.Fatal("default backup store type assertion failed")
	}
	now := time.Now().UTC()
	_, _ = backupStore.CreateBackup(context.Background(), store.CreateBackupInput{
		ID:             backupID,
		EnvironmentID:  environmentID,
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/" + environmentID + "/" + backupID + ".tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	_, _ = backupStore.MarkBackupRunning(context.Background(), backupID, now)
	_, _ = backupStore.MarkBackupCompleted(context.Background(), backupID, "sha256:test", 128, now)

	restoreReq2 := httptest.NewRequest(http.MethodPost, "/environments/"+environmentID+"/restore", strings.NewReader(`{"backup_id":"`+backupID+`"}`))
	restoreReq2.AddCookie(sessionCookie)
	restoreRR2 := httptest.NewRecorder()
	router.ServeHTTP(restoreRR2, restoreReq2)
	if restoreRR2.Code != http.StatusAccepted {
		t.Fatalf("restore status = %d, want %d", restoreRR2.Code, http.StatusAccepted)
	}
}

func newTestRouter(t *testing.T, sessionTTL time.Duration) http.Handler {
	t.Helper()
	router, _ := newTestRouterWithAudit(t, sessionTTL)
	return router
}

func newTestRouterWithAudit(t *testing.T, sessionTTL time.Duration) (http.Handler, *audit.InMemoryStore) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	nodeStore := nodes.NewInMemoryStore(nil)
	_, _ = nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.20",
		PublicIP:   "192.0.2.20",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
	})
	siteStore := store.NewInMemorySiteStore(0)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	if err := os.Setenv("PRESSLUFT_DISABLE_RUNTIME_PROBES", "1"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("PRESSLUFT_DISABLE_RUNTIME_PROBES")
	})
	providerStore := providers.NewInMemoryStore(nil)
	return NewRouter(logger, authService, jobStore, metricsService, auditService, nodeStore, providerStore), auditStore
}

func newTestRouterWithSeedSite(t *testing.T, sessionTTL time.Duration) (http.Handler, string, string) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	nodeStore := nodes.NewInMemoryStore(nil)
	siteStore := store.NewInMemorySiteStore(0)
	_ = store.NewInMemoryBackupStore()
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.25",
		PublicIP:   "192.0.2.25",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        now,
	})
	site, environment, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme Co",
		Slug:       "acme",
		NodeID:     providerNode.ID,
		NodePublic: providerNode.PublicIP,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	if err := os.Setenv("PRESSLUFT_DISABLE_RUNTIME_PROBES", "1"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("PRESSLUFT_DISABLE_RUNTIME_PROBES")
	})
	providerStore := providers.NewInMemoryStore(nil)
	router := NewRouter(logger, authService, jobStore, metricsService, auditService, nodeStore, providerStore)
	return router, site.ID, environment.ID
}

func newTestRouterWithNodes(t *testing.T, sessionTTL time.Duration) http.Handler {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())

	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	nodeStore := nodes.NewInMemoryStore([]nodes.Node{
		{
			ID:        nodes.SelfNodeID,
			Name:      nodes.SelfNodeName,
			Hostname:  "127.0.0.1",
			PublicIP:  "127.0.0.1",
			SSHPort:   22,
			SSHUser:   "pressluft",
			Status:    nodes.StatusActive,
			IsLocal:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	})

	siteStore := store.NewInMemorySiteStore(0)
	_ = store.NewInMemoryBackupStore()
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	if err := os.Setenv("PRESSLUFT_DISABLE_RUNTIME_PROBES", "1"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("PRESSLUFT_DISABLE_RUNTIME_PROBES")
	})
	providerStore := providers.NewInMemoryStore(nil)
	return NewRouter(logger, authService, jobStore, metricsService, auditService, nodeStore, providerStore)
}

type mockSSHRunner struct {
	version string
	err     error
}

type forcedReadinessChecker struct {
	report nodes.ReadinessReport
}

func (f forcedReadinessChecker) Evaluate(context.Context, nodes.Node) (nodes.ReadinessReport, error) {
	return f.report, nil
}

func (m *mockSSHRunner) WordPressVersion(ctx context.Context, host string, port int, user string, siteSlug string, envSlug string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.version, nil
}

func (m *mockSSHRunner) CheckNodePrerequisites(ctx context.Context, host string, port int, user string, isLocal bool) ([]string, error) {
	_ = ctx
	_ = host
	_ = port
	_ = user
	_ = isLocal
	return nil, nil
}

func newTestRouterWithWordPressVersionAndIDs(t *testing.T, sessionTTL time.Duration, version string, queryErr error) (http.Handler, string, string) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())

	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	nodeID := "44444444-4444-4444-4444-444444444444"
	nodeStore := nodes.NewInMemoryStore([]nodes.Node{
		{
			ID:        nodeID,
			Name:      "test-node",
			Hostname:  "127.0.0.1",
			PublicIP:  "127.0.0.1",
			SSHPort:   22,
			SSHUser:   "pressluft",
			Status:    nodes.StatusActive,
			IsLocal:   true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	})

	siteStore := store.NewInMemorySiteStore(0)
	_ = store.NewInMemoryBackupStore()
	site, environment, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Test Site",
		Slug:       "testsite",
		NodeID:     nodeID,
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)

	readiness := nodes.NewReadinessChecker(&mockSSHRunner{})
	siteService := sites.NewService(siteStore, nodeStore, jobStore, readiness)
	envService := environments.NewService(siteStore, jobStore, nodeStore, readiness)
	backupService := backups.NewService(siteStore, store.DefaultBackupStore(), jobStore)

	router := &Router{
		logger:       logger,
		authService:  authService,
		jobs:         jobStore,
		metrics:      metricsService,
		auditor:      auditService,
		nodeStore:    nodeStore,
		sites:        siteService,
		environments: envService,
		backups:      backupService,
		restores:     environments.NewRestoreService(siteStore, store.DefaultBackupStore(), store.DefaultRestoreRequestStore(), jobStore),
		sshRunner:    &mockSSHRunner{version: version, err: queryErr},
	}

	t.Logf("Created site %s with environment %s (status=%s)", site.ID, environment.ID, environment.Status)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", router.handleLogin)
	mux.HandleFunc("/logout", router.handleLogout)
	mux.HandleFunc("/providers", router.handleProviders)
	mux.HandleFunc("/nodes", router.handleNodes)
	mux.HandleFunc("/jobs", router.handleJobsList)
	mux.HandleFunc("/jobs/", router.handleJobDetail)
	mux.HandleFunc("/metrics", router.handleMetrics)
	mux.HandleFunc("/sites", router.handleSites)
	mux.HandleFunc("/sites/", router.handleSiteByID)
	mux.HandleFunc("/environments/", router.handleEnvironmentByID)

	return router.withAuth(mux), site.ID, environment.ID
}

func newTestRouterWithForcedReadiness(t *testing.T, sessionTTL time.Duration, report nodes.ReadinessReport) http.Handler {
	t.Helper()
	router, _, _ := newTestRouterWithForcedReadinessAndSeedSite(t, sessionTTL, report)
	return router
}

func newTestRouterWithForcedReadinessAndSeedSite(t *testing.T, sessionTTL time.Duration, report nodes.ReadinessReport) (http.Handler, string, string) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	nodeStore := nodes.NewInMemoryStore(nil)
	providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.26",
		PublicIP:   "192.0.2.26",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        now,
	})

	siteStore := store.NewInMemorySiteStore(0)
	_ = store.NewInMemoryBackupStore()
	site, environment, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme Co",
		Slug:       "acme",
		NodeID:     providerNode.ID,
		NodePublic: providerNode.PublicIP,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	readiness := forcedReadinessChecker{report: report}

	router := &Router{
		logger:       logger,
		authService:  authService,
		jobs:         jobStore,
		metrics:      metricsService,
		auditor:      auditService,
		nodeStore:    nodeStore,
		sites:        sites.NewService(siteStore, nodeStore, jobStore, readiness),
		environments: environments.NewService(siteStore, jobStore, nodeStore, readiness),
		backups:      backups.NewService(siteStore, store.DefaultBackupStore(), jobStore),
		restores:     environments.NewRestoreService(siteStore, store.DefaultBackupStore(), store.DefaultRestoreRequestStore(), jobStore),
		sshRunner:    &mockSSHRunner{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", router.handleLogin)
	mux.HandleFunc("/logout", router.handleLogout)
	mux.HandleFunc("/providers", router.handleProviders)
	mux.HandleFunc("/nodes", router.handleNodes)
	mux.HandleFunc("/jobs", router.handleJobsList)
	mux.HandleFunc("/jobs/", router.handleJobDetail)
	mux.HandleFunc("/metrics", router.handleMetrics)
	mux.HandleFunc("/sites", router.handleSites)
	mux.HandleFunc("/sites/", router.handleSiteByID)
	mux.HandleFunc("/environments/", router.handleEnvironmentByID)

	return router.withAuth(mux), site.ID, environment.ID
}

type failingAuditRecorder struct{}

func (failingAuditRecorder) Record(context.Context, audit.Entry) error {
	return errors.New("audit unavailable")
}

func (failingAuditRecorder) RecordAsyncAccepted(context.Context, audit.Entry) error {
	return errors.New("audit unavailable")
}

func (failingAuditRecorder) UpdateAsyncResult(context.Context, string, string, string, string) error {
	return errors.New("audit unavailable")
}

func loginAndGetCookie(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	body := strings.NewReader(`{"email":"admin@pressluft.local","password":"pressluft-dev-password"}`)
	request := httptest.NewRequest(http.MethodPost, "/login", body)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()

	for _, cookie := range result.Cookies() {
		if cookie.Name == auth.CookieName {
			return cookie
		}
	}

	t.Fatal("session cookie not set")
	return nil
}

func assertErrorShape(t *testing.T, body []byte, expectedCode string) {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error body: %v", err)
	}

	if payload["code"] != expectedCode {
		t.Fatalf("code = %q, want %q", payload["code"], expectedCode)
	}

	if payload["message"] == "" {
		t.Fatal("message is empty")
	}
}

func seedTestJobs() []jobs.Job {
	now := time.Now().UTC()
	siteID := "11111111-1111-1111-1111-111111111111"
	environmentID := "22222222-2222-2222-2222-222222222222"
	nodeID := "33333333-3333-3333-3333-333333333333"
	workerID := "worker-1"
	errorCode := "node_unreachable"
	errorMessage := "ssh timeout while provisioning node"

	runningStarted := now.Add(-90 * time.Second)
	failedStarted := now.Add(-4 * time.Minute)
	failedFinished := now.Add(-3 * time.Minute)

	return []jobs.Job{
		{
			ID:            "job-queued",
			JobType:       "node_provision",
			Status:        jobs.StatusQueued,
			SiteID:        nil,
			EnvironmentID: nil,
			NodeID:        &nodeID,
			AttemptCount:  0,
			MaxAttempts:   3,
			CreatedAt:     now.Add(-5 * time.Minute),
			UpdatedAt:     now.Add(-5 * time.Minute),
		},
		{
			ID:            "job-running",
			JobType:       "environment_deploy",
			Status:        jobs.StatusRunning,
			SiteID:        &siteID,
			EnvironmentID: &environmentID,
			NodeID:        &nodeID,
			AttemptCount:  1,
			MaxAttempts:   3,
			LockedAt:      &runningStarted,
			LockedBy:      &workerID,
			StartedAt:     &runningStarted,
			CreatedAt:     now.Add(-2 * time.Minute),
			UpdatedAt:     now.Add(-1 * time.Minute),
		},
		{
			ID:            "job-failed",
			JobType:       "environment_deploy",
			Status:        jobs.StatusFailed,
			SiteID:        &siteID,
			EnvironmentID: &environmentID,
			NodeID:        &nodeID,
			AttemptCount:  3,
			MaxAttempts:   3,
			LockedAt:      &failedStarted,
			LockedBy:      &workerID,
			StartedAt:     &failedStarted,
			FinishedAt:    &failedFinished,
			ErrorCode:     &errorCode,
			ErrorMessage:  &errorMessage,
			CreatedAt:     now.Add(-4 * time.Minute),
			UpdatedAt:     now.Add(-3 * time.Minute),
		},
	}
}
