package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StoredServer is a persisted server node record.
type StoredServer struct {
	ID               int64  `json:"id"`
	ProviderID       int64  `json:"provider_id"`
	ProviderType     string `json:"provider_type"`
	ProviderServerID string `json:"provider_server_id,omitempty"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	ServerType       string `json:"server_type"`
	Image            string `json:"image"`
	ProfileKey       string `json:"profile_key"`
	Status           string `json:"status"`
	ActionID         string `json:"action_id,omitempty"`
	ActionStatus     string `json:"action_status,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
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
	Status       string
}

// ServerStore provides persistence for server records.
type ServerStore struct {
	db *sql.DB
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
		`SELECT id, provider_id, provider_type, provider_server_id, name, location, server_type, image, profile_key, status, action_id, action_status, created_at, updated_at
		 FROM servers
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}
	defer rows.Close()

	var out []StoredServer
	for rows.Next() {
		var (
			s                StoredServer
			providerServerID sql.NullString
			actionID         sql.NullString
			actionStatus     sql.NullString
		)
		if err := rows.Scan(
			&s.ID,
			&s.ProviderID,
			&s.ProviderType,
			&providerServerID,
			&s.Name,
			&s.Location,
			&s.ServerType,
			&s.Image,
			&s.ProfileKey,
			&s.Status,
			&actionID,
			&actionStatus,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan server: %w", err)
		}
		s.ProviderServerID = nullStringValue(providerServerID)
		s.ActionID = nullStringValue(actionID)
		s.ActionStatus = nullStringValue(actionStatus)
		out = append(out, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate servers: %w", err)
	}

	return out, nil
}

func (s *ServerStore) UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus, status string) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("status is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE servers
		 SET provider_server_id = ?, action_id = ?, action_status = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		providerServerID,
		actionID,
		actionStatus,
		status,
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
	if strings.TrimSpace(in.Status) == "" {
		return fmt.Errorf("status is required")
	}
	return nil
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
