package provider

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/security"
)

// StoredProvider represents a provider row persisted in the database.
type StoredProvider struct {
	ID                int64  `json:"id"`
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

// Create inserts a new provider and returns its ID.
func (s *Store) Create(ctx context.Context, providerType, name, apiToken string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	encrypted, keyID, version, err := security.EncryptProviderToken(apiToken)
	if err != nil {
		return 0, fmt.Errorf("encrypt provider token: %w", err)
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO providers (type, name, api_token, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at)
		 VALUES (?, ?, NULL, ?, ?, ?, 'active', ?, ?)`,
		providerType, name, encrypted, keyID, version, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "no such column") {
			res, err = s.db.ExecContext(ctx,
				`INSERT INTO providers (type, name, api_token, status, created_at, updated_at)
				 VALUES (?, ?, ?, 'active', ?, ?)`,
				providerType, name, apiToken, now, now,
			)
			if err != nil {
				return 0, fmt.Errorf("insert provider legacy: %w", err)
			}
		} else {
			return 0, fmt.Errorf("insert provider: %w", err)
		}
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
		`SELECT id, type, name, COALESCE(api_token, ''), COALESCE(api_token_encrypted, ''), COALESCE(api_token_key_id, ''), COALESCE(api_token_version, 0), status, created_at, updated_at
		 FROM providers
		 WHERE id = ?`,
		id,
	)

	var p StoredProvider
	if err := row.Scan(&p.ID, &p.Type, &p.Name, &p.APIToken, &p.APITokenEncrypted, &p.APITokenKeyID, &p.APITokenVersion, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if strings.Contains(err.Error(), "no such column") {
			legacyRow := s.db.QueryRowContext(ctx,
				`SELECT id, type, name, api_token, status, created_at, updated_at
				 FROM providers
				 WHERE id = ?`,
				id,
			)
			if err := legacyRow.Scan(&p.ID, &p.Type, &p.Name, &p.APIToken, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
				if err == sql.ErrNoRows {
					return nil, fmt.Errorf("provider %d not found", id)
				}
				return nil, fmt.Errorf("get provider legacy: %w", err)
			}
			return &p, nil
		}
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider %d not found", id)
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

func (s *Store) BackfillEncryptedTokens(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, api_token
		FROM providers
		WHERE COALESCE(api_token, '') <> ''
		  AND COALESCE(api_token_encrypted, '') = ''
	`)
	if err != nil {
		if strings.Contains(err.Error(), "no such column") {
			return nil
		}
		return fmt.Errorf("query legacy provider tokens: %w", err)
	}
	defer rows.Close()

	type legacyRow struct {
		id    int64
		token string
	}
	var legacy []legacyRow
	for rows.Next() {
		var row legacyRow
		if err := rows.Scan(&row.id, &row.token); err != nil {
			return fmt.Errorf("scan legacy provider token: %w", err)
		}
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy provider tokens: %w", err)
	}

	for _, row := range legacy {
		encrypted, keyID, version, err := security.EncryptProviderToken(row.token)
		if err != nil {
			return fmt.Errorf("encrypt provider token for %d: %w", row.id, err)
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE providers
			SET api_token_encrypted = ?, api_token_key_id = ?, api_token_version = ?, updated_at = ?
			WHERE id = ?
		`, encrypted, keyID, version, time.Now().UTC().Format(time.RFC3339), row.id); err != nil {
			return fmt.Errorf("backfill provider token %d: %w", row.id, err)
		}
	}

	return nil
}
