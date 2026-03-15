package devdiag

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pressluft/internal/infra/pki"
	"pressluft/internal/platform"
	"pressluft/internal/shared/envconfig"
	"pressluft/internal/shared/security"

	_ "modernc.org/sqlite"
)

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
)

type Check struct {
	Name   string      `json:"name"`
	Status CheckStatus `json:"status"`
	Detail string      `json:"detail"`
}

type Report struct {
	Runtime                  envconfig.ControlPlaneRuntime `json:"-"`
	CallbackURLMode          platform.CallbackURLMode      `json:"callback_url_mode"`
	DurableReconnectExpected bool                          `json:"durable_reconnect"`
	Checks                   []Check                       `json:"checks"`
}

// Inspect runs all diagnostic checks and returns a consolidated report.
// This is the single source of truth for system health.
func Inspect(runtime envconfig.ControlPlaneRuntime) Report {
	report := Report{
		Runtime:                  runtime,
		CallbackURLMode:          platform.DetectCallbackURLMode(runtime.ControlPlaneURL),
		DurableReconnectExpected: platform.DetectCallbackURLMode(runtime.ControlPlaneURL) == platform.CallbackURLModeStable,
	}

	// File existence checks.
	allowAgeGenerate := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH")) == ""
	report.add(checkOptionalFile("db", runtime.DBPath, true))
	report.add(checkAgeKey(runtime.AgeKeyPath, allowAgeGenerate))
	report.add(checkOptionalFile("session_key", runtime.SessionSecretPath, true))

	// Deep checks: verify artifacts are loadable, not just present.
	report.add(checkDBOpenable(runtime.DBPath))
	report.add(checkAgeKeyLoadable(runtime.AgeKeyPath))
	report.add(checkCAKeyLoadable(runtime.CAKeyPath, runtime.AgeKeyPath))

	// CA state consistency (cert in DB matches key on disk).
	report.add(checkStoredCA(runtime.DBPath, runtime.AgeKeyPath, runtime.CAKeyPath))

	return report
}

func (r Report) Healthy() bool {
	for _, check := range r.Checks {
		if check.Status == CheckStatusError {
			return false
		}
	}
	return true
}

func (r Report) Issues() []string {
	var issues []string
	for _, check := range r.Checks {
		if check.Status == CheckStatusError {
			issues = append(issues, check.Detail)
		}
	}
	return issues
}

// JSON returns the report as indented JSON bytes.
func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func checkAgeKey(path string, allowGenerate bool) Check {
	return checkFile("age_key", path, !allowGenerate)
}

func checkOptionalFile(name, path string, treatMissingAsWarning bool) Check {
	return checkFile(name, path, !treatMissingAsWarning)
}

func checkFile(name, path string, required bool) Check {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return Check{Name: name, Status: CheckStatusError, Detail: fmt.Sprintf("%s path is a directory: %s", name, path)}
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return Check{Name: name, Status: CheckStatusError, Detail: fmt.Sprintf("read %s: %v", name, openErr)}
		}
		_ = file.Close()
		return Check{Name: name, Status: CheckStatusOK, Detail: fmt.Sprintf("%s ready at %s", name, path)}
	}
	if !os.IsNotExist(err) {
		return Check{Name: name, Status: CheckStatusError, Detail: fmt.Sprintf("stat %s: %v", name, err)}
	}
	if required {
		return Check{Name: name, Status: CheckStatusError, Detail: fmt.Sprintf("%s is missing: %s", name, path)}
	}
	return Check{Name: name, Status: CheckStatusWarning, Detail: fmt.Sprintf("%s is missing and will be created on startup: %s", name, path)}
}

func checkDBOpenable(dbPath string) Check {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "db_openable", Status: CheckStatusWarning, Detail: "database does not exist yet; will be created on startup"}
		}
		return Check{Name: "db_openable", Status: CheckStatusError, Detail: fmt.Sprintf("stat db: %v", err)}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return Check{Name: "db_openable", Status: CheckStatusError, Detail: fmt.Sprintf("open db: %v", err)}
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return Check{Name: "db_openable", Status: CheckStatusError, Detail: fmt.Sprintf("ping db: %v", err)}
	}
	return Check{Name: "db_openable", Status: CheckStatusOK, Detail: "database opens and responds to ping"}
}

func checkAgeKeyLoadable(path string) Check {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "age_key_loadable", Status: CheckStatusWarning, Detail: "age key does not exist yet; will be generated on startup"}
		}
		return Check{Name: "age_key_loadable", Status: CheckStatusError, Detail: fmt.Sprintf("stat age key: %v", err)}
	}
	if err := security.ValidateAgeKey(path); err != nil {
		return Check{Name: "age_key_loadable", Status: CheckStatusError, Detail: fmt.Sprintf("age key is invalid: %v", err)}
	}
	return Check{Name: "age_key_loadable", Status: CheckStatusOK, Detail: "age key parses correctly"}
}

func checkCAKeyLoadable(caKeyPath, ageKeyPath string) Check {
	if _, err := os.Stat(caKeyPath); err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "ca_key_loadable", Status: CheckStatusWarning, Detail: "CA key does not exist yet; will be created on startup"}
		}
		return Check{Name: "ca_key_loadable", Status: CheckStatusError, Detail: fmt.Sprintf("stat CA key: %v", err)}
	}
	if _, err := os.Stat(ageKeyPath); err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "ca_key_loadable", Status: CheckStatusWarning, Detail: "age key missing; cannot verify CA key"}
		}
		return Check{Name: "ca_key_loadable", Status: CheckStatusError, Detail: fmt.Sprintf("stat age key for CA verification: %v", err)}
	}
	if err := pki.ValidateCAKey(caKeyPath, ageKeyPath); err != nil {
		return Check{Name: "ca_key_loadable", Status: CheckStatusError, Detail: fmt.Sprintf("CA key decryption failed: %v", err)}
	}
	return Check{Name: "ca_key_loadable", Status: CheckStatusOK, Detail: "CA key decrypts successfully"}
}

func checkStoredCA(dbPath, ageKeyPath, caKeyPath string) Check {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return Check{Name: "ca_state", Status: CheckStatusWarning, Detail: "database is missing, so no CA state exists yet"}
		}
		return Check{Name: "ca_state", Status: CheckStatusError, Detail: fmt.Sprintf("stat db: %v", err)}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return Check{Name: "ca_state", Status: CheckStatusError, Detail: fmt.Sprintf("open db for CA inspection: %v", err)}
	}
	defer db.Close()

	present, err := pki.ValidateStoredCA(db, ageKeyPath, caKeyPath)
	if err != nil {
		return Check{Name: "ca_state", Status: CheckStatusError, Detail: fmt.Sprintf("stored CA state is inconsistent: %v", err)}
	}
	if !present {
		return Check{Name: "ca_state", Status: CheckStatusWarning, Detail: fmt.Sprintf("no stored CA found in %s; startup will create one", filepath.Clean(dbPath))}
	}
	return Check{Name: "ca_state", Status: CheckStatusOK, Detail: fmt.Sprintf("stored CA decrypts with age key and is ready at %s", caKeyPath)}
}

func (r *Report) add(check Check) {
	r.Checks = append(r.Checks, check)
}
