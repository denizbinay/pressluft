package server

import (
	"database/sql"

	"pressluft/internal/controlplane/server/stores"
)

// Re-export store types for backward compatibility.
type StoredServer = stores.StoredServer
type CreateServerNodeInput = stores.CreateServerNodeInput
type ServerStore = stores.ServerStore
type QueueServerJobInput = stores.QueueServerJobInput

// Re-export sentinel errors for backward compatibility.
var (
	ErrServerActionConflict = stores.ErrServerActionConflict
	ErrServerDeleting       = stores.ErrServerDeleting
	ErrServerDeleted        = stores.ErrServerDeleted
)

// NewServerStore creates a new server store.
func NewServerStore(db *sql.DB) *ServerStore {
	return stores.NewServerStore(db)
}
