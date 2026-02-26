package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	serverID := mustInsertJobServer(t, db)

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
			name:     "rebuild valid payload",
			kind:     "rebuild_server",
			serverID: serverID,
			payload: map[string]any{
				"server_name":  "agency-prod-01",
				"server_image": "ubuntu-24.04",
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "rebuild invalid payload type",
			kind:     "rebuild_server",
			serverID: serverID,
			payload:  "not-an-object",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize missing server_type",
			kind:     "resize_server",
			serverID: serverID,
			payload: map[string]any{
				"upgrade_disk": upgrade,
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize missing upgrade_disk",
			kind:     "resize_server",
			serverID: serverID,
			payload: map[string]any{
				"server_type": "cx32",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "resize valid payload",
			kind:     "resize_server",
			serverID: serverID,
			payload: map[string]any{
				"server_type":  "cx32",
				"upgrade_disk": upgrade,
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "update firewalls empty list",
			kind:     "update_firewalls",
			serverID: serverID,
			payload: map[string]any{
				"firewalls": []string{},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "update firewalls valid payload",
			kind:     "update_firewalls",
			serverID: serverID,
			payload: map[string]any{
				"firewalls": []string{"web", ""},
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "manage volume missing name",
			kind:     "manage_volume",
			serverID: serverID,
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
			serverID: serverID,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "present",
				"size_gb":     10,
				"location":    "fsn1",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "manage volume absent valid",
			kind:     "manage_volume",
			serverID: serverID,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "absent",
			},
			wantCode: http.StatusAccepted,
		},
		{
			name:     "manage volume present valid",
			kind:     "manage_volume",
			serverID: serverID,
			payload: map[string]any{
				"volume_name": "data",
				"state":       "present",
				"size_gb":     20,
				"location":    "fsn1",
				"automount":   automount,
			},
			wantCode: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(map[string]any{
				"kind":      tt.kind,
				"server_id": tt.serverID,
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
			id INTEGER PRIMARY KEY AUTOINCREMENT
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

func mustInsertJobServer(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	res, err := db.Exec(`INSERT INTO servers DEFAULT VALUES`)
	if err != nil {
		t.Fatalf("insert server: %v", err)
	}
	serverID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("server insert id: %v", err)
	}
	return serverID
}
