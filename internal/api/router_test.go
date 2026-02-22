package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
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
	nodeStore := store.NewInMemoryNodeStore(2)
	siteStore := store.NewInMemorySiteStore(0)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	router := NewRouter(logger, authService, jobStore, metricsService, failingAuditRecorder{})

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
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
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
		t.Fatalf("first create status = %d, want %d", firstRR.Code, http.StatusAccepted)
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
	nodeStore := store.NewInMemoryNodeStore(2)
	siteStore := store.NewInMemorySiteStore(0)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	return NewRouter(logger, authService, jobStore, metricsService, auditService), auditStore
}

func newTestRouterWithSeedSite(t *testing.T, sessionTTL time.Duration) (http.Handler, string, string) {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	nodeStore := store.NewInMemoryNodeStore(2)
	siteStore := store.NewInMemorySiteStore(0)
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	site, environment, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme Co",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)
	router := NewRouter(logger, authService, jobStore, metricsService, auditService)
	return router, site.ID, environment.ID
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
