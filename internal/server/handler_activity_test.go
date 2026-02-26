package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"pressluft/internal/activity"

	_ "modernc.org/sqlite"
)

func TestActivityListEndpoint(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	// Create test activities
	for i := 0; i < 3; i++ {
		_, err := store.Emit(t.Context(), activity.EmitInput{
			EventType: activity.EventJobCreated,
			Category:  activity.CategoryJob,
			Level:     activity.LevelInfo,
			ActorType: activity.ActorSystem,
			Title:     "Test job created",
		})
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var response struct {
		Data       []activity.Activity `json:"data"`
		NextCursor string              `json:"next_cursor"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Data) != 3 {
		t.Errorf("len = %d, want 3", len(response.Data))
	}
}

func TestActivityGetEndpoint(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	act, err := store.Emit(t.Context(), activity.EmitInput{
		EventType: activity.EventServerCreated,
		Category:  activity.CategoryServer,
		Level:     activity.LevelSuccess,
		ActorType: activity.ActorUser,
		Title:     "Server created",
		Message:   "Production server provisioned",
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/activity/"+strconv.FormatInt(act.ID, 10), nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var fetched activity.Activity
	if err := json.Unmarshal(res.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if fetched.ID != act.ID {
		t.Errorf("id = %d, want %d", fetched.ID, act.ID)
	}
	if fetched.Title != "Server created" {
		t.Errorf("title = %q, want %q", fetched.Title, "Server created")
	}
}

func TestActivityGetNotFound(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/activity/999", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
}

func TestActivityMarkReadEndpoint(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	act, err := store.Emit(t.Context(), activity.EmitInput{
		EventType:         activity.EventJobFailed,
		Category:          activity.CategoryJob,
		Level:             activity.LevelError,
		ActorType:         activity.ActorSystem,
		Title:             "Job failed",
		RequiresAttention: true,
	})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/activity/"+strconv.FormatInt(act.ID, 10)+"/read", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var updated activity.Activity
	if err := json.Unmarshal(res.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if updated.ReadAt == "" {
		t.Error("expected read_at to be set")
	}
}

func TestActivityMarkAllReadEndpoint(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	// Create unread activities
	for i := 0; i < 3; i++ {
		_, err := store.Emit(t.Context(), activity.EmitInput{
			EventType:         activity.EventJobFailed,
			Category:          activity.CategoryJob,
			Level:             activity.LevelError,
			ActorType:         activity.ActorSystem,
			Title:             "Job failed",
			RequiresAttention: true,
		})
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/activity/read-all?category=job", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	// Verify all are read
	requiresAttention := true
	count, _ := store.CountUnread(t.Context(), activity.ListFilter{RequiresAttention: &requiresAttention})
	if count != 0 {
		t.Errorf("unread count = %d, want 0", count)
	}
}

func TestActivityUnreadCountEndpoint(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	// Create attention-required activities
	for i := 0; i < 5; i++ {
		_, err := store.Emit(t.Context(), activity.EmitInput{
			EventType:         activity.EventJobFailed,
			Category:          activity.CategoryJob,
			Level:             activity.LevelError,
			ActorType:         activity.ActorSystem,
			Title:             "Job failed",
			RequiresAttention: true,
		})
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/activity/unread-count", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var response struct {
		Count int64 `json:"count"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Count != 5 {
		t.Errorf("count = %d, want 5", response.Count)
	}
}

func TestActivityListWithFilters(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	// Create job activities
	for i := 0; i < 2; i++ {
		_, _ = store.Emit(t.Context(), activity.EmitInput{
			EventType: activity.EventJobCreated,
			Category:  activity.CategoryJob,
			Level:     activity.LevelInfo,
			ActorType: activity.ActorSystem,
			Title:     "Job created",
		})
	}

	// Create server activities
	for i := 0; i < 3; i++ {
		_, _ = store.Emit(t.Context(), activity.EmitInput{
			EventType: activity.EventServerCreated,
			Category:  activity.CategoryServer,
			Level:     activity.LevelInfo,
			ActorType: activity.ActorSystem,
			Title:     "Server created",
		})
	}

	// Filter by category
	req := httptest.NewRequest(http.MethodGet, "/api/activity?category=server", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var response struct {
		Data []activity.Activity `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Data) != 3 {
		t.Errorf("len = %d, want 3", len(response.Data))
	}
}

func TestActivityListWithPagination(t *testing.T) {
	db := mustOpenActivityHandlerDB(t)
	handler := NewHandler(db)
	store := activity.NewStore(db)

	// Create 10 activities
	for i := 0; i < 10; i++ {
		_, err := store.Emit(t.Context(), activity.EmitInput{
			EventType: activity.EventJobCreated,
			Category:  activity.CategoryJob,
			Level:     activity.LevelInfo,
			ActorType: activity.ActorSystem,
			Title:     "Job created",
		})
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	// First page
	req := httptest.NewRequest(http.MethodGet, "/api/activity?limit=3", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	var page1 struct {
		Data       []activity.Activity `json:"data"`
		NextCursor string              `json:"next_cursor"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &page1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}

	if len(page1.Data) != 3 {
		t.Errorf("page1 len = %d, want 3", len(page1.Data))
	}
	if page1.NextCursor == "" {
		t.Error("expected next_cursor for page1")
	}

	// Second page using cursor
	req = httptest.NewRequest(http.MethodGet, "/api/activity?limit=3&cursor="+page1.NextCursor, nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	var page2 struct {
		Data       []activity.Activity `json:"data"`
		NextCursor string              `json:"next_cursor"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &page2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}

	if len(page2.Data) != 3 {
		t.Errorf("page2 len = %d, want 3", len(page2.Data))
	}

	// Verify no overlap
	for _, p1 := range page1.Data {
		for _, p2 := range page2.Data {
			if p1.ID == p2.ID {
				t.Errorf("overlap: activity %d appears in both pages", p1.ID)
			}
		}
	}
}

func mustOpenActivityHandlerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	// Create minimal schema needed for handler tests
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

		CREATE TABLE IF NOT EXISTS activity (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type           TEXT NOT NULL,
			category             TEXT NOT NULL,
			level                TEXT NOT NULL,
			resource_type        TEXT,
			resource_id          INTEGER,
			parent_resource_type TEXT,
			parent_resource_id   INTEGER,
			actor_type           TEXT NOT NULL,
			actor_id             TEXT,
			title                TEXT NOT NULL,
			message              TEXT,
			payload              TEXT,
			requires_attention   INTEGER NOT NULL DEFAULT 0,
			read_at              TEXT,
			created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}
