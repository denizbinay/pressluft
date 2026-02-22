package audit

import (
	"context"
	"testing"
	"time"
)

func TestServiceRecordCreatesEntry(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()
	svc := NewService(store)
	now := time.Date(2026, 2, 22, 0, 30, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	err := svc.Record(ctx, Entry{
		UserID:       "admin",
		Action:       "auth_login",
		ResourceType: "session",
		ResourceID:   "admin",
		Result:       "succeeded",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	entries, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
	if entries[0].CreatedAt != now {
		t.Fatalf("created_at = %s, want %s", entries[0].CreatedAt.Format(time.RFC3339), now.Format(time.RFC3339))
	}
	if entries[0].ID == "" {
		t.Fatal("id is empty")
	}
}

func TestServiceAsyncUpsertUpdatesResult(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()
	svc := NewService(store)
	now := time.Date(2026, 2, 22, 0, 31, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	err := svc.RecordAsyncAccepted(ctx, Entry{
		UserID:       "admin",
		Action:       "node_provision",
		ResourceType: "job",
		ResourceID:   "job-1",
	})
	if err != nil {
		t.Fatalf("RecordAsyncAccepted() error = %v", err)
	}

	err = svc.UpdateAsyncResult(ctx, "node_provision", "job", "job-1", "succeeded")
	if err != nil {
		t.Fatalf("UpdateAsyncResult() error = %v", err)
	}

	entries, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
	if entries[0].Result != "succeeded" {
		t.Fatalf("result = %s, want succeeded", entries[0].Result)
	}
}
