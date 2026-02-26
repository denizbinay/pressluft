package security

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

func DefaultAgeKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "/etc/pressluft/age.key"
	}
	return filepath.Join(home, ".pressluft", "age.key")
}

func Encrypt(plaintext []byte) (string, string, error) {
	recipients, keyID, err := loadRecipients(ageKeyPath())
	if err != nil {
		return "", "", err
	}

	var out bytes.Buffer
	armorWriter := armor.NewWriter(&out)
	writer, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		_ = armorWriter.Close()
		return "", "", fmt.Errorf("encrypt: %w", err)
	}
	if _, err := writer.Write(plaintext); err != nil {
		_ = writer.Close()
		_ = armorWriter.Close()
		return "", "", fmt.Errorf("encrypt write: %w", err)
	}
	if err := writer.Close(); err != nil {
		_ = armorWriter.Close()
		return "", "", fmt.Errorf("encrypt close: %w", err)
	}
	if err := armorWriter.Close(); err != nil {
		return "", "", fmt.Errorf("encrypt armor close: %w", err)
	}

	return out.String(), keyID, nil
}

func Decrypt(ciphertext string) ([]byte, error) {
	identities, err := loadIdentities(ageKeyPath())
	if err != nil {
		return nil, err
	}

	reader := armor.NewReader(strings.NewReader(ciphertext))
	decrypted, err := age.Decrypt(reader, identities...)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	data, err := io.ReadAll(decrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt read: %w", err)
	}
	return data, nil
}

func loadIdentities(path string) ([]age.Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read age identity file: %w", err)
	}
	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse age identities: %w", err)
	}
	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found")
	}
	return identities, nil
}

func loadRecipients(path string) ([]age.Recipient, string, error) {
	identities, err := loadIdentities(path)
	if err != nil {
		return nil, "", err
	}
	recipients := make([]age.Recipient, 0, len(identities))
	for _, identity := range identities {
		switch typed := identity.(type) {
		case *age.X25519Identity:
			recipients = append(recipients, typed.Recipient())
		default:
			return nil, "", fmt.Errorf("unsupported age identity type: %T", identity)
		}
	}
	if len(recipients) == 0 {
		return nil, "", fmt.Errorf("no age recipients derived")
	}
	keyID := keyIDFromRecipients(recipients)
	return recipients, keyID, nil
}

func ageKeyPath() string {
	path := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH"))
	if path == "" {
		return DefaultAgeKeyPath()
	}
	return path
}

func keyIDFromRecipients(recipients []age.Recipient) string {
	values := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		values = append(values, fmt.Sprint(recipient))
	}
	sort.Strings(values)
	sum := sha256.Sum256([]byte(strings.Join(values, ",")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func EnsureAgeKey(path string, allowGenerate bool) (bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, fmt.Errorf("age key path is empty")
	}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("age key path is a directory: %s", path)
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat age key: %w", err)
	}
	if !allowGenerate {
		return false, fmt.Errorf("age key file missing: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("create age key directory: %w", err)
	}

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return false, fmt.Errorf("generate age identity: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("create age key file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(identity.String() + "\n"); err != nil {
		return false, fmt.Errorf("write age key file: %w", err)
	}

	return true, nil
}
