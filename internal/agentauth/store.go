package agentauth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

type Token struct {
	ID         int64
	ServerID   int64
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(serverID int64, expiresIn time.Duration) (string, error) {
	plaintext, err := GenerateToken()
	if err != nil {
		return "", err
	}

	hash := HashToken(plaintext)
	expiresAt := time.Now().Add(expiresIn)

	_, err = s.db.Exec(`
		INSERT INTO agent_ws_tokens (server_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, serverID, hash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("insert dev token: %w", err)
	}

	return plaintext, nil
}

func (s *Store) ValidateAndLookupServerID(plaintext string) (int64, error) {
	hash := HashToken(plaintext)

	var serverID int64
	err := s.db.QueryRow(`
		SELECT server_id
		FROM agent_ws_tokens
		WHERE token_hash = ?
		  AND revoked_at IS NULL
		  AND expires_at > datetime('now')
		ORDER BY id DESC
		LIMIT 1
	`, hash).Scan(&serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("invalid or expired token")
	}
	if err != nil {
		return 0, fmt.Errorf("lookup dev token: %w", err)
	}

	_, _ = s.db.Exec(`
		UPDATE agent_ws_tokens
		SET last_used_at = datetime('now')
		WHERE token_hash = ?
	`, hash)

	return serverID, nil
}
