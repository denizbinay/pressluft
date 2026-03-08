package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pressluft/internal/platform"

	_ "modernc.org/sqlite"
)

func TestJobsCreateAndGetEndpoints(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)

	body, _ := json.Marshal(map[string]any{"kind": "provision_server", "server_id": 0})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusAccepted)
	}

	var created struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID <= 0 {
		t.Fatal("expected job id")
	}
	if created.Status != "queued" {
		t.Fatalf("status = %q, want %q", created.Status, "queued")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/jobs/"+intToString(created.ID), nil)
	getRes := httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)

	if getRes.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", getRes.Code, http.StatusOK)
	}
}

func TestJobsCreatePayloadValidation(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)
	upgrade := true
	automount := true

	tests := []struct {
		name     string
		kind     string
		serverID int64
		payload  any
		wantCode int
	}{
		{
			name:     "unknown kind rejected",
			kind:     "unknown_job",
			serverID: 1,
			payload:  map[string]any{},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "rebuild valid payload",
			kind:     "rebuild_server",
			serverID: 1,
			payload: map[string]any{
				"server_name":  "agency-prod-01",
				"server_image": "ubuntu-24.04",
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "rebuild invalid payload type",
			kind:     "rebuild_server",
			serverID: 1,
			payload:  "not-an-object",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize missing server_type",
			kind:     "resize_server",
			serverID: 1,
			payload: map[string]any{
				"upgrade_disk": upgrade,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize missing upgrade_disk",
			kind:     "resize_server",
			serverID: 1,
			payload: map[string]any{
				"server_type": "cx32",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize valid payload",
			kind:     "resize_server",
			serverID: 1,
			payload: map[string]any{
				"server_type":  "cx32",
				"upgrade_disk": upgrade,
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "update firewalls empty list",
			kind:     "update_firewalls",
			serverID: 1,
			payload: map[string]any{
				"firewalls": []string{},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "update firewalls valid payload",
			kind:     "update_firewalls",
			serverID: 1,
			payload: map[string]any{
				"firewalls": []string{"web", ""},
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "manage volume missing name",
			kind:     "manage_volume",
			serverID: 1,
			payload: map[string]any{
				"state":     "present",
				"size_gb":   10,
				"location":  "fsn1",
				"automount": automount,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "manage volume missing automount",
			kind:     "manage_volume",
			serverID: 1,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "present",
				"size_gb":     10,
				"location":    "fsn1",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "manage volume missing size_gb",
			kind:     "manage_volume",
			serverID: 1,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "present",
				"automount":   automount,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "manage volume absent valid",
			kind:     "manage_volume",
			serverID: 1,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "absent",
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "manage volume present valid",
			kind:     "manage_volume",
			serverID: 1,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "present",
				"size_gb":     20,
				"automount":   automount,
			},
			wantCode: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverID := int64(0)
			if tt.serverID > 0 {
				serverID = mustInsertJobServer(t, db, string(platform.ServerStatusReady))
			}
			bodyBytes, _ := json.Marshal(map[string]any{
				"kind":      tt.kind,
				"server_id": serverID,
				"payload":   tt.payload,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			if res.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body = %s", res.Code, tt.wantCode, res.Body.String())
			}
		})
	}
}

func TestJobsCreateRejectsMissingKind(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)

	body, _ := json.Marshal(map[string]any{"server_id": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "kind is required") {
		t.Fatalf("body = %q, want kind validation error", res.Body.String())
	}
}

func TestJobsCreateRejectsDuplicateDestructiveAction(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)
	serverID := mustInsertJobServer(t, db, string(platform.ServerStatusReady))

	body, _ := json.Marshal(map[string]any{
		"kind":      "rebuild_server",
		"server_id": serverID,
		"payload": map[string]any{
			"server_image": "ubuntu-24.04",
		},
	})

	firstReq := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	firstReq.Header.Set("Content-Type", "application/json")
	firstRes := httptest.NewRecorder()
	handler.ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want %d; body = %s", firstRes.Code, http.StatusAccepted, firstRes.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRes := httptest.NewRecorder()
	handler.ServeHTTP(secondRes, secondReq)

	if secondRes.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", secondRes.Code, http.StatusConflict, secondRes.Body.String())
	}
}

func TestJobsCreateRejectsActionsForDeletedServer(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)
	serverID := mustInsertJobServer(t, db, string(platform.ServerStatusDeleted))

	body, _ := json.Marshal(map[string]any{
		"kind":      "restart_service",
		"server_id": serverID,
		"payload": map[string]any{
			"service_name": "nginx",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusConflict, res.Body.String())
	}
}

func TestJobsCreateRejectsDisallowedRestartServiceName(t *testing.T) {
	db := mustOpenJobsHandlerDB(t)
	handler := NewHandler(db)
	serverID := mustInsertJobServer(t, db, string(platform.ServerStatusReady))

	body, _ := json.Marshal(map[string]any{
		"kind":      "restart_service",
		"server_id": serverID,
		"payload": map[string]any{
			"service_name": "sshd",
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "not allowed") {
		t.Fatalf("body = %q, want not allowed validation message", res.Body.String())
	}
}

func mustOpenJobsHandlerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE servers (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id        INTEGER,
			provider_type      TEXT,
			provider_server_id TEXT,
			ipv4               TEXT,
			ipv6               TEXT,
			name               TEXT,
			location           TEXT,
			server_type        TEXT,
			image              TEXT,
			profile_key        TEXT,
			status             TEXT    NOT NULL DEFAULT 'ready',
			setup_state        TEXT    NOT NULL DEFAULT 'not_started',
			setup_last_error   TEXT,
			action_id          TEXT,
			action_status      TEXT,
			node_status        TEXT DEFAULT 'unknown',
			node_last_seen     TEXT,
			node_version       TEXT,
			created_at         TEXT    NOT NULL DEFAULT '2026-01-01T00:00:00Z',
			updated_at         TEXT    NOT NULL DEFAULT '2026-01-01T00:00:00Z'
		);

		CREATE TABLE server_keys (
			server_id             INTEGER PRIMARY KEY,
			public_key            TEXT    NOT NULL,
			private_key_encrypted TEXT    NOT NULL,
			encryption_key_id     TEXT    NOT NULL,
			created_at            TEXT    NOT NULL,
			rotated_at            TEXT,
			FOREIGN KEY (server_id) REFERENCES servers(id)
		);

		CREATE TABLE jobs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id    INTEGER,
			kind         TEXT    NOT NULL,
			status       TEXT    NOT NULL,
			current_step TEXT    NOT NULL DEFAULT '',
			retry_count  INTEGER NOT NULL DEFAULT 0,
			last_error   TEXT,
			payload      TEXT,
			started_at   TEXT,
			finished_at  TEXT,
			timeout_at   TEXT,
			command_id   TEXT,
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL,
			FOREIGN KEY (server_id) REFERENCES servers(id)
		);

		CREATE TABLE job_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id     INTEGER NOT NULL,
			seq        INTEGER NOT NULL,
			event_type TEXT    NOT NULL,
			level      TEXT    NOT NULL,
			step_key   TEXT,
			status     TEXT,
			message    TEXT    NOT NULL,
			payload    TEXT,
			created_at TEXT    NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
		);
	`); err != nil {
		t.Fatalf("create jobs schema: %v", err)
	}

	return db
}

func mustInsertJobServer(t *testing.T, db *sql.DB, status string) int64 {
	t.Helper()

	res, err := db.Exec(
		`INSERT INTO servers (provider_id, provider_type, name, location, server_type, image, profile_key, status, setup_state, created_at, updated_at)
		 VALUES (1, 'hetzner', 'job-server', 'fsn1', 'cx22', 'ubuntu-24.04', 'nginx-stack', ?, 'ready', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		status,
	)
	if err != nil {
		t.Fatalf("insert server: %v", err)
	}
	serverID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("server insert id: %v", err)
	}
	return serverID
}
