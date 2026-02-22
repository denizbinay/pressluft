package backups

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func LocalArtifactPath(storagePath string) string {
	trimmed := strings.TrimSpace(storagePath)
	trimmed = strings.TrimPrefix(trimmed, "s3://")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		trimmed = "backups/unknown/unknown.tar.zst"
	}
	return filepath.Join(os.TempDir(), "pressluft-artifacts", filepath.FromSlash(trimmed))
}

func ChecksumAndSize(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil))), size, nil
}
