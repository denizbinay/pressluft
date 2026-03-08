package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
)

var (
	ErrServerActionConflict = errors.New("server action already in progress")
	ErrServerDeleting       = errors.New("server is deleting")
	ErrServerDeleted        = errors.New("server is deleted")
)

// StoredServer is a persisted server node record.
type StoredServer struct {
	ID               int64                 `json:"id"`
	ProviderID       int64                 `json:"provider_id"`
	ProviderType     string                `json:"provider_type"`
	ProviderServerID string                `json:"provider_server_id,omitempty"`
	IPv4             string                `json:"ipv4,omitempty"`
	IPv6             string                `json:"ipv6,omitempty"`
	Name             string                `json:"name"`
	Location         string                `json:"location"`
	ServerType       string                `json:"server_type"`
	Image            string                `json:"image"`
	ProfileKey       string                `json:"profile_key"`
	Status           platform.ServerStatus `json:"status"`
	SetupState       platform.SetupState   `json:"setup_state"`
	SetupLastError   string                `json:"setup_last_error,omitempty"`
	ActionID         string                `json:"action_id,omitempty"`
	ActionStatus     string                `json:"action_status,omitempty"`
	HasKey           bool                  `json:"has_key"`
	NodeStatus       platform.NodeStatus   `json:"node_status,omitempty"`
	NodeLastSeen     string                `json:"node_last_seen,omitempty"`
	NodeVersion      string                `json:"node_version,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

// CreateServerNodeInput is required to create a server record.
type CreateServerNodeInput struct {
	ProviderID   int64
	ProviderType string
	Name         string
	Location     string
	ServerType   string
	Image        string
	ProfileKey   string
	Status       platform.ServerStatus
}

// ServerStore provides persistence for server records.
type ServerStore struct {
	db *sql.DB
}

type QueueServerJobInput struct {
	ServerID int64
	Kind     string
	Payload  string
}

func NewServerStore(db *sql.DB) *ServerStore {
	return &ServerStore{db: db}
}

func (s *ServerStore) Create(ctx context.Context, in CreateServerNodeInput) (int64, error) {
	if err := validateCreateServerNodeInput(in); err != nil {
		return 0, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO servers (
			provider_id, provider_type, name, location, server_type, image, profile_key, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ProviderID,
		in.ProviderType,
		in.Name,
		in.Location,
		in.ServerType,
		in.Image,
		in.ProfileKey,
		in.Status,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert server: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("server insert id: %w", err)
	}

	return id, nil
}

func (s *ServerStore) List(ctx context.Context) ([]StoredServer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, s.provider_id, s.provider_type, s.provider_server_id, s.ipv4, s.ipv6, s.name, s.location, s.server_type, s.image, s.profile_key, s.status, s.setup_state, s.setup_last_error, s.action_id, s.action_status, s.node_status, s.node_last_seen, s.node_version, s.created_at, s.updated_at,
		 CASE WHEN k.server_id IS NULL THEN 0 ELSE 1 END AS has_key
		 FROM servers s
		 LEFT JOIN server_keys k ON k.server_id = s.id
		 ORDER BY s.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}
	defer rows.Close()

	var out []StoredServer
	for rows.Next() {
		var (
			srv              StoredServer
			status           string
			setupState       string
			providerServerID sql.NullString
			ipv4             sql.NullString
			ipv6             sql.NullString
			actionID         sql.NullString
			actionStatus     sql.NullString
			setupLastError   sql.NullString
			nodeStatus       sql.NullString
			nodeLastSeen     sql.NullString
			nodeVersion      sql.NullString
			hasKey           int
		)
		if err := rows.Scan(
			&srv.ID,
			&srv.ProviderID,
			&srv.ProviderType,
			&providerServerID,
			&ipv4,
			&ipv6,
			&srv.Name,
			&srv.Location,
			&srv.ServerType,
			&srv.Image,
			&srv.ProfileKey,
			&status,
			&setupState,
			&setupLastError,
			&actionID,
			&actionStatus,
			&nodeStatus,
			&nodeLastSeen,
			&nodeVersion,
			&srv.CreatedAt,
			&srv.UpdatedAt,
			&hasKey,
		); err != nil {
			return nil, fmt.Errorf("scan server: %w", err)
		}
		srv.ProviderServerID = nullStringValue(providerServerID)
		srv.IPv4 = nullStringValue(ipv4)
		srv.IPv6 = nullStringValue(ipv6)
		srv.ActionID = nullStringValue(actionID)
		srv.ActionStatus = nullStringValue(actionStatus)
		srv.SetupLastError = nullStringValue(setupLastError)
		normalizedStatus, err := platform.NormalizeServerStatus(status)
		if err != nil {
			return nil, fmt.Errorf("scan server status: %w", err)
		}
		srv.Status = normalizedStatus
		normalizedSetupState, err := platform.NormalizeSetupState(setupState)
		if err != nil {
			return nil, fmt.Errorf("scan setup state: %w", err)
		}
		srv.SetupState = normalizedSetupState
		srv.NodeStatus, err = normalizeStoredNodeStatus(nodeStatus)
		if err != nil {
			return nil, fmt.Errorf("scan node status: %w", err)
		}
		srv.NodeLastSeen = nullStringValue(nodeLastSeen)
		srv.NodeVersion = nullStringValue(nodeVersion)
		srv.HasKey = hasKey != 0
		out = append(out, srv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate servers: %w", err)
	}

	return out, nil
}

func (s *ServerStore) UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus string, status platform.ServerStatus, ipv4, ipv6 string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if _, err := platform.NormalizeServerStatus(string(status)); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers
		 SET provider_server_id = ?, action_id = ?, action_status = ?, status = ?, ipv4 = ?, ipv6 = ?, updated_at = ?
		 WHERE id = ?`,
		providerServerID,
		actionID,
		actionStatus,
		string(status),
		ipv4,
		ipv6,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update server provisioning: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

func validateCreateServerNodeInput(in CreateServerNodeInput) error {
	if in.ProviderID <= 0 {
		return fmt.Errorf("provider_id must be greater than zero")
	}
	if strings.TrimSpace(in.ProviderType) == "" {
		return fmt.Errorf("provider_type is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.Location) == "" {
		return fmt.Errorf("location is required")
	}
	if strings.TrimSpace(in.ServerType) == "" {
		return fmt.Errorf("server_type is required")
	}
	if strings.TrimSpace(in.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if strings.TrimSpace(in.ProfileKey) == "" {
		return fmt.Errorf("profile_key is required")
	}
	if _, err := platform.NormalizeServerStatus(string(in.Status)); err != nil {
		return err
	}
	return nil
}

// GetByID returns a server by its ID.
func (s *ServerStore) GetByID(ctx context.Context, id int64) (*StoredServer, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id must be greater than zero")
	}

	var (
		srv              StoredServer
		status           string
		setupState       string
		providerServerID sql.NullString
		ipv4             sql.NullString
		ipv6             sql.NullString
		actionID         sql.NullString
		actionStatus     sql.NullString
		setupLastError   sql.NullString
		nodeStatus       sql.NullString
		nodeLastSeen     sql.NullString
		nodeVersion      sql.NullString
		hasKey           int
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT s.id, s.provider_id, s.provider_type, s.provider_server_id, s.ipv4, s.ipv6, s.name, s.location, s.server_type, s.image, s.profile_key, s.status, s.setup_state, s.setup_last_error, s.action_id, s.action_status, s.node_status, s.node_last_seen, s.node_version, s.created_at, s.updated_at,
		 CASE WHEN k.server_id IS NULL THEN 0 ELSE 1 END AS has_key
		 FROM servers s
		 LEFT JOIN server_keys k ON k.server_id = s.id
		 WHERE s.id = ?`,
		id,
	).Scan(
		&srv.ID,
		&srv.ProviderID,
		&srv.ProviderType,
		&providerServerID,
		&ipv4,
		&ipv6,
		&srv.Name,
		&srv.Location,
		&srv.ServerType,
		&srv.Image,
		&srv.ProfileKey,
		&status,
		&setupState,
		&setupLastError,
		&actionID,
		&actionStatus,
		&nodeStatus,
		&nodeLastSeen,
		&nodeVersion,
		&srv.CreatedAt,
		&srv.UpdatedAt,
		&hasKey,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server %d not found", id)
		}
		return nil, fmt.Errorf("get server: %w", err)
	}

	srv.ProviderServerID = nullStringValue(providerServerID)
	srv.IPv4 = nullStringValue(ipv4)
	srv.IPv6 = nullStringValue(ipv6)
	srv.ActionID = nullStringValue(actionID)
	srv.ActionStatus = nullStringValue(actionStatus)
	srv.SetupLastError = nullStringValue(setupLastError)
	srv.Status, err = platform.NormalizeServerStatus(status)
	if err != nil {
		return nil, fmt.Errorf("get server status: %w", err)
	}
	srv.SetupState, err = platform.NormalizeSetupState(setupState)
	if err != nil {
		return nil, fmt.Errorf("get setup state: %w", err)
	}
	srv.NodeStatus, err = normalizeStoredNodeStatus(nodeStatus)
	if err != nil {
		return nil, fmt.Errorf("get node status: %w", err)
	}
	srv.NodeLastSeen = nullStringValue(nodeLastSeen)
	srv.NodeVersion = nullStringValue(nodeVersion)
	srv.HasKey = hasKey != 0

	return &srv, nil
}

// UpdateStatus updates only the status field of a server.
func (s *ServerStore) UpdateStatus(ctx context.Context, id int64, status platform.ServerStatus) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if _, err := platform.NormalizeServerStatus(string(status)); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`,
		string(status),
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update server status: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

func (s *ServerStore) UpdateSetupState(ctx context.Context, id int64, setupState platform.SetupState, setupLastError string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if _, err := platform.NormalizeSetupState(string(setupState)); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET setup_state = ?, setup_last_error = ?, updated_at = ? WHERE id = ?`,
		string(setupState),
		nullableServerString(setupLastError),
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update server setup state: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

// UpdateServerType updates the server_type field of a server.
func (s *ServerStore) UpdateServerType(ctx context.Context, id int64, serverType string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if strings.TrimSpace(serverType) == "" {
		return fmt.Errorf("server_type is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET server_type = ?, updated_at = ? WHERE id = ?`,
		serverType,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update server type: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

// UpdateImage updates the image field of a server.
func (s *ServerStore) UpdateImage(ctx context.Context, id int64, image string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("image is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET image = ?, updated_at = ? WHERE id = ?`,
		image,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update server image: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

func (s *ServerStore) QueueServerJob(ctx context.Context, in QueueServerJobInput) (StoredServer, orchestrator.Job, error) {
	if in.ServerID <= 0 {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server_id must be greater than zero")
	}
	if !orchestrator.IsKnownJobKind(in.Kind) {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("unsupported job kind: %s", in.Kind)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var serverStatusRaw string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM servers WHERE id = ?`, in.ServerID).Scan(&serverStatusRaw); err != nil {
		if err == sql.ErrNoRows {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server %d not found", in.ServerID)
		}
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("read server status: %w", err)
	}

	serverStatus, err := platform.NormalizeServerStatus(serverStatusRaw)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("normalize server status: %w", err)
	}
	if platform.IsDeletingOrDeletedServerStatus(string(serverStatus)) {
		if serverStatus == platform.ServerStatusDeleting {
			return StoredServer{}, orchestrator.Job{}, ErrServerDeleting
		}
		return StoredServer{}, orchestrator.Job{}, ErrServerDeleted
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if queuedStatus, ok := orchestrator.QueuedServerStatusForKind(in.Kind); ok {
		var activeJobID int64
		var activeJobKind string
		err := tx.QueryRowContext(ctx,
			`SELECT id, kind
				 FROM jobs
				 WHERE server_id = ?
				   AND kind IN (?, ?, ?)
				   AND status IN (?, ?)
				 ORDER BY created_at DESC
				 LIMIT 1`,
			in.ServerID,
			string(orchestrator.JobKindDeleteServer),
			string(orchestrator.JobKindRebuildServer),
			string(orchestrator.JobKindResizeServer),
			orchestrator.JobStatusQueued,
			orchestrator.JobStatusRunning,
		).Scan(&activeJobID, &activeJobKind)
		if err != nil && err != sql.ErrNoRows {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("check active destructive jobs: %w", err)
		}
		if err == nil {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("%w: job %d (%s)", ErrServerActionConflict, activeJobID, activeJobKind)
		}

		res, err := tx.ExecContext(ctx,
			`UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`,
			string(queuedStatus),
			now,
			in.ServerID,
		)
		if err != nil {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("update queued lifecycle status: %w", err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server %d not found", in.ServerID)
		}
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO jobs (server_id, kind, status, current_step, retry_count, payload, created_at, updated_at)
		 VALUES (?, ?, ?, '', 0, ?, ?, ?)`,
		in.ServerID,
		in.Kind,
		orchestrator.JobStatusQueued,
		nullableServerString(in.Payload),
		now,
		now,
	)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("insert job: %w", err)
	}

	jobID, err := res.LastInsertId()
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("job insert id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("commit transaction: %w", err)
	}

	server, err := s.GetByID(ctx, in.ServerID)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, err
	}
	job, err := orchestrator.NewStore(s.db).GetJob(ctx, jobID)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, err
	}

	return *server, job, nil
}

// UpdateNodeStatus updates the node_status, node_last_seen, and node_version fields.
func (s *ServerStore) UpdateNodeStatus(ctx context.Context, id int64, status platform.NodeStatus, lastSeen, version string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if _, err := platform.NormalizeNodeStatus(string(status)); err != nil {
		return err
	}
	lastSeen = strings.TrimSpace(lastSeen)
	if lastSeen != "" {
		if _, err := time.Parse(time.RFC3339, lastSeen); err != nil {
			return fmt.Errorf("invalid node_last_seen timestamp: %w", err)
		}
	}
	version = strings.TrimSpace(version)

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET node_status = ?, node_last_seen = ?, node_version = ?, updated_at = ? WHERE id = ?`,
		string(status),
		lastSeen,
		version,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %d not found", id)
	}

	return nil
}

func (s *ServerStore) MarkNodesOfflineBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	cutoffText := cutoff.UTC().Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers
		 SET node_status = ?, updated_at = ?
		 WHERE node_status IN (?, ?)
		   AND node_last_seen IS NOT NULL
		   AND node_last_seen != ''
		   AND node_last_seen < ?`,
		string(platform.NodeStatusOffline),
		now,
		string(platform.NodeStatusOnline),
		string(platform.NodeStatusUnhealthy),
		cutoffText,
	)
	if err != nil {
		return 0, fmt.Errorf("mark nodes offline: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("mark nodes offline rows affected: %w", err)
	}
	return rows, nil
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func nullableServerString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func normalizeStoredNodeStatus(value sql.NullString) (platform.NodeStatus, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return platform.NodeStatusUnknown, nil
	}
	return platform.NormalizeNodeStatus(value.String)
}
