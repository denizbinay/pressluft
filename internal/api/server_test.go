package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/domains"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/migration"
	"pressluft/internal/promotion"
	"pressluft/internal/secrets"
	"pressluft/internal/settings"
	"pressluft/internal/sites"
	"pressluft/internal/ssh"
	"pressluft/internal/store"
)

func TestLoginSuccessSetsCookieAndCreatesSession(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"secret"}`))
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	cookie := getCookie(rec.Result().Cookies(), "session_token")
	if cookie == nil || cookie.Value == "" {
		t.Fatalf("expected session_token cookie")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM auth_sessions").Scan(&count); err != nil {
		t.Fatalf("query auth_sessions count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one auth session, got %d", count)
	}

	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'auth.login' AND result = 'success'").Scan(&count); err != nil {
		t.Fatalf("query audit_logs count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one login audit log, got %d", count)
	}
}

func TestLoginInvalidCredentialsReturnsUnauthorizedErrorShape(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`))
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] == "" || body["message"] == "" {
		t.Fatalf("expected canonical error payload with code and message")
	}
}

func TestLogoutRevokesSessionAndClearsCookie(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	logoutCookie := getCookie(rec.Result().Cookies(), "session_token")
	if logoutCookie == nil || logoutCookie.MaxAge != -1 {
		t.Fatalf("expected clearing cookie with MaxAge=-1")
	}

	var revoked sql.NullString
	if err := db.QueryRow("SELECT revoked_at FROM auth_sessions WHERE session_token = ?", token).Scan(&revoked); err != nil {
		t.Fatalf("query revoked_at: %v", err)
	}
	if !revoked.Valid {
		t.Fatalf("expected session to be revoked")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'auth.logout' AND result = 'success'").Scan(&count); err != nil {
		t.Fatalf("query logout audit logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one logout audit log, got %d", count)
	}
}

func TestProtectedEndpointRejectsRevokedSession(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec("UPDATE auth_sessions SET revoked_at = datetime('now') WHERE session_token = ?", token); err != nil {
		t.Fatalf("revoke auth session: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestDashboardFallbackServesIndexForClientRoutes(t *testing.T) {
	t.Parallel()

	server, _ := newTestServerWithDashboardFS(t, fstest.MapFS{
		"index.html":       {Data: []byte("<html>dashboard</html>")},
		"_nuxt/app.js":     {Data: []byte("console.log('asset')")},
		"_nuxt/styles.css": {Data: []byte("body{margin:0}")},
		"favicon.ico":      {Data: []byte("icon")},
	})

	clientRoute := httptest.NewRecorder()
	clientRouteReq := httptest.NewRequest(http.MethodGet, "/app/sites/site-1", nil)
	server.Handler().ServeHTTP(clientRoute, clientRouteReq)

	if clientRoute.Code != http.StatusOK {
		t.Fatalf("unexpected client route status: %d", clientRoute.Code)
	}
	if !strings.Contains(clientRoute.Body.String(), "dashboard") {
		t.Fatalf("expected SPA fallback body")
	}

	assetRec := httptest.NewRecorder()
	assetReq := httptest.NewRequest(http.MethodGet, "/_nuxt/app.js", nil)
	server.Handler().ServeHTTP(assetRec, assetReq)

	if assetRec.Code != http.StatusOK {
		t.Fatalf("unexpected asset status: %d", assetRec.Code)
	}
	if !strings.Contains(assetRec.Body.String(), "asset") {
		t.Fatalf("expected static asset body")
	}
}

func TestDashboardFallbackDoesNotOverrideAPIOrAdminRoutes(t *testing.T) {
	t.Parallel()

	server, _ := newTestServerWithDashboardFS(t, fstest.MapFS{
		"index.html": {Data: []byte("<html>dashboard</html>")},
	})

	apiRec := httptest.NewRecorder()
	apiReq := httptest.NewRequest(http.MethodGet, "/api/login", nil)
	server.Handler().ServeHTTP(apiRec, apiReq)

	if apiRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected api route status: %d", apiRec.Code)
	}
	if !strings.Contains(apiRec.Body.String(), "method_not_allowed") {
		t.Fatalf("expected API json error response")
	}

	adminRec := httptest.NewRecorder()
	adminReq := httptest.NewRequest(http.MethodGet, "/_admin/settings/domain-config", nil)
	server.Handler().ServeHTTP(adminRec, adminReq)

	if adminRec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected admin route status: %d", adminRec.Code)
	}
	if !strings.Contains(adminRec.Body.String(), "auth_unauthorized") {
		t.Fatalf("expected admin auth error response")
	}
}

func TestCreateSiteReturnsAcceptedAndPersistsRecords(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewBufferString(`{"name":"Acme","slug":"acme"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	var siteID, status, primaryEnvironmentID string
	if err := db.QueryRow("SELECT id, status, primary_environment_id FROM sites WHERE slug = 'acme'").Scan(&siteID, &status, &primaryEnvironmentID); err != nil {
		t.Fatalf("query site: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected site status active, got %s", status)
	}

	var envType, envStatus, previewURL string
	if err := db.QueryRow(`
		SELECT environment_type, status, preview_url
		FROM environments
		WHERE id = ?
	`, primaryEnvironmentID).Scan(&envType, &envStatus, &previewURL); err != nil {
		t.Fatalf("query environment: %v", err)
	}
	if envType != "production" || envStatus != "active" {
		t.Fatalf("unexpected environment values type=%s status=%s", envType, envStatus)
	}
	if !strings.HasPrefix(previewURL, "http://") || !strings.Contains(previewURL, ".sslip.io") {
		t.Fatalf("unexpected preview url: %s", previewURL)
	}

	var jobType, jobStatus, jobSiteID string
	if err := db.QueryRow("SELECT job_type, status, site_id FROM jobs WHERE id = ?", body["job_id"]).Scan(&jobType, &jobStatus, &jobSiteID); err != nil {
		t.Fatalf("query job: %v", err)
	}
	if jobType != "site_create" || jobStatus != "queued" || jobSiteID != siteID {
		t.Fatalf("unexpected job record type=%s status=%s site=%s", jobType, jobStatus, jobSiteID)
	}

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'site.create' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query site create audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one site.create audit log, got %d", auditCount)
	}
}

func TestCreateSiteRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewBufferString(`{"name":"Acme","slug":""}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestGetSiteReturnsPersistedShape(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewBufferString(`{"name":"Acme","slug":"acme"}`))
	createReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status: %d", createRec.Code)
	}

	var siteID string
	if err := db.QueryRow("SELECT id FROM sites WHERE slug = 'acme'").Scan(&siteID); err != nil {
		t.Fatalf("query site id: %v", err)
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID, nil)
	getReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getRec.Code)
	}

	var siteResp map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &siteResp); err != nil {
		t.Fatalf("decode get site response: %v", err)
	}
	if siteResp["id"] != siteID {
		t.Fatalf("expected site id %s, got %v", siteID, siteResp["id"])
	}
	if siteResp["slug"] != "acme" {
		t.Fatalf("expected slug acme, got %v", siteResp["slug"])
	}
}

func TestCreateEnvironmentReturnsAcceptedAndSetsCloningState(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	siteID, sourceEnvironmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/environments", bytes.NewBufferString(`{"name":"Staging","slug":"staging","type":"staging","source_environment_id":"`+sourceEnvironmentID+`","promotion_preset":"content-protect"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	var siteStatus string
	if err := db.QueryRow("SELECT status FROM sites WHERE id = ?", siteID).Scan(&siteStatus); err != nil {
		t.Fatalf("query site status: %v", err)
	}
	if siteStatus != "cloning" {
		t.Fatalf("expected site status cloning, got %s", siteStatus)
	}

	var envID, envStatus, envType string
	if err := db.QueryRow("SELECT id, status, environment_type FROM environments WHERE site_id = ? AND slug = 'staging'", siteID).Scan(&envID, &envStatus, &envType); err != nil {
		t.Fatalf("query created environment: %v", err)
	}
	if envStatus != "cloning" || envType != "staging" {
		t.Fatalf("unexpected environment values status=%s type=%s", envStatus, envType)
	}

	var jobType, jobStatus string
	if err := db.QueryRow("SELECT job_type, status FROM jobs WHERE id = ?", body["job_id"]).Scan(&jobType, &jobStatus); err != nil {
		t.Fatalf("query env job: %v", err)
	}
	if jobType != "env_create" || jobStatus != "queued" {
		t.Fatalf("unexpected job values type=%s status=%s", jobType, jobStatus)
	}

	_ = envID
}

func TestEnvironmentListAndGetReturnRepresentations(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	siteID, sourceEnvironmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/environments", bytes.NewBufferString(`{"name":"Staging","slug":"staging","type":"staging","source_environment_id":"`+sourceEnvironmentID+`","promotion_preset":"content-protect"}`))
	createReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status: %d", createRec.Code)
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/environments", nil)
	listReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: %d", listRec.Code)
	}

	var listed []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(listed))
	}

	var stagingID string
	if err := db.QueryRow("SELECT id FROM environments WHERE site_id = ? AND slug = 'staging'", siteID).Scan(&stagingID); err != nil {
		t.Fatalf("query staging id: %v", err)
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/environments/"+stagingID, nil)
	getReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status: %d", getRec.Code)
	}

	var envResp map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &envResp); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if envResp["id"] != stagingID {
		t.Fatalf("expected id %s, got %v", stagingID, envResp["id"])
	}
	if envResp["environment_type"] != "staging" {
		t.Fatalf("expected staging type, got %v", envResp["environment_type"])
	}
}

func TestCreateEnvironmentRejectsInvalidType(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	siteID, sourceEnvironmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/environments", bytes.NewBufferString(`{"name":"Prod2","slug":"prod2","type":"production","source_environment_id":"`+sourceEnvironmentID+`","promotion_preset":"content-protect"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestImportSiteReturnsAcceptedAndQueuesImportJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	siteID, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/import", bytes.NewBufferString(`{"archive_url":"https://example.com/archive.tar.gz"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "site_import", body["job_id"])
	assertDBString(t, db, "SELECT status FROM sites WHERE id = ?", "restoring", siteID)
	assertDBString(t, db, "SELECT status FROM environments WHERE id = ?", "restoring", environmentID)

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'site.import' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query site.import audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one site.import audit log, got %d", auditCount)
	}
}

func TestImportSiteRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	siteID, _ := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/import", bytes.NewBufferString(`{"archive_url":"file:///tmp/archive.tar.gz"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestUpdateDomainConfigSettingsPersistsAndRedactsSecret(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/_admin/settings/domain-config", bytes.NewBufferString(`{
		"control_plane_domain":"panel.example.com",
		"preview_domain":"wp.example.com",
		"dns01_provider":"cloudflare",
		"dns01_credentials_json":{"CF_DNS_API_TOKEN":"test-token"}
	}`))
	updateReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d", updateRec.Code)
	}

	var updateBody map[string]any
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateBody); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateBody["dns01_credentials_json"] != nil {
		t.Fatalf("expected redacted dns01_credentials_json field")
	}
	if configured, ok := updateBody["dns01_credentials_configured"].(bool); !ok || !configured {
		t.Fatalf("expected dns01_credentials_configured=true")
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/_admin/settings/domain-config", nil)
	getReq.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", getRec.Code)
	}

	var getBody map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getBody["control_plane_domain"] != "panel.example.com" {
		t.Fatalf("unexpected control_plane_domain: %v", getBody["control_plane_domain"])
	}
	if getBody["preview_domain"] != "wp.example.com" {
		t.Fatalf("unexpected preview_domain: %v", getBody["preview_domain"])
	}
	if getBody["dns01_provider"] != "cloudflare" {
		t.Fatalf("unexpected dns01_provider: %v", getBody["dns01_provider"])
	}
	if getBody["dns01_credentials_json"] != nil {
		t.Fatalf("expected redacted dns01_credentials_json in get response")
	}
}

func TestUpdateDomainConfigSettingsRejectsInvalidCombination(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/_admin/settings/domain-config", bytes.NewBufferString(`{
		"control_plane_domain":"panel.example.com",
		"preview_domain":"wp.example.com",
		"dns01_provider":null,
		"dns01_credentials_json":null
	}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestJobsAndMetricsEndpointsRequireAuth(t *testing.T) {
	t.Parallel()

	server, _ := newTestServer(t)

	for _, path := range []string{"/api/jobs", "/api/jobs/job-1", "/api/metrics"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected unauthorized for %s, got %d", path, rec.Code)
		}
	}
}

func TestListJobsReturnsStablePayloadShape(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES
			('job-queued', 'site_create', 'queued', 'site-a', NULL, 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '2026-02-21T10:00:00Z', '2026-02-21T10:00:00Z'),
			('job-running', 'env_deploy', 'running', 'site-a', 'env-a', 'node-1', '{}', 1, 3, NULL, '2026-02-21T10:05:00Z', 'worker-1', '2026-02-21T10:05:00Z', NULL, NULL, NULL, '2026-02-21T10:04:00Z', '2026-02-21T10:05:00Z')
	`); err != nil {
		t.Fatalf("seed jobs: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var jobsList []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &jobsList); err != nil {
		t.Fatalf("decode jobs response: %v", err)
	}
	if len(jobsList) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobsList))
	}

	first := jobsList[0]
	if first["id"] == nil || first["status"] == nil || first["attempt_count"] == nil || first["max_attempts"] == nil {
		t.Fatalf("expected stable job payload shape")
	}
}

func TestGetJobReturnsFullState(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (
			'job-failed', 'site_import', 'failed', 'site-a', 'env-a', 'node-1', '{}',
			3, 3, '2026-02-21T10:00:00Z', '2026-02-21T10:01:00Z', 'worker-1',
			'2026-02-21T10:01:00Z', '2026-02-21T10:02:00Z', 'SITE_IMPORT_FAILED', 'import failed',
			'2026-02-21T10:00:00Z', '2026-02-21T10:02:00Z'
		)
	`); err != nil {
		t.Fatalf("seed failed job: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/jobs/job-failed", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["attempt_count"] != float64(3) || body["max_attempts"] != float64(3) {
		t.Fatalf("unexpected attempts fields: %v/%v", body["attempt_count"], body["max_attempts"])
	}
	if body["error_code"] != "SITE_IMPORT_FAILED" || body["error_message"] != "import failed" {
		t.Fatalf("unexpected error fields: code=%v message=%v", body["error_code"], body["error_message"])
	}
}

func TestMetricsReturnsPointInTimeCounters(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES
			('node-1', 'node-1', 'localhost', '203.0.113.10', 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1),
			('node-2', 'node-2', 'localhost', '203.0.113.11', 22, 'root', 'unreachable', 0, NULL, datetime('now'), datetime('now'), 1);
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES
			('site-1', 'Site 1', 'site-1', 'active', NULL, datetime('now'), datetime('now'), 1),
			('site-2', 'Site 2', 'site-2', 'active', NULL, datetime('now'), datetime('now'), 1);
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES
			('job-running', 'env_deploy', 'running', 'site-1', NULL, 'node-1', '{}', 1, 3, NULL, NULL, NULL, '2026-02-21T10:01:00Z', NULL, NULL, NULL, '2026-02-21T10:00:00Z', '2026-02-21T10:01:00Z'),
			('job-queued', 'site_create', 'queued', 'site-2', NULL, 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '2026-02-21T10:00:00Z', '2026-02-21T10:00:00Z');
	`); err != nil {
		t.Fatalf("seed metrics data: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["jobs_running"] != float64(1) || body["jobs_queued"] != float64(1) || body["nodes_active"] != float64(1) || body["sites_total"] != float64(2) {
		t.Fatalf("unexpected metrics payload: %+v", body)
	}
}

func TestJobControlEndpointsRequireAuth(t *testing.T) {
	t.Parallel()

	server, _ := newTestServer(t)

	for _, endpoint := range []string{"/api/jobs/job-1/cancel", "/api/sites/site-1/reset", "/api/environments/env-1/reset"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, endpoint, nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected unauthorized for %s, got %d", endpoint, rec.Code)
		}
	}
}

func TestCancelJobReturnsSuccessForQueuedJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-cancel', 'site_create', 'queued', 'site-1', NULL, 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed queued job: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/job-cancel/cancel", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	assertDBString(t, db, "SELECT status FROM jobs WHERE id = ?", "cancelled", "job-cancel")
}

func TestCancelJobReturnsConflictWhenStateNotCancellable(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-done', 'site_create', 'succeeded', 'site-1', NULL, 'node-1', '{}', 1, 3, NULL, NULL, NULL, datetime('now'), datetime('now'), NULL, NULL, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed succeeded job: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/job-done/cancel", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "job_not_cancellable" {
		t.Fatalf("expected job_not_cancellable, got %q", body["code"])
	}
}

func TestResetSiteReturnsSuccess(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)
	siteID, _ := seedSiteWithProductionEnvironment(t, db, "acme")

	if _, err := db.Exec(`UPDATE sites SET status = 'failed' WHERE id = ?`, siteID); err != nil {
		t.Fatalf("set site failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/reset", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	assertDBString(t, db, "SELECT status FROM sites WHERE id = ?", "active", siteID)
}

func TestResetEnvironmentReturnsSuccess(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	token := loginForToken(t, server)
	siteID, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	if _, err := db.Exec(`UPDATE sites SET status = 'failed' WHERE id = ?`, siteID); err != nil {
		t.Fatalf("set site failed: %v", err)
	}
	if _, err := db.Exec(`UPDATE environments SET status = 'failed' WHERE id = ?`, environmentID); err != nil {
		t.Fatalf("set site/environment failed: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/reset", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	assertDBString(t, db, "SELECT status FROM environments WHERE id = ?", "active", environmentID)
}

func TestCreateBackupReturnsAcceptedAndQueuesBackupJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/backups", bytes.NewBufferString(`{"backup_scope":"full"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "backup_create", body["job_id"])
	assertDBString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", body["job_id"])

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'backup.create' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query backup.create audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one backup.create audit log, got %d", auditCount)
	}
}

func TestDeployEnvironmentReturnsAcceptedAndQueuesDeployJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/deploy", bytes.NewBufferString(`{"source_type":"git","source_ref":"git@github.com:acme/site.git#main"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_deploy", body["job_id"])
	assertDBString(t, db, "SELECT status FROM environments WHERE id = ?", "deploying", environmentID)

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.deploy' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.deploy audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.deploy audit log, got %d", auditCount)
	}
}

func TestUpdatesEnvironmentReturnsAcceptedAndQueuesUpdateJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/updates", bytes.NewBufferString(`{"scope":"all"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_update", body["job_id"])

	var backupCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM backups WHERE environment_id = ?", environmentID).Scan(&backupCount); err != nil {
		t.Fatalf("count pre-update backups: %v", err)
	}
	if backupCount != 1 {
		t.Fatalf("expected pre-update backup row, got %d", backupCount)
	}

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.update' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.update audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.update audit log, got %d", auditCount)
	}
}

func TestRestoreEnvironmentReturnsAcceptedAndQueuesRestoreJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")
	seedBackup(t, db, environmentID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/restore", bytes.NewBufferString(`{"backup_id":"backup-1"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_restore", body["job_id"])
	assertDBString(t, db, "SELECT status FROM environments WHERE id = ?", "restoring", environmentID)

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.restore' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.restore audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.restore audit log, got %d", auditCount)
	}
}

func TestRestoreEnvironmentRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/restore", bytes.NewBufferString(`{"backup_id":""}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestToggleEnvironmentCacheReturnsAcceptedAndQueuesJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/environments/"+environmentID+"/cache", bytes.NewBufferString(`{"fastcgi_cache_enabled":false}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_cache_toggle", body["job_id"])

	var fastcgi, redis int
	if err := db.QueryRow("SELECT fastcgi_cache_enabled, redis_cache_enabled FROM environments WHERE id = ?", environmentID).Scan(&fastcgi, &redis); err != nil {
		t.Fatalf("query cache flags: %v", err)
	}
	if fastcgi != 0 || redis != 1 {
		t.Fatalf("unexpected cache flags fastcgi=%d redis=%d", fastcgi, redis)
	}

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.cache_toggle' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.cache_toggle audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.cache_toggle audit log, got %d", auditCount)
	}
}

func TestToggleEnvironmentCacheRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/environments/"+environmentID+"/cache", bytes.NewBufferString(`{}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestPurgeEnvironmentCacheReturnsAcceptedAndQueuesJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/cache/purge", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "cache_purge", body["job_id"])

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.cache_purge' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.cache_purge audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.cache_purge audit log, got %d", auditCount)
	}
}

func TestMagicLoginReturnsURLWithoutEnqueuingJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServerWithSSHRunner(t, apiStubSSHRunner{output: "https://example.test/wp-admin/?token=abc"})
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/magic-login", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["login_url"]) == "" || strings.TrimSpace(body["expires_at"]) == "" {
		t.Fatalf("expected login_url and expires_at in response")
	}

	var jobCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM jobs").Scan(&jobCount); err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobCount != 0 {
		t.Fatalf("expected no job enqueue for magic login, got %d jobs", jobCount)
	}

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'magic_login' AND result = 'success'").Scan(&auditCount); err != nil {
		t.Fatalf("query magic_login audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one successful magic_login audit log, got %d", auditCount)
	}
}

func TestMagicLoginReturnsEnvironmentNotActive(t *testing.T) {
	t.Parallel()

	server, db := newTestServerWithSSHRunner(t, apiStubSSHRunner{output: "https://example.test/wp-admin/?token=abc"})
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	if _, err := db.Exec("UPDATE environments SET status = 'deploying' WHERE id = ?", environmentID); err != nil {
		t.Fatalf("set environment deploying: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/magic-login", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "environment_not_active" {
		t.Fatalf("expected environment_not_active code, got %q", body["code"])
	}
}

func TestMagicLoginReturnsNodeUnreachable(t *testing.T) {
	t.Parallel()

	server, db := newTestServerWithSSHRunner(t, apiStubSSHRunner{err: context.DeadlineExceeded})
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/magic-login", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "node_unreachable" {
		t.Fatalf("expected node_unreachable code, got %q", body["code"])
	}
}

func TestMagicLoginReturnsWPCliError(t *testing.T) {
	t.Parallel()

	server, db := newTestServerWithSSHRunner(t, apiStubSSHRunner{output: "Error: command failed", err: errors.New("exit status 1")})
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/magic-login", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "wp_cli_error" {
		t.Fatalf("expected wp_cli_error code, got %q", body["code"])
	}
}

func TestDriftCheckReturnsAcceptedAndQueuesJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/drift-check", bytes.NewBuffer(nil))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "drift_check", body["job_id"])

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.drift_check' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.drift_check audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.drift_check audit log, got %d", auditCount)
	}
}

func TestPromoteReturnsConflictWhenGatesAreUnmet(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, sourceEnvironmentID := seedSiteWithProductionEnvironment(t, db, "acme")
	targetEnvironmentID := seedSecondEnvironment(t, db, "site-acme", "production-copy", "production")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+sourceEnvironmentID+"/promote", bytes.NewBufferString(`{"target_environment_id":"`+targetEnvironmentID+`"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestPromoteReturnsAcceptedWhenGatesPass(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, sourceEnvironmentID := seedSiteWithProductionEnvironment(t, db, "acme")
	targetEnvironmentID := seedSecondEnvironment(t, db, "site-acme", "staging-copy", "staging")
	seedCleanDriftState(t, db, sourceEnvironmentID)
	seedFreshBackup(t, db, targetEnvironmentID, "backup-target")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+sourceEnvironmentID+"/promote", bytes.NewBufferString(`{"target_environment_id":"`+targetEnvironmentID+`"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_promote", body["job_id"])

	var auditCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_logs WHERE action = 'environment.promote' AND result = 'accepted'").Scan(&auditCount); err != nil {
		t.Fatalf("query environment.promote audit logs: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one environment.promote audit log, got %d", auditCount)
	}
}

func TestListBackupsReturnsRetentionMetadata(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")
	seedBackup(t, db, environmentID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/environments/"+environmentID+"/backups", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var listed []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one backup, got %d", len(listed))
	}
	if strings.TrimSpace(asString(t, listed[0]["retention_until"])) == "" {
		t.Fatalf("expected retention metadata")
	}
	if asString(t, listed[0]["status"]) != "completed" {
		t.Fatalf("expected completed status, got %v", listed[0]["status"])
	}
}

func TestAddDomainReturnsAcceptedAndQueuesDomainAddJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/environments/"+environmentID+"/domains", bytes.NewBufferString(`{"hostname":"example.com"}`))
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "domain_add", body["job_id"])

	var domainCount int
	if err := db.QueryRow("SELECT COUNT(1) FROM domains WHERE environment_id = ? AND hostname = 'example.com' AND tls_status = 'pending'", environmentID).Scan(&domainCount); err != nil {
		t.Fatalf("count pending domains: %v", err)
	}
	if domainCount != 1 {
		t.Fatalf("expected one pending domain row, got %d", domainCount)
	}
}

func TestListDomainsReturnsTLSStatus(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")

	seedDomainRecord(t, db, "domain-1", environmentID, "example.com", "active")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/environments/"+environmentID+"/domains", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var listed []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one domain, got %d", len(listed))
	}
	if asString(t, listed[0]["tls_status"]) != "active" {
		t.Fatalf("expected tls_status active, got %v", listed[0]["tls_status"])
	}
}

func TestDeleteDomainReturnsAcceptedAndQueuesDomainRemoveJob(t *testing.T) {
	t.Parallel()

	server, db := newTestServer(t)
	seedUser(t, db, "admin@example.com", "secret")
	seedActiveNode(t, db, "node-1", "203.0.113.5")
	token := loginForToken(t, server)
	_, environmentID := seedSiteWithProductionEnvironment(t, db, "acme")
	seedDomainRecord(t, db, "domain-1", environmentID, "example.com", "active")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/domains/domain-1", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if strings.TrimSpace(body["job_id"]) == "" {
		t.Fatalf("expected job_id in accepted response")
	}

	assertDBString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "domain_remove", body["job_id"])
}

func newTestServer(t *testing.T) (*Server, *sql.DB) {
	return newTestServerWithSSHRunner(t, apiStubSSHRunner{output: "https://example.test/wp-admin/?token=abc"})
}

func newTestServerWithDashboardFS(t *testing.T, dashboardFS fs.FS) (*Server, *sql.DB) {
	t.Helper()
	return newTestServerWithSSHRunner(t, apiStubSSHRunner{output: "https://example.test/wp-admin/?token=abc"}, WithDashboardFS(dashboardFS))
}

func newTestServerWithSSHRunner(t *testing.T, runner ssh.Runner, opts ...ServerOption) (*Server, *sql.DB) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "api-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE auth_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			session_token TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			revoked_at TEXT NULL
		);
		CREATE TABLE audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			result TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE nodes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			hostname TEXT NOT NULL,
			public_ip TEXT NULL,
			ssh_port INTEGER NOT NULL,
			ssh_user TEXT NOT NULL,
			status TEXT NOT NULL,
			is_local INTEGER NOT NULL,
			last_seen_at TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			primary_environment_id TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			environment_type TEXT NOT NULL,
			status TEXT NOT NULL,
			node_id TEXT NOT NULL,
			source_environment_id TEXT NULL,
			promotion_preset TEXT NOT NULL,
			preview_url TEXT NOT NULL,
			primary_domain_id TEXT NULL,
			current_release_id TEXT NULL,
			drift_status TEXT NOT NULL,
			drift_checked_at TEXT NULL,
			last_drift_check_id TEXT NULL,
			fastcgi_cache_enabled INTEGER NOT NULL,
			redis_cache_enabled INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL,
			UNIQUE(site_id, slug)
		);
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			job_type TEXT NOT NULL,
			status TEXT NOT NULL,
			site_id TEXT NULL,
			environment_id TEXT NULL,
			node_id TEXT NULL,
			payload_json TEXT NOT NULL,
			attempt_count INTEGER NOT NULL,
			max_attempts INTEGER NOT NULL,
			run_after TEXT NULL,
			locked_at TEXT NULL,
			locked_by TEXT NULL,
			started_at TEXT NULL,
			finished_at TEXT NULL,
			error_code TEXT NULL,
			error_message TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE backups (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			backup_scope TEXT NOT NULL,
			status TEXT NOT NULL,
			storage_type TEXT NOT NULL,
			storage_path TEXT NOT NULL,
			retention_until TEXT NOT NULL,
			checksum TEXT NULL,
			size_bytes INTEGER NULL,
			created_at TEXT NOT NULL,
			completed_at TEXT NULL
		);
		CREATE TABLE domains (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			hostname TEXT NOT NULL UNIQUE,
			tls_status TEXT NOT NULL,
			tls_issuer TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE releases (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_ref TEXT NOT NULL,
			path TEXT NOT NULL,
			health_status TEXT NOT NULL,
			notes TEXT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE drift_checks (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			promotion_preset TEXT NOT NULL,
			status TEXT NOT NULL,
			db_checksums_json TEXT NULL,
			file_checksums_json TEXT NULL,
			checked_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create auth schema: %v", err)
	}

	secretStore := secrets.NewStore(filepath.Join(t.TempDir(), "secrets"))
	settingsService := settings.NewService(db, secretStore)

	return NewServer(auth.NewService(db), sites.NewService(db), environments.NewService(db), promotion.NewService(db), ssh.NewService(db, runner), settingsService, jobs.NewService(db), metrics.NewService(db), backups.NewService(db), domains.NewService(db), migration.NewService(db), audit.NewService(db), opts...), db
}

type apiStubSSHRunner struct {
	output string
	err    error
}

func (s apiStubSSHRunner) Run(_ context.Context, _ string, _ int, _ string, _ ...string) (string, error) {
	return s.output, s.err
}

func seedUser(t *testing.T, db *sql.DB, email, password string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, display_name, role, password_hash, is_active, created_at, updated_at)
		VALUES ('user-1', ?, 'Admin', 'admin', ?, 1, datetime('now'), datetime('now'))
	`, email, password); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedActiveNode(t *testing.T, db *sql.DB, nodeID, publicIP string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES (?, 'local-node', 'localhost', ?, 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1)
	`, nodeID, publicIP); err != nil {
		t.Fatalf("seed node: %v", err)
	}
}

func seedSiteWithProductionEnvironment(t *testing.T, db *sql.DB, slug string) (string, string) {
	t.Helper()

	siteID := "site-" + slug
	envID := "env-" + slug + "-production"

	if _, err := db.Exec(`
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES (?, ?, ?, 'active', ?, datetime('now'), datetime('now'), 1)
	`, siteID, strings.ToUpper(slug), slug, envID); err != nil {
		t.Fatalf("seed site: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES (?, ?, 'Production', 'production', 'production', 'active', 'node-1', NULL, 'content-protect', 'http://prod.203-0-113-5.sslip.io', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1)
	`, envID, siteID); err != nil {
		t.Fatalf("seed production environment: %v", err)
	}

	return siteID, envID
}

func seedSecondEnvironment(t *testing.T, db *sql.DB, siteID, slug, environmentType string) string {
	t.Helper()

	envID := "env-" + slug
	if _, err := db.Exec(`
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES (?, ?, ?, ?, ?, 'active', 'node-1', NULL, 'content-protect', 'http://preview.test', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1)
	`, envID, siteID, strings.ToUpper(slug), slug, environmentType); err != nil {
		t.Fatalf("seed second environment: %v", err)
	}

	return envID
}

func seedCleanDriftState(t *testing.T, db *sql.DB, environmentID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO drift_checks (id, environment_id, promotion_preset, status, db_checksums_json, file_checksums_json, checked_at)
		VALUES ('drift-clean', ?, 'content-protect', 'clean', '{}', '{}', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	`, environmentID); err != nil {
		t.Fatalf("seed drift check: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE environments
		SET drift_status = 'clean', last_drift_check_id = 'drift-clean'
		WHERE id = ?
	`, environmentID); err != nil {
		t.Fatalf("seed clean drift state: %v", err)
	}
}

func seedFreshBackup(t *testing.T, db *sql.DB, environmentID, backupID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'completed', 's3', 's3://pressluft/backups/test/backup.tar.zst', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+30 day'), 'sha256:ok', 123, strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'))
	`, backupID, environmentID); err != nil {
		t.Fatalf("seed fresh backup: %v", err)
	}
}

func loginForToken(t *testing.T, server *Server) string {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"secret"}`))
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status: %d", rec.Code)
	}

	cookie := getCookie(rec.Result().Cookies(), "session_token")
	if cookie == nil {
		t.Fatalf("missing session cookie")
	}

	return cookie.Value
}

func getCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func seedBackup(t *testing.T, db *sql.DB, environmentID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (
			'backup-1', ?, 'full', 'completed', 's3', 's3://pressluft/backups/test/backup-1.tar.zst',
			datetime('now', '+30 day'), 'sha256:abc', 100, datetime('now'), datetime('now')
		)
	`, environmentID); err != nil {
		t.Fatalf("seed backup: %v", err)
	}
}

func seedDomainRecord(t *testing.T, db *sql.DB, domainID, environmentID, hostname, tlsStatus string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO domains (id, environment_id, hostname, tls_status, tls_issuer, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'letsencrypt', datetime('now'), datetime('now'))
	`, domainID, environmentID, hostname, tlsStatus); err != nil {
		t.Fatalf("seed domain: %v", err)
	}
}

func assertDBString(t *testing.T, db *sql.DB, query, expected, arg string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}

func asString(t *testing.T, value any) string {
	t.Helper()

	str, ok := value.(string)
	if !ok {
		t.Fatalf("expected string value, got %T", value)
	}
	return str
}
