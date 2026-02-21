package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/store"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUnauthorized = errors.New("unauthorized")

const passwordHashPBKDF2Prefix = "pbkdf2_sha256:"

type Service struct {
	db *sql.DB
}

type LoginResult struct {
	UserID       string
	SessionToken string
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	var userID string
	var passwordHash string
	var active int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, is_active
		FROM users
		WHERE email = ?
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(email))).Scan(&userID, &passwordHash, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("query user: %w", err)
	}

	if active != 1 || !passwordMatches(passwordHash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := randomToken(32)
	if err != nil {
		return LoginResult{}, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	sessionID, err := randomID("session")
	if err != nil {
		return LoginResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO auth_sessions (id, user_id, session_token, expires_at, created_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, NULL)
		`, sessionID, userID, token, expiresAt.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("insert auth session: %w", err)
		}
		return nil
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{UserID: userID, SessionToken: token}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE session_token = ?
		  AND revoked_at IS NULL
	`, now, token)
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked rows: %w", err)
	}
	if rows == 0 {
		return ErrUnauthorized
	}

	return nil
}

func (s *Service) ValidateSession(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx, `
		SELECT s.user_id
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_token = ?
		  AND s.revoked_at IS NULL
		  AND s.expires_at > ?
		  AND u.is_active = 1
		LIMIT 1
	`, token, time.Now().UTC().Format(time.RFC3339)).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUnauthorized
		}
		return "", fmt.Errorf("validate session: %w", err)
	}

	return userID, nil
}

func passwordMatches(stored, plain string) bool {
	if strings.HasPrefix(stored, passwordHashPBKDF2Prefix) {
		ok, err := pbkdf2Matches(stored, plain)
		return err == nil && ok
	}
	if strings.HasPrefix(stored, "sha256:") {
		hash := sha256.Sum256([]byte(plain))
		candidate := "sha256:" + hex.EncodeToString(hash[:])
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(stored)) == 1
	}

	return subtle.ConstantTimeCompare([]byte(stored), []byte(plain)) == 1
}

// HashPassword returns a strong salted password hash suitable for storage.
// Format: pbkdf2_sha256:<iterations>:<salt_hex>:<dk_hex>
func HashPassword(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", fmt.Errorf("hash password: empty password")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hash password: generate salt: %w", err)
	}

	iterations := 120_000
	dk := pbkdf2Key([]byte(plain), salt, iterations, 32)
	return fmt.Sprintf(
		"%s%d:%s:%s",
		passwordHashPBKDF2Prefix,
		iterations,
		hex.EncodeToString(salt),
		hex.EncodeToString(dk),
	), nil
}

func pbkdf2Matches(stored, plain string) (bool, error) {
	parts := strings.Split(stored, ":")
	if len(parts) != 4 {
		return false, fmt.Errorf("invalid pbkdf2 hash format")
	}
	if parts[0]+":" != passwordHashPBKDF2Prefix {
		return false, fmt.Errorf("invalid pbkdf2 prefix")
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 10_000 {
		return false, fmt.Errorf("invalid pbkdf2 iterations")
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil || len(salt) < 8 {
		return false, fmt.Errorf("invalid pbkdf2 salt")
	}
	expected, err := hex.DecodeString(parts[3])
	if err != nil || len(expected) < 16 {
		return false, fmt.Errorf("invalid pbkdf2 derived key")
	}

	dk := pbkdf2Key([]byte(plain), salt, iter, len(expected))
	return subtle.ConstantTimeCompare(dk, expected) == 1, nil
}

func pbkdf2Key(password, salt []byte, iterations, keyLen int) []byte {
	hLen := sha256.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)

	blockBuf := make([]byte, len(salt)+4)
	copy(blockBuf, salt)

	for block := 1; block <= numBlocks; block++ {
		blockBuf[len(salt)+0] = byte(block >> 24)
		blockBuf[len(salt)+1] = byte(block >> 16)
		blockBuf[len(salt)+2] = byte(block >> 8)
		blockBuf[len(salt)+3] = byte(block)

		u := hmacSHA256(password, blockBuf)
		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			u = hmacSHA256(password, u)
			for j := 0; j < len(t); j++ {
				t[j] ^= u[j]
			}
		}

		out = append(out, t...)
	}

	return out[:keyLen]
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func randomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf)), nil
}
