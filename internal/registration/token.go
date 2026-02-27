package registration

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	var buf strings.Builder
	encoder := base64.NewEncoder(base64.RawURLEncoding, &buf)
	_, err := encoder.Write(bytes)
	if err != nil {
		return "", fmt.Errorf("encode token: %w", err)
	}
	encoder.Close()

	return buf.String(), nil
}

func HashToken(plaintext string) string {
	hash := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", hash)
}
