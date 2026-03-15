package provider

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pressluft/internal/shared/idutil"
	"pressluft/internal/shared/security"
)

// StoredProvider represents a provider row persisted in the database.
type StoredProvider struct {
	ID                string `json:"id"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	APIToken          string `json:"-"` // never serialised to JSON
	APITokenEncrypted string `json:"-"`
	APITokenKeyID     string `json:"-"`
	APITokenVersion   int    `json:"-"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// Store handles persistence of provider credentials.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new provider and returns its app ID.
func (s *Store) Create(ctx context.Context, providerType, name, apiToken string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	providerID, err := idutil.New()
	if err != nil {
		return "", err
	}
	encrypted, keyID, version, err := security.EncryptProviderToken(apiToken)
	if err != nil {
		return "", fmt.Errorf("encrypt provider token: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO providers (id, type, name, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		providerID, providerType, name, encrypted, keyID, version, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert provider: %w", err)
	}
	return providerID, nil
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

// Delete removes a provider by app ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	providerID, err := idutil.Normalize(id)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, providerID)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("provider %s not found", providerID)
	}
	return nil
}

// GetByID returns a provider row by app ID including the API token.
func (s *Store) GetByID(ctx context.Context, id string) (*StoredProvider, error) {
	providerID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, type, name, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at
		 FROM providers
		 WHERE id = ?`,
		providerID,
	)

	var p StoredProvider
	if err := row.Scan(&p.ID, &p.Type, &p.Name, &p.APITokenEncrypted, &p.APITokenKeyID, &p.APITokenVersion, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider %s not found", providerID)
		}
		return nil, fmt.Errorf("get provider: %w", err)
	}
	if p.APITokenEncrypted != "" {
		token, err := security.DecryptProviderToken(p.APITokenEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt provider token: %w", err)
		}
		p.APIToken = token
	}

	return &p, nil
}
