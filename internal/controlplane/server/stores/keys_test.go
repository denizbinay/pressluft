package stores

import (
	"context"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestKeysCreateAndRetrieve(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	err := store.CreateKey(context.Background(), CreateServerKeyInput{
		ServerID:            serverID,
		PublicKey:           "ssh-ed25519 AAAA...",
		PrivateKeyEncrypted: "encrypted-private-key",
		EncryptionKeyID:     "key-001",
	})
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	key, err := store.GetKey(context.Background(), serverID)
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if key == nil {
		t.Fatal("GetKey() returned nil, want key")
	}
	if key.ServerID != serverID {
		t.Fatalf("ServerID = %q, want %q", key.ServerID, serverID)
	}
	if key.PublicKey != "ssh-ed25519 AAAA..." {
		t.Fatalf("PublicKey = %q, want %q", key.PublicKey, "ssh-ed25519 AAAA...")
	}
	if key.PrivateKeyEncrypted != "encrypted-private-key" {
		t.Fatalf("PrivateKeyEncrypted = %q, want %q", key.PrivateKeyEncrypted, "encrypted-private-key")
	}
	if key.EncryptionKeyID != "key-001" {
		t.Fatalf("EncryptionKeyID = %q, want %q", key.EncryptionKeyID, "key-001")
	}
	if key.CreatedAt == "" {
		t.Fatal("CreatedAt is empty")
	}
	if key.RotatedAt != "" {
		t.Fatalf("RotatedAt = %q, want empty (no rotation)", key.RotatedAt)
	}
}

func TestKeysCreateWithRotatedAt(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	err := store.CreateKey(context.Background(), CreateServerKeyInput{
		ServerID:            serverID,
		PublicKey:           "ssh-ed25519 BBBB...",
		PrivateKeyEncrypted: "encrypted-key-2",
		EncryptionKeyID:     "key-002",
		RotatedAt:           "2026-01-15T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateKey() error = %v", err)
	}

	key, err := store.GetKey(context.Background(), serverID)
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if key.RotatedAt != "2026-01-15T12:00:00Z" {
		t.Fatalf("RotatedAt = %q, want %q", key.RotatedAt, "2026-01-15T12:00:00Z")
	}
}

func TestKeysCreateDuplicateServerFails(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	input := CreateServerKeyInput{
		ServerID:            serverID,
		PublicKey:           "ssh-ed25519 CCCC...",
		PrivateKeyEncrypted: "encrypted-key-3",
		EncryptionKeyID:     "key-003",
	}

	if err := store.CreateKey(context.Background(), input); err != nil {
		t.Fatalf("first CreateKey() error = %v", err)
	}

	err := store.CreateKey(context.Background(), input)
	if err == nil {
		t.Fatal("second CreateKey() expected error for duplicate server_id, got nil")
	}
}

func TestKeysGetMissingKeyReturnsNil(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	key, err := store.GetKey(context.Background(), serverID)
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if key != nil {
		t.Fatalf("GetKey() = %+v, want nil for missing key", key)
	}
}

func TestKeysGetInvalidServerIDFails(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	_, err := store.GetKey(context.Background(), "not-a-valid-id")
	if err == nil {
		t.Fatal("GetKey() expected error for invalid server ID, got nil")
	}
}

func TestKeysCreateValidationMissingFields(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	cases := []struct {
		name  string
		input CreateServerKeyInput
		want  string
	}{
		{
			name:  "missing server_id",
			input: CreateServerKeyInput{PublicKey: "key", PrivateKeyEncrypted: "enc", EncryptionKeyID: "kid"},
			want:  "server_id",
		},
		{
			name:  "missing public_key",
			input: CreateServerKeyInput{ServerID: "00000000-0000-7000-8000-000000000001", PrivateKeyEncrypted: "enc", EncryptionKeyID: "kid"},
			want:  "public_key",
		},
		{
			name:  "missing private_key_encrypted",
			input: CreateServerKeyInput{ServerID: "00000000-0000-7000-8000-000000000001", PublicKey: "key", EncryptionKeyID: "kid"},
			want:  "private_key_encrypted",
		},
		{
			name:  "missing encryption_key_id",
			input: CreateServerKeyInput{ServerID: "00000000-0000-7000-8000-000000000001", PublicKey: "key", PrivateKeyEncrypted: "enc"},
			want:  "encryption_key_id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.CreateKey(context.Background(), tc.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want to contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestKeysCreateForNonExistentServerFails(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	err := store.CreateKey(context.Background(), CreateServerKeyInput{
		ServerID:            "00000000-0000-7000-8000-999999999999",
		PublicKey:           "ssh-ed25519 DDDD...",
		PrivateKeyEncrypted: "encrypted-key-4",
		EncryptionKeyID:     "key-004",
	})
	if err == nil {
		t.Fatal("CreateKey() expected error for non-existent server, got nil")
	}
}
