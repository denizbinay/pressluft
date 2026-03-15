package stores

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/platform"
	"pressluft/internal/shared/idutil"
)

var (
	ErrServerActionConflict = errors.New("server action already in progress")
	ErrServerDeleting       = errors.New("server is deleting")
	ErrServerDeleted        = errors.New("server is deleted")
)

// StoredServer is a persisted server node record.
type StoredServer struct {
	ID               string                `json:"id"`
	ProviderID       string                `json:"provider_id"`
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
	ProviderID   string
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
	ServerID string
	Kind     string
	Payload  string
}

func NewServerStore(db *sql.DB) *ServerStore {
	return &ServerStore{db: db}
}

func (s *ServerStore) Create(ctx context.Context, in CreateServerNodeInput) (string, error) {
	if err := validateCreateServerNodeInput(in); err != nil {
		return "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	publicID, err := idutil.New()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO servers (
			id, provider_id, provider_type, name, location, server_type, image, profile_key, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		publicID,
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
		return "", fmt.Errorf("insert server: %w", err)
	}
	return publicID, nil
}

func (s *ServerStore) List(ctx context.Context) ([]StoredServer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT s.id, p.id, s.provider_type, s.provider_server_id, s.ipv4, s.ipv6, s.name, s.location, s.server_type, s.image, s.profile_key, s.status, s.setup_state, s.setup_last_error, s.action_id, s.action_status, s.node_status, s.node_last_seen, s.node_version, s.created_at, s.updated_at,
		 CASE WHEN k.server_id IS NULL THEN 0 ELSE 1 END AS has_key
		 FROM servers s
		 JOIN providers p ON p.id = s.provider_id
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

func (s *ServerStore) UpdateProvisioning(ctx context.Context, id string, providerServerID, actionID, actionStatus string, status platform.ServerStatus, ipv4, ipv6 string) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
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
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update server provisioning: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}

	return nil
}

func validateCreateServerNodeInput(in CreateServerNodeInput) error {
	if _, err := idutil.Normalize(in.ProviderID); err != nil {
		return fmt.Errorf("provider_id: %w", err)
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
func (s *ServerStore) GetByID(ctx context.Context, id string) (*StoredServer, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
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
	err = s.db.QueryRowContext(ctx,
		`SELECT s.id, p.id, s.provider_type, s.provider_server_id, s.ipv4, s.ipv6, s.name, s.location, s.server_type, s.image, s.profile_key, s.status, s.setup_state, s.setup_last_error, s.action_id, s.action_status, s.node_status, s.node_last_seen, s.node_version, s.created_at, s.updated_at,
		 CASE WHEN k.server_id IS NULL THEN 0 ELSE 1 END AS has_key
		 FROM servers s
		 JOIN providers p ON p.id = s.provider_id
		 LEFT JOIN server_keys k ON k.server_id = s.id
		 WHERE s.id = ?`,
		publicID,
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
			return nil, fmt.Errorf("server %s not found", publicID)
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
func (s *ServerStore) UpdateStatus(ctx context.Context, id string, status platform.ServerStatus) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
	}
	if _, err := platform.NormalizeServerStatus(string(status)); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`,
		string(status),
		now,
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update server status: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}

	return nil
}

func (s *ServerStore) UpdateSetupState(ctx context.Context, id string, setupState platform.SetupState, setupLastError string) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
	}
	if _, err := platform.NormalizeSetupState(string(setupState)); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET setup_state = ?, setup_last_error = ?, updated_at = ? WHERE id = ?`,
		string(setupState),
		nullableString(setupLastError),
		now,
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update server setup state: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}

	return nil
}

// UpdateServerType updates the server_type field of a server.
func (s *ServerStore) UpdateServerType(ctx context.Context, id string, serverType string) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(serverType) == "" {
		return fmt.Errorf("server_type is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET server_type = ?, updated_at = ? WHERE id = ?`,
		serverType,
		now,
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update server type: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}

	return nil
}

// UpdateImage updates the image field of a server.
func (s *ServerStore) UpdateImage(ctx context.Context, id string, image string) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("image is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers SET image = ?, updated_at = ? WHERE id = ?`,
		image,
		now,
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update server image: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", serverID)
	}

	return nil
}

// UpdateNodeStatus updates the node_status, node_last_seen, and node_version fields.
func (s *ServerStore) UpdateNodeStatus(ctx context.Context, id string, status platform.NodeStatus, lastSeen, version string) error {
	serverID, err := s.lookupServerID(ctx, id)
	if err != nil {
		return err
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
		serverID,
	)
	if err != nil {
		return fmt.Errorf("update node status: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("server %s not found", id)
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

func (s *ServerStore) lookupServerID(ctx context.Context, id string) (string, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return "", err
	}
	var serverID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM servers WHERE id = ?`, publicID).Scan(&serverID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("server %s not found", publicID)
		}
		return "", fmt.Errorf("lookup server id: %w", err)
	}
	return serverID, nil
}
