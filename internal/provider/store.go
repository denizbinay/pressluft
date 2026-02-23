package provider

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// StoredProvider represents a provider row persisted in the database.
type StoredProvider struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	APIToken  string `json:"-"` // never serialised to JSON
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Store handles persistence of provider credentials.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new provider and returns its ID.
func (s *Store) Create(ctx context.Context, providerType, name, apiToken string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO providers (type, name, api_token, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'active', ?, ?)`,
		providerType, name, apiToken, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert provider: %w", err)
	}
	return res.LastInsertId()
}

// List returns all providers. API tokens are NOT included in the result.
func (s *Store) List(ctx context.Context) ([]StoredProvider, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, status, created_at, updated_at
		 FROM providers ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	defer rows.Close()

	var out []StoredProvider
	for rows.Next() {
		var p StoredProvider
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Delete removes a provider by ID.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("provider %d not found", id)
	}
	return nil
}

// GetByID returns a provider row by ID including the API token.
func (s *Store) GetByID(ctx context.Context, id int64) (*StoredProvider, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, type, name, api_token, status, created_at, updated_at
		 FROM providers
		 WHERE id = ?`,
		id,
	)

	var p StoredProvider
	if err := row.Scan(&p.ID, &p.Type, &p.Name, &p.APIToken, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider %d not found", id)
		}
		return nil, fmt.Errorf("get provider: %w", err)
	}

	return &p, nil
}
