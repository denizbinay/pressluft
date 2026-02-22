package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func newTestRouter(t *testing.T, sessionTTL time.Duration) http.Handler {
	t.Helper()
	logger := log.New(&bytes.Buffer{}, "", 0)
	sessionStore := store.NewInMemorySessionStore()
	authService := auth.NewService(sessionStore, "admin@pressluft.local", "pressluft-dev-password", sessionTTL)
	jobStore := jobs.NewInMemoryRepository(seedTestJobs())
	nodeStore := store.NewInMemoryNodeStore(2)
	siteStore := store.NewInMemorySiteStore(5)
	metricsService := metrics.NewService(jobStore, nodeStore, siteStore)
	return NewRouter(logger, authService, jobStore, metricsService)
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
