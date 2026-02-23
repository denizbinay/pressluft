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
