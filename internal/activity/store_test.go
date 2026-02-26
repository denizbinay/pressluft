package activity

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEmitAndGetByID(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	input := EmitInput{
		EventType:    EventJobCreated,
		Category:     CategoryJob,
		Level:        LevelInfo,
		ResourceType: ResourceJob,
		ResourceID:   42,
		ActorType:    ActorSystem,
		Title:        "Job created",
		Message:      "Provisioning job queued",
	}

	act, err := store.Emit(ctx, input)
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	if act.ID <= 0 {
		t.Fatal("expected positive ID")
	}
	if act.EventType != EventJobCreated {
		t.Errorf("event_type = %q, want %q", act.EventType, EventJobCreated)
	}
	if act.Title != "Job created" {
		t.Errorf("title = %q, want %q", act.Title, "Job created")
	}

	// Fetch by ID
	fetched, err := store.GetByID(ctx, act.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if fetched.ID != act.ID {
		t.Errorf("fetched.ID = %d, want %d", fetched.ID, act.ID)
	}
	if fetched.Message != "Provisioning job queued" {
		t.Errorf("message = %q, want %q", fetched.Message, "Provisioning job queued")
	}
}

func TestEmitValidatesEventType(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	input := EmitInput{
		EventType: "invalid.event.type",
		Category:  CategoryJob,
		Level:     LevelInfo,
		ActorType: ActorSystem,
		Title:     "Test",
	}

	_, err := store.Emit(ctx, input)
	if err == nil {
		t.Fatal("expected error for invalid event type")
	}
}

func TestEmitRequiresTitle(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	input := EmitInput{
		EventType: EventJobCreated,
		Category:  CategoryJob,
		Level:     LevelInfo,
		ActorType: ActorSystem,
		Title:     "",
	}

	_, err := store.Emit(ctx, input)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestListWithCursor(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create 5 activities
	for i := 0; i < 5; i++ {
		_, err := store.Emit(ctx, EmitInput{
			EventType: EventJobCreated,
			Category:  CategoryJob,
			Level:     LevelInfo,
			ActorType: ActorSystem,
			Title:     "Job created",
		})
		if err != nil {
			t.Fatalf("emit: %v", err)
		}
	}

	// List all with limit 3
	activities, cursor, err := store.List(ctx, ListFilter{Limit: 3})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("len = %d, want 3", len(activities))
	}
	if cursor == "" {
		t.Error("expected cursor for next page")
	}

	// List next page
	nextActivities, nextCursor, err := store.List(ctx, ListFilter{Limit: 3, Cursor: activities[2].ID})
	if err != nil {
		t.Fatalf("list next: %v", err)
	}
	if len(nextActivities) != 2 {
		t.Errorf("len = %d, want 2", len(nextActivities))
	}
	if nextCursor != "" {
		t.Errorf("expected empty cursor, got %q", nextCursor)
	}
}

func TestListByCategory(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create job activity
	_, _ = store.Emit(ctx, EmitInput{
		EventType: EventJobCreated,
		Category:  CategoryJob,
		Level:     LevelInfo,
		ActorType: ActorSystem,
		Title:     "Job created",
	})

	// Create server activity
	_, _ = store.Emit(ctx, EmitInput{
		EventType: EventServerCreated,
		Category:  CategoryServer,
		Level:     LevelInfo,
		ActorType: ActorSystem,
		Title:     "Server created",
	})

	// Filter by job category
	activities, _, err := store.List(ctx, ListFilter{Category: CategoryJob})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("len = %d, want 1", len(activities))
	}
	if activities[0].Category != CategoryJob {
		t.Errorf("category = %q, want %q", activities[0].Category, CategoryJob)
	}
}

func TestMarkRead(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	act, _ := store.Emit(ctx, EmitInput{
		EventType:         EventJobFailed,
		Category:          CategoryJob,
		Level:             LevelError,
		ActorType:         ActorSystem,
		Title:             "Job failed",
		RequiresAttention: true,
	})

	if act.ReadAt != "" {
		t.Error("expected read_at to be empty initially")
	}

	if err := store.MarkRead(ctx, act.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	updated, _ := store.GetByID(ctx, act.ID)
	if updated.ReadAt == "" {
		t.Error("expected read_at to be set")
	}
}

func TestMarkAllRead(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create job activities
	for i := 0; i < 3; i++ {
		_, _ = store.Emit(ctx, EmitInput{
			EventType:         EventJobFailed,
			Category:          CategoryJob,
			Level:             LevelError,
			ActorType:         ActorSystem,
			Title:             "Job failed",
			RequiresAttention: true,
		})
	}

	// Create server activity (should not be marked)
	_, _ = store.Emit(ctx, EmitInput{
		EventType:         EventServerDeleted,
		Category:          CategoryServer,
		Level:             LevelWarning,
		ActorType:         ActorSystem,
		Title:             "Server deleted",
		RequiresAttention: true,
	})

	// Mark all job activities as read
	if err := store.MarkAllRead(ctx, ListFilter{Category: CategoryJob}); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	// Check job activities are read
	jobActivities, _, _ := store.List(ctx, ListFilter{Category: CategoryJob})
	for _, a := range jobActivities {
		if a.ReadAt == "" {
			t.Errorf("job activity %d should be read", a.ID)
		}
	}

	// Check server activity is still unread
	serverActivities, _, _ := store.List(ctx, ListFilter{Category: CategoryServer})
	for _, a := range serverActivities {
		if a.ReadAt != "" {
			t.Errorf("server activity %d should be unread", a.ID)
		}
	}
}

func TestCountUnread(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create activities requiring attention
	for i := 0; i < 3; i++ {
		_, _ = store.Emit(ctx, EmitInput{
			EventType:         EventJobFailed,
			Category:          CategoryJob,
			Level:             LevelError,
			ActorType:         ActorSystem,
			Title:             "Job failed",
			RequiresAttention: true,
		})
	}

	// Create activity not requiring attention
	_, _ = store.Emit(ctx, EmitInput{
		EventType: EventJobCompleted,
		Category:  CategoryJob,
		Level:     LevelSuccess,
		ActorType: ActorSystem,
		Title:     "Job completed",
	})

	requiresAttention := true
	count, err := store.CountUnread(ctx, ListFilter{RequiresAttention: &requiresAttention})
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestGetLatestID(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Empty table
	latest, err := store.GetLatestID(ctx)
	if err != nil {
		t.Fatalf("get latest id: %v", err)
	}
	if latest != 0 {
		t.Errorf("latest = %d, want 0", latest)
	}

	// Add entries
	var lastID int64
	for i := 0; i < 3; i++ {
		act, _ := store.Emit(ctx, EmitInput{
			EventType: EventJobCreated,
			Category:  CategoryJob,
			Level:     LevelInfo,
			ActorType: ActorSystem,
			Title:     "Job created",
		})
		lastID = act.ID
	}

	latest, err = store.GetLatestID(ctx)
	if err != nil {
		t.Fatalf("get latest id: %v", err)
	}
	if latest != lastID {
		t.Errorf("latest = %d, want %d", latest, lastID)
	}
}

func TestListSince(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create initial activity
	first, _ := store.Emit(ctx, EmitInput{
		EventType: EventJobCreated,
		Category:  CategoryJob,
		Level:     LevelInfo,
		ActorType: ActorSystem,
		Title:     "First",
	})

	// Create more activities
	for i := 0; i < 3; i++ {
		_, _ = store.Emit(ctx, EmitInput{
			EventType: EventJobStarted,
			Category:  CategoryJob,
			Level:     LevelInfo,
			ActorType: ActorSystem,
			Title:     "Later",
		})
	}

	// List since first
	activities, err := store.ListSince(ctx, first.ID, 100)
	if err != nil {
		t.Fatalf("list since: %v", err)
	}
	if len(activities) != 3 {
		t.Errorf("len = %d, want 3", len(activities))
	}
	for _, a := range activities {
		if a.ID <= first.ID {
			t.Errorf("activity %d should be > %d", a.ID, first.ID)
		}
	}
}

func TestListByResource(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create activities for different jobs
	_, _ = store.Emit(ctx, EmitInput{
		EventType:    EventJobCreated,
		Category:     CategoryJob,
		Level:        LevelInfo,
		ResourceType: ResourceJob,
		ResourceID:   100,
		ActorType:    ActorSystem,
		Title:        "Job 100 created",
	})

	_, _ = store.Emit(ctx, EmitInput{
		EventType:    EventJobCreated,
		Category:     CategoryJob,
		Level:        LevelInfo,
		ResourceType: ResourceJob,
		ResourceID:   200,
		ActorType:    ActorSystem,
		Title:        "Job 200 created",
	})

	// Filter by resource
	activities, _, err := store.List(ctx, ListFilter{
		ResourceType: ResourceJob,
		ResourceID:   100,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("len = %d, want 1", len(activities))
	}
	if activities[0].ResourceID != 100 {
		t.Errorf("resource_id = %d, want 100", activities[0].ResourceID)
	}
}

func TestListByParentResource(t *testing.T) {
	db := mustOpenActivityDB(t)
	store := NewStore(db)
	ctx := context.Background()

	// Create job activities for different servers
	_, _ = store.Emit(ctx, EmitInput{
		EventType:          EventJobCreated,
		Category:           CategoryJob,
		Level:              LevelInfo,
		ResourceType:       ResourceJob,
		ResourceID:         1,
		ParentResourceType: ResourceServer,
		ParentResourceID:   10,
		ActorType:          ActorSystem,
		Title:              "Job for server 10",
	})

	_, _ = store.Emit(ctx, EmitInput{
		EventType:          EventJobCreated,
		Category:           CategoryJob,
		Level:              LevelInfo,
		ResourceType:       ResourceJob,
		ResourceID:         2,
		ParentResourceType: ResourceServer,
		ParentResourceID:   20,
		ActorType:          ActorSystem,
		Title:              "Job for server 20",
	})

	// Filter by parent resource
	activities, _, err := store.List(ctx, ListFilter{
		ParentResourceType: ResourceServer,
		ParentResourceID:   10,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("len = %d, want 1", len(activities))
	}
	if activities[0].ParentResourceID != 10 {
		t.Errorf("parent_resource_id = %d, want 10", activities[0].ParentResourceID)
	}
}

func mustOpenActivityDB(t *testing.T) *sql.DB {
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
		)
	`); err != nil {
		t.Fatalf("create activity schema: %v", err)
	}

	return db
}
