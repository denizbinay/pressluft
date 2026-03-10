package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"pressluft/internal/envconfig"
	"pressluft/internal/pki"
	"pressluft/internal/security"

	_ "modernc.org/sqlite"
)

type pathStatus struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	IsDir     bool   `json:"is_dir"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type statusOutput struct {
	Mode     string     `json:"mode"`
	DataDir  pathStatus `json:"data_dir"`
	DB       pathStatus `json:"db"`
	DBWAL    pathStatus `json:"db_wal"`
	DBSHM    pathStatus `json:"db_shm"`
	AgeKey   pathStatus `json:"age_key"`
	CAKey    pathStatus `json:"ca_key"`
	Env      any        `json:"env_overrides"`
	Commands []string   `json:"commands"`
}

type tableCount struct {
	Table string `json:"table"`
	Count int64  `json:"count"`
}

type statsOutput struct {
	Mode        string       `json:"mode"`
	DBPath      string       `json:"db_path"`
	TableCounts []tableCount `json:"table_counts"`
}

type healthCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
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
	ID                 int64   `json:"id"`
	EventType          string  `json:"event_type"`
	Category           string  `json:"category"`
	Level              string  `json:"level"`
	ResourceType       *string `json:"resource_type,omitempty"`
	ResourceID         *int64  `json:"resource_id,omitempty"`
	ParentResourceType *string `json:"parent_resource_type,omitempty"`
	ParentResourceID   *int64  `json:"parent_resource_id,omitempty"`
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

func main() {
	args := os.Args[1:]
	command := "status"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	var err error
	switch command {
	case "status":
		err = runStatus()
	case "stats":
		err = runStats()
	case "events":
		err = runEvents(args)
	case "health":
		err = runHealth()
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown command %q", command)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "pressluft-devctl: %v\n", err)
		os.Exit(1)
	}
}

func runStatus() error {
	paths := envconfig.Resolve()
	output := statusOutput{
		Mode:    envconfig.Mode,
		DataDir: statPath(paths.DataDir),
		DB:      statPath(paths.DBPath),
		DBWAL:   statPath(paths.DBPath + "-wal"),
		DBSHM:   statPath(paths.DBPath + "-shm"),
		AgeKey:  statPath(paths.AgeKeyPath),
		CAKey:   statPath(paths.CAKeyPath),
		Env: map[string]string{
			"PRESSLUFT_DB":           strings.TrimSpace(os.Getenv("PRESSLUFT_DB")),
			"PRESSLUFT_AGE_KEY_PATH": strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH")),
			"PRESSLUFT_CA_KEY_PATH":  strings.TrimSpace(os.Getenv("PRESSLUFT_CA_KEY_PATH")),
			"XDG_DATA_HOME":          strings.TrimSpace(os.Getenv("XDG_DATA_HOME")),
		},
		Commands: []string{
			"pressluft-devctl status",
			"pressluft-devctl stats",
			"pressluft-devctl events --limit 20",
			"pressluft-devctl health",
		},
	}
	return writeJSON(output)
}

func runStats() error {
	paths := envconfig.Resolve()
	db, err := openExistingDB(paths.DBPath)
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
		DBPath:      paths.DBPath,
		TableCounts: counts,
	})
}

func runEvents(args []string) error {
	limit, err := parseEventsLimit(args)
	if err != nil {
		return err
	}

	paths := envconfig.Resolve()
	db, err := openExistingDB(paths.DBPath)
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
		DBPath:    paths.DBPath,
		Limit:     limit,
		JobEvents: jobEvents,
		Activity:  activityEvents,
	})
}

func runHealth() error {
	paths := envconfig.Resolve()
	checks := []healthCheck{
		pathCheck("data_dir_exists", paths.DataDir),
		fileCheck("db_exists", paths.DBPath),
		fileCheck("age_key_exists", paths.AgeKeyPath),
		fileCheck("ca_key_exists", paths.CAKeyPath),
	}

	db, dbErr := openExistingDB(paths.DBPath)
	checks = append(checks, resultCheck("db_openable", dbErr))
	if dbErr == nil {
		checks = append(checks, resultCheck("db_ping", db.Ping()))
		_ = db.Close()
	}

	checks = append(checks, resultCheck("age_key_loadable", security.ValidateAgeKey(paths.AgeKeyPath)))
	checks = append(checks, resultCheck("ca_key_loadable", pki.ValidateCAKey(paths.CAKeyPath, paths.AgeKeyPath)))

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

func printUsage() {
	fmt.Println("usage: pressluft-devctl [status|stats|events|health]")
	fmt.Println("  status           show resolved runtime paths and file presence")
	fmt.Println("  stats            show row counts for key runtime tables")
	fmt.Println("  events [--limit N]  show recent job_events and activity rows")
	fmt.Println("  health           verify runtime artifacts can be opened")
}

func writeJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func statPath(path string) pathStatus {
	info, err := os.Stat(path)
	if err != nil {
		return pathStatus{Path: path}
	}
	return pathStatus{
		Path:      path,
		Exists:    true,
		IsDir:     info.IsDir(),
		SizeBytes: info.Size(),
	}
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
		var resourceID sql.NullInt64
		var parentResourceType sql.NullString
		var parentResourceID sql.NullInt64
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
		item.ResourceID = nullInt64Ptr(resourceID)
		item.ParentResourceType = nullStringPtr(parentResourceType)
		item.ParentResourceID = nullInt64Ptr(parentResourceID)
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

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	number := value.Int64
	return &number
}
