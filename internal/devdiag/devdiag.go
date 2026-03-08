package devdiag

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pressluft/internal/envconfig"
	"pressluft/internal/pki"
	"pressluft/internal/platform"

	_ "modernc.org/sqlite"
)

type Workflow string

const (
	WorkflowDev Workflow = "dev"
	WorkflowLab Workflow = "lab"
)

type CheckStatus string

const (
	CheckStatusOK      CheckStatus = "ok"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusError   CheckStatus = "error"
)

type Check struct {
	Name   string
	Status CheckStatus
	Detail string
}

type Report struct {
	Runtime                  envconfig.ControlPlaneRuntime
	CallbackURLMode          platform.CallbackURLMode
	DurableReconnectExpected bool
	Checks                   []Check
}

func Inspect(runtime envconfig.ControlPlaneRuntime) Report {
	report := Report{
		Runtime:                  runtime,
		CallbackURLMode:          platform.DetectCallbackURLMode(runtime.ControlPlaneURL),
		DurableReconnectExpected: platform.DetectCallbackURLMode(runtime.ControlPlaneURL) == platform.CallbackURLModeStable,
	}

	allowAgeGenerate := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH")) == ""
	report.add(checkOptionalFile("db", runtime.DBPath, true))
	report.add(checkAgeKey(runtime.AgeKeyPath, allowAgeGenerate))
	report.add(checkOptionalFile("session_key", runtime.SessionSecretPath, true))
	report.add(checkStoredCA(runtime.DBPath, runtime.AgeKeyPath, runtime.CAKeyPath))

	return report
}

func (r Report) HealthyFor(workflow Workflow) bool {
	if workflow == WorkflowLab && r.CallbackURLMode != platform.CallbackURLModeStable {
		return false
	}
	for _, check := range r.Checks {
		if check.Status == CheckStatusError {
			return false
		}
	}
	return true
}

func (r Report) WorkflowIssues(workflow Workflow) []string {
	var issues []string
	switch workflow {
	case WorkflowLab:
		switch r.CallbackURLMode {
		case platform.CallbackURLModeStable:
		case platform.CallbackURLModeEphemeral:
			issues = append(issues, "dev-lab requires a stable PRESSLUFT_CONTROL_PLANE_URL; quick tunnels are session-scoped")
		default:
			issues = append(issues, "dev-lab requires PRESSLUFT_CONTROL_PLANE_URL with a stable public URL")
		}
	case WorkflowDev:
		if r.CallbackURLMode == platform.CallbackURLModeEphemeral {
			issues = append(issues, "remote connectivity is session-scoped in dev; remote agents will not reconnect after control-plane restart")
		}
	}
	for _, check := range r.Checks {
		if check.Status == CheckStatusError {
			issues = append(issues, check.Detail)
		}
	}
	return issues
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
