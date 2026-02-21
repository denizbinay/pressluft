package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const referencePrefix = "secret://"
const masterKeyFile = ".master.key"

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: strings.TrimSpace(dir)}
}

func (s *Store) Put(ctx context.Context, name string, plaintext []byte) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	normalized := normalizeName(name)
	if normalized == "" {
		return "", errors.New("secret name required")
	}

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("create secrets dir: %w", err)
	}

	key, err := s.loadOrCreateMasterKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	blob := append(nonce, ciphertext...)
	encoded := base64.StdEncoding.EncodeToString(blob)

	path := s.pathForName(normalized)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		return "", fmt.Errorf("write secret file: %w", err)
	}

	return referencePrefix + normalized, nil
}

func (s *Store) Delete(_ context.Context, reference string) error {
	name, err := parseReference(reference)
	if err != nil {
		return err
	}

	if err := os.Remove(s.pathForName(name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove secret file: %w", err)
	}

	return nil
}

func normalizeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(trimmed)
}

func parseReference(reference string) (string, error) {
	trimmed := strings.TrimSpace(reference)
	if trimmed == "" {
		return "", errors.New("secret reference required")
	}

	if !strings.HasPrefix(trimmed, referencePrefix) {
		return "", errors.New("invalid secret reference")
	}

	name := strings.TrimPrefix(trimmed, referencePrefix)
	if normalizeName(name) != name {
		return "", errors.New("invalid secret reference")
	}

	return name, nil
}

func (s *Store) pathForName(name string) string {
	return filepath.Join(s.dir, name+".enc")
}

func (s *Store) loadOrCreateMasterKey() ([]byte, error) {
	path := filepath.Join(s.dir, masterKeyFile)
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != 32 {
			return nil, errors.New("invalid master key length")
		}
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read master key: %w", err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}

	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("write master key: %w", err)
	}

	return key, nil
}
