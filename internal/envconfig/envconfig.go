package envconfig

import (
	"os"
	"path/filepath"
	"strings"
)

type RuntimePaths struct {
	DataDir    string
	DBPath     string
	AgeKeyPath string
	CAKeyPath  string
}

func Resolve() RuntimePaths {
	dataDir := filepath.Clean(defaultDataDir())
	dbPath := strings.TrimSpace(os.Getenv("PRESSLUFT_DB"))
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "pressluft.db")
	}

	ageKeyPath := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH"))
	if ageKeyPath == "" {
		ageKeyPath = DefaultAgeKeyPath()
	}

	caKeyPath := strings.TrimSpace(os.Getenv("PRESSLUFT_CA_KEY_PATH"))
	if caKeyPath == "" {
		caKeyPath = filepath.Join(dataDir, "ca.key")
	}

	return RuntimePaths{
		DataDir:    dataDir,
		DBPath:     filepath.Clean(dbPath),
		AgeKeyPath: filepath.Clean(ageKeyPath),
		CAKeyPath:  filepath.Clean(caKeyPath),
	}
}

func DefaultDataDir() string {
	return filepath.Clean(defaultDataDir())
}

func DefaultAgeKeyPath() string {
	return filepath.Clean(defaultAgeKeyPath())
}
