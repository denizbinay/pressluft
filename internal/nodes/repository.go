package nodes

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"pressluft/internal/store"
)

type Node struct {
	ID       string
	Name     string
	Hostname string
	Status   string
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureLocalProvisioningNode(ctx context.Context, hostname string) (Node, bool, error) {
	var result Node
	created := false

	err := store.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
			SELECT id, name, hostname, status
			FROM nodes
			WHERE is_local = 1
			LIMIT 1
		`)

		err := row.Scan(&result.ID, &result.Name, &result.Hostname, &result.Status)
		if err == nil {
			return nil
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("query local node: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		id, err := randomID("node")
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO nodes (
				id, name, hostname, public_ip, ssh_port, ssh_user, status,
				is_local, last_seen_at, created_at, updated_at, state_version
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, "local-node", hostname, nil, 22, "root", "provisioning", 1, nil, now, now, 1); err != nil {
			return fmt.Errorf("insert local node: %w", err)
		}

		result = Node{
			ID:       id,
			Name:     "local-node",
			Hostname: hostname,
			Status:   "provisioning",
		}
		created = true
		return nil
	})
	if err != nil {
		return Node{}, false, err
	}

	return result, created, nil
}

func randomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}

	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf)), nil
}
