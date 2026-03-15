package agentauth

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/shared/idutil"
)

type Store struct {
	db *sql.DB
}

type Token struct {
	ID         string
	ServerID   string
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(serverID string, expiresIn time.Duration) (string, error) {
	serverID, err := lookupServerID(s.db, serverID)
	if err != nil {
		return "", err
	}
	plaintext, err := GenerateToken()
	if err != nil {
		return "", err
	}

	hash := HashToken(plaintext)
	expiresAt := time.Now().Add(expiresIn)

	tokenID, err := idutil.New()
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`
		INSERT INTO agent_ws_tokens (id, server_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, tokenID, serverID, hash, expiresAt.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("insert dev token: %w", err)
	}

	return plaintext, nil
}

func (s *Store) ValidateAndLookupServerID(plaintext string) (string, error) {
	hash := HashToken(plaintext)

	var serverID string
	err := s.db.QueryRow(`
		SELECT servers.id
		FROM agent_ws_tokens
		JOIN servers ON servers.id = agent_ws_tokens.server_id
		WHERE token_hash = ?
		  AND revoked_at IS NULL
		  AND expires_at > datetime('now')
		ORDER BY agent_ws_tokens.id DESC
		LIMIT 1
	`, hash).Scan(&serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("invalid or expired token")
	}
	if err != nil {
		return "", fmt.Errorf("lookup dev token: %w", err)
	}

	_, _ = s.db.Exec(`
		UPDATE agent_ws_tokens
		SET last_used_at = datetime('now')
		WHERE token_hash = ?
	`, hash)

	return serverID, nil
}

func lookupServerID(db *sql.DB, serverID string) (string, error) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return "", fmt.Errorf("server_id is required")
	}
	var storedID string
	if err := db.QueryRow(`SELECT id FROM servers WHERE id = ?`, serverID).Scan(&storedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("server %s not found", serverID)
		}
		return "", fmt.Errorf("lookup server id: %w", err)
	}
	return storedID, nil
}
