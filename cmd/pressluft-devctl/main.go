package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"pressluft/internal/devdiag"
	"pressluft/internal/envconfig"
	"pressluft/internal/pki"
	"pressluft/internal/security"

	_ "modernc.org/sqlite"
)

type tableCount struct {
	Table string `json:"table"`
	Count int64  `json:"count"`
}

type healthCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type statsOutput struct {
	Mode        string       `json:"mode"`
	DBPath      string       `json:"db_path"`
	TableCounts []tableCount `json:"table_counts"`
}

type healthOutput struct {
	Mode   string        `json:"mode"`
	Checks []healthCheck `json:"checks"`
}

type jobEvent struct {
	ID        int64   `json:"id"`
	JobID     int64   `json:"job_id"`
	Seq       int64   `json:"seq"`
	EventType string  `json:"event_type"`
	Level     string  `json:"level"`
	StepKey   *string `json:"step_key,omitempty"`
	Status    *string `json:"status,omitempty"`
	Message   string  `json:"message"`
	Payload   *string `json:"payload,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type activityEvent struct {
	ID                 string  `json:"id"`
	EventType          string  `json:"event_type"`
	Category           string  `json:"category"`
	Level              string  `json:"level"`
	ResourceType       *string `json:"resource_type,omitempty"`
	ResourceID         *string `json:"resource_id,omitempty"`
	ParentResourceType *string `json:"parent_resource_type,omitempty"`
	ParentResourceID   *string `json:"parent_resource_id,omitempty"`
	ActorType          string  `json:"actor_type"`
	ActorID            *string `json:"actor_id,omitempty"`
	Title              string  `json:"title"`
	Message            *string `json:"message,omitempty"`
	Payload            *string `json:"payload,omitempty"`
	RequiresAttention  bool    `json:"requires_attention"`
	ReadAt             *string `json:"read_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

type eventsOutput struct {
	Mode      string          `json:"mode"`
	DBPath    string          `json:"db_path"`
	Limit     int             `json:"limit"`
	JobEvents []jobEvent      `json:"job_events"`
	Activity  []activityEvent `json:"activity"`
}

type sshAccessTarget struct {
	ID   string
	Name string
	IPv4 string
	IPv6 string
	Key  string
}

func main() {
	args := os.Args[1:]
	command := "status"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	cwd, _ := os.Getwd()
	runtime, err := envconfig.ResolveControlPlaneRuntime(true, cwd)
	if err != nil {
		exitErr(fmt.Errorf("resolve control-plane runtime: %w", err))
	}

	switch command {
	case "status":
		printReport(devdiag.Inspect(runtime))
	case "preflight":
		workflow, err := parseWorkflow(args)
		if err != nil {
			exitErr(err)
		}
		report := devdiag.Inspect(runtime)
		printReport(report)
		if !report.HealthyFor(workflow) {
			fmt.Println()
			fmt.Printf("workflow=%s preflight failed\n", workflow)
			for _, issue := range report.WorkflowIssues(workflow) {
				fmt.Printf("- %s\n", issue)
			}
			fmt.Println("Suggested next steps: make dev-status ; make dev-reset CONFIRM=1")
			os.Exit(1)
		}
	case "stats":
		if err := runStats(runtime); err != nil {
			exitErr(err)
		}
	case "events":
		if err := runEvents(runtime, args); err != nil {
			exitErr(err)
		}
	case "health":
		if err := runHealth(runtime); err != nil {
			exitErr(err)
		}
	case "server-ssh":
		if err := runServerSSH(runtime, args); err != nil {
			exitErr(err)
		}
	case "reset":
		if err := reset(runtime, args); err != nil {
			exitErr(err)
		}
	case "help", "-h", "--help":
		usage()
	default:
		exitErr(fmt.Errorf("unknown command %q", command))
	}
}

func usage() {
	fmt.Println("pressluft-devctl")
	fmt.Println("  status                   Inspect local dev state and callback durability")
	fmt.Println("  preflight --workflow=X   Validate local state for dev or lab workflow")
	fmt.Println("  stats                    Show row counts for key runtime tables")
	fmt.Println("  events [--limit N]       Show recent job_events and activity rows")
	fmt.Println("  health                   Verify runtime artifacts can be opened")
	fmt.Println("  server-ssh TARGET        Print or execute one-off SSH access for a server")
	fmt.Println("  reset --force            Remove the local Pressluft state bundle")
}

func runStats(runtime envconfig.ControlPlaneRuntime) error {
	db, err := openExistingDB(runtime.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	tables := []string{
		"providers",
		"servers",
		"jobs",
		"job_steps",
		"job_events",
		"job_checkpoints",
		"activity",
		"ca_certificates",
		"node_certificates",
		"registration_tokens",
		"agent_ws_tokens",
	}

	counts := make([]tableCount, 0, len(tables))
	for _, table := range tables {
		count, err := countRows(db, table)
		if err != nil {
			return fmt.Errorf("count %s: %w", table, err)
		}
		counts = append(counts, tableCount{Table: table, Count: count})
	}

	return writeJSON(statsOutput{
		Mode:        envconfig.Mode,
		DBPath:      runtime.DBPath,
		TableCounts: counts,
	})
}

func runEvents(runtime envconfig.ControlPlaneRuntime, args []string) error {
	limit, err := parseEventsLimit(args)
	if err != nil {
		return err
	}

	db, err := openExistingDB(runtime.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	jobEvents, err := recentJobEvents(db, limit)
	if err != nil {
		return err
	}
	activityEvents, err := recentActivity(db, limit)
	if err != nil {
		return err
	}

	return writeJSON(eventsOutput{
		Mode:      envconfig.Mode,
		DBPath:    runtime.DBPath,
		Limit:     limit,
		JobEvents: jobEvents,
		Activity:  activityEvents,
	})
}

func runHealth(runtime envconfig.ControlPlaneRuntime) error {
	checks := []healthCheck{
		pathCheck("data_dir_exists", runtime.DataDir),
		fileCheck("db_exists", runtime.DBPath),
		fileCheck("age_key_exists", runtime.AgeKeyPath),
		fileCheck("ca_key_exists", runtime.CAKeyPath),
		fileCheck("session_key_exists", runtime.SessionSecretPath),
	}

	db, dbErr := openExistingDB(runtime.DBPath)
	checks = append(checks, resultCheck("db_openable", dbErr))
	if dbErr == nil {
		checks = append(checks, resultCheck("db_ping", db.Ping()))
		_ = db.Close()
	}

	checks = append(checks, resultCheck("age_key_loadable", security.ValidateAgeKey(runtime.AgeKeyPath)))
	checks = append(checks, resultCheck("ca_key_loadable", pki.ValidateCAKey(runtime.CAKeyPath, runtime.AgeKeyPath)))

	if err := writeJSON(healthOutput{Mode: envconfig.Mode, Checks: checks}); err != nil {
		return err
	}

	for _, check := range checks {
		if !check.OK {
			return errors.New("one or more health checks failed")
		}
	}
	return nil
}

func runServerSSH(runtime envconfig.ControlPlaneRuntime, args []string) error {
	target, execSSH, printKey, err := parseServerSSHArgs(args)
	if err != nil {
		return err
	}

	db, err := openExistingDB(runtime.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	access, err := lookupSSHAccessTarget(db, target)
	if err != nil {
		return err
	}
	decryptedKey, err := security.Decrypt(access.Key)
	if err != nil {
		return fmt.Errorf("decrypt server ssh key: %w", err)
	}

	if printKey {
		_, err := os.Stdout.Write(decryptedKey)
		return err
	}

	host := strings.TrimSpace(access.IPv4)
	if host == "" {
		host = strings.TrimSpace(access.IPv6)
	}
	if host == "" {
		return fmt.Errorf("server %q has no recorded IP address", access.ID)
	}

	keyFile, err := os.CreateTemp("", "pressluft-server-ssh-*.key")
	if err != nil {
		return fmt.Errorf("create temp key file: %w", err)
	}
	defer keyFile.Close()
	if err := keyFile.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod temp key file: %w", err)
	}
	if _, err := keyFile.Write(decryptedKey); err != nil {
		return fmt.Errorf("write temp key file: %w", err)
	}

	sshArgs := []string{"-i", keyFile.Name(), "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "root@" + host}
	if execSSH {
		cmd := exec.Command("ssh", sshArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run ssh: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nSSH key left at %s\n", keyFile.Name())
		return nil
	}

	fmt.Printf("server: %s (%s)\n", access.Name, access.ID)
	fmt.Printf("host: %s\n", host)
	fmt.Printf("key_file: %s\n", keyFile.Name())
	fmt.Printf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@%s\n", keyFile.Name(), host)
	return nil
}

func parseServerSSHArgs(args []string) (target string, execSSH bool, printKey bool, err error) {
	for _, arg := range args {
		switch arg {
		case "--exec":
			execSSH = true
		case "--print-key":
			printKey = true
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, false, fmt.Errorf("unknown server-ssh argument %q", arg)
			}
			if target != "" {
				return "", false, false, fmt.Errorf("server-ssh accepts exactly one TARGET")
			}
			target = strings.TrimSpace(arg)
		}
	}
	if target == "" {
		return "", false, false, fmt.Errorf("server-ssh requires TARGET")
	}
	if execSSH && printKey {
		return "", false, false, fmt.Errorf("--exec and --print-key cannot be combined")
	}
	return target, execSSH, printKey, nil
}

func lookupSSHAccessTarget(db *sql.DB, target string) (*sshAccessTarget, error) {
	rows, err := db.Query(`
		SELECT s.id, s.name, s.ipv4, s.ipv6, k.private_key_encrypted
		FROM servers s
		JOIN server_keys k ON k.server_id = s.id
		WHERE s.id = ? OR s.name = ?
		ORDER BY s.created_at DESC
	`, target, target)
	if err != nil {
		return nil, fmt.Errorf("query server ssh access: %w", err)
	}
	defer rows.Close()

	var matches []sshAccessTarget
	for rows.Next() {
		var item sshAccessTarget
		var ipv4 sql.NullString
		var ipv6 sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &ipv4, &ipv6, &item.Key); err != nil {
			return nil, fmt.Errorf("scan server ssh access: %w", err)
		}
		if ipv4.Valid {
			item.IPv4 = ipv4.String
		}
		if ipv6.Valid {
			item.IPv6 = ipv6.String
		}
		matches = append(matches, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate server ssh access: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no server with stored SSH key matched %q", target)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, fmt.Sprintf("%s (%s)", match.Name, match.ID))
		}
		return nil, fmt.Errorf("multiple servers matched %q: %s", target, strings.Join(ids, ", "))
	}
	return &matches[0], nil
}

func parseWorkflow(args []string) (devdiag.Workflow, error) {
	workflow := devdiag.WorkflowDev
	for _, arg := range args {
		if strings.HasPrefix(arg, "--workflow=") {
			value := strings.TrimPrefix(arg, "--workflow=")
			switch value {
			case string(devdiag.WorkflowDev):
				workflow = devdiag.WorkflowDev
			case string(devdiag.WorkflowLab):
				workflow = devdiag.WorkflowLab
			default:
				return "", fmt.Errorf("unsupported workflow %q", value)
			}
		}
	}
	return workflow, nil
}

func parseEventsLimit(args []string) (int, error) {
	limit := 20
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 >= len(args) {
				return 0, fmt.Errorf("--limit requires a value")
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return 0, fmt.Errorf("invalid --limit value %q", args[i+1])
			}
			limit = parsed
			i++
		default:
			return 0, fmt.Errorf("unknown events argument %q", args[i])
		}
	}
	return limit, nil
}

func printReport(report devdiag.Report) {
	fmt.Println("Pressluft dev state")
	fmt.Printf("execution_mode: %s\n", report.Runtime.ExecutionMode)
	fmt.Printf("data_dir: %s\n", report.Runtime.DataDir)
	fmt.Printf("db_path: %s\n", report.Runtime.DBPath)
	fmt.Printf("age_key_path: %s\n", report.Runtime.AgeKeyPath)
	fmt.Printf("ca_key_path: %s\n", report.Runtime.CAKeyPath)
	fmt.Printf("session_key_path: %s\n", report.Runtime.SessionSecretPath)
	if strings.TrimSpace(report.Runtime.ControlPlaneURL) == "" {
		fmt.Println("callback_url: <unset>")
	} else {
		fmt.Printf("callback_url: %s\n", report.Runtime.ControlPlaneURL)
	}
	fmt.Printf("callback_url_mode: %s\n", report.CallbackURLMode)
	if report.DurableReconnectExpected {
		fmt.Println("durable_reconnect: yes")
	} else {
		fmt.Println("durable_reconnect: no")
	}
	fmt.Println("checks:")
	for _, check := range report.Checks {
		fmt.Printf("- [%s] %s: %s\n", check.Status, check.Name, check.Detail)
	}
}

func reset(runtime envconfig.ControlPlaneRuntime, args []string) error {
	if len(args) != 1 || args[0] != "--force" {
		return fmt.Errorf("reset requires --force")
	}
	paths := []string{
		runtime.DBPath,
		runtime.DBPath + "-wal",
		runtime.DBPath + "-shm",
		runtime.AgeKeyPath,
		runtime.CAKeyPath,
		runtime.SessionSecretPath,
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	_ = os.Remove(runtime.DataDir)
	fmt.Printf("Removed Pressluft local state bundle:\n- %s\n- %s\n- %s\n- %s\n", runtime.DBPath, runtime.AgeKeyPath, runtime.CAKeyPath, runtime.SessionSecretPath)
	return nil
}

func writeJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func openExistingDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat db: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func countRows(db *sql.DB, table string) (int64, error) {
	row := db.QueryRow("SELECT COUNT(*) FROM " + table)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func recentJobEvents(db *sql.DB, limit int) ([]jobEvent, error) {
	rows, err := db.Query(`
		SELECT id, job_id, seq, event_type, level, step_key, status, message, payload, created_at
		FROM job_events
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query job_events: %w", err)
	}
	defer rows.Close()

	var events []jobEvent
	for rows.Next() {
		var item jobEvent
		var stepKey sql.NullString
		var status sql.NullString
		var payload sql.NullString
		if err := rows.Scan(&item.ID, &item.JobID, &item.Seq, &item.EventType, &item.Level, &stepKey, &status, &item.Message, &payload, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan job_events: %w", err)
		}
		item.StepKey = nullStringPtr(stepKey)
		item.Status = nullStringPtr(status)
		item.Payload = nullStringPtr(payload)
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job_events: %w", err)
	}
	return events, nil
}

func recentActivity(db *sql.DB, limit int) ([]activityEvent, error) {
	rows, err := db.Query(`
		SELECT id, event_type, category, level, resource_type, resource_id, parent_resource_type, parent_resource_id,
		       actor_type, actor_id, title, message, payload, requires_attention, read_at, created_at
		FROM activity
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query activity: %w", err)
	}
	defer rows.Close()

	var events []activityEvent
	for rows.Next() {
		var item activityEvent
		var resourceType sql.NullString
		var resourceID sql.NullString
		var parentResourceType sql.NullString
		var parentResourceID sql.NullString
		var actorID sql.NullString
		var message sql.NullString
		var payload sql.NullString
		var readAt sql.NullString
		var requiresAttention int64
		if err := rows.Scan(
			&item.ID,
			&item.EventType,
			&item.Category,
			&item.Level,
			&resourceType,
			&resourceID,
			&parentResourceType,
			&parentResourceID,
			&item.ActorType,
			&actorID,
			&item.Title,
			&message,
			&payload,
			&requiresAttention,
			&readAt,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		item.ResourceType = nullStringPtr(resourceType)
		item.ResourceID = nullStringPtr(resourceID)
		item.ParentResourceType = nullStringPtr(parentResourceType)
		item.ParentResourceID = nullStringPtr(parentResourceID)
		item.ActorID = nullStringPtr(actorID)
		item.Message = nullStringPtr(message)
		item.Payload = nullStringPtr(payload)
		item.ReadAt = nullStringPtr(readAt)
		item.RequiresAttention = requiresAttention != 0
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity: %w", err)
	}
	return events, nil
}

func fileCheck(name, path string) healthCheck {
	if info, err := os.Stat(path); err != nil {
		return healthCheck{Name: name, Detail: err.Error()}
	} else if info.IsDir() {
		return healthCheck{Name: name, Detail: "path is a directory"}
	}
	return healthCheck{Name: name, OK: true}
}

func pathCheck(name, path string) healthCheck {
	if _, err := os.Stat(path); err != nil {
		return healthCheck{Name: name, Detail: err.Error()}
	}
	return healthCheck{Name: name, OK: true}
}

func resultCheck(name string, err error) healthCheck {
	if err != nil {
		return healthCheck{Name: name, Detail: err.Error()}
	}
	return healthCheck{Name: name, OK: true}
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "pressluft-devctl: %v\n", err)
	os.Exit(1)
}
