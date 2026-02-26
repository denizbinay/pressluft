package activity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store persists activity entries.
type Store struct {
	db *sql.DB
}

// NewStore creates a new activity store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Emit creates a new activity entry.
func (s *Store) Emit(ctx context.Context, in EmitInput) (Activity, error) {
	// Validate event type
	if err := ValidateEventType(in.EventType); err != nil {
		return Activity{}, err
	}

	if strings.TrimSpace(in.Title) == "" {
		return Activity{}, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(string(in.Category)) == "" {
		return Activity{}, fmt.Errorf("category is required")
	}
	if strings.TrimSpace(string(in.Level)) == "" {
		return Activity{}, fmt.Errorf("level is required")
	}
	if strings.TrimSpace(string(in.ActorType)) == "" {
		return Activity{}, fmt.Errorf("actor_type is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	requiresAttention := 0
	if in.RequiresAttention {
		requiresAttention = 1
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO activity (
			event_type, category, level,
			resource_type, resource_id,
			parent_resource_type, parent_resource_id,
			actor_type, actor_id,
			title, message, payload,
			requires_attention, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.EventType,
		in.Category,
		in.Level,
		nullableString(string(in.ResourceType)),
		nullableInt64(in.ResourceID),
		nullableString(string(in.ParentResourceType)),
		nullableInt64(in.ParentResourceID),
		in.ActorType,
		nullableString(in.ActorID),
		in.Title,
		nullableString(in.Message),
		nullableString(in.Payload),
		requiresAttention,
		now,
	)
	if err != nil {
		return Activity{}, fmt.Errorf("insert activity: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Activity{}, fmt.Errorf("activity insert id: %w", err)
	}

	return s.GetByID(ctx, id)
}

// GetByID retrieves a single activity entry by ID.
func (s *Store) GetByID(ctx context.Context, id int64) (Activity, error) {
	if id <= 0 {
		return Activity{}, fmt.Errorf("id must be greater than zero")
	}

	var (
		a                  Activity
		resourceType       sql.NullString
		resourceID         sql.NullInt64
		parentResourceType sql.NullString
		parentResourceID   sql.NullInt64
		actorID            sql.NullString
		message            sql.NullString
		payload            sql.NullString
		readAt             sql.NullString
		requiresAttention  int
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, event_type, category, level,
			resource_type, resource_id,
			parent_resource_type, parent_resource_id,
			actor_type, actor_id,
			title, message, payload,
			requires_attention, read_at, created_at
		FROM activity
		WHERE id = ?`,
		id,
	).Scan(
		&a.ID,
		&a.EventType,
		&a.Category,
		&a.Level,
		&resourceType,
		&resourceID,
		&parentResourceType,
		&parentResourceID,
		&a.ActorType,
		&actorID,
		&a.Title,
		&message,
		&payload,
		&requiresAttention,
		&readAt,
		&a.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Activity{}, fmt.Errorf("activity %d not found", id)
		}
		return Activity{}, fmt.Errorf("query activity: %w", err)
	}

	if resourceType.Valid {
		a.ResourceType = ResourceType(resourceType.String)
	}
	if resourceID.Valid {
		a.ResourceID = resourceID.Int64
	}
	if parentResourceType.Valid {
		a.ParentResourceType = ResourceType(parentResourceType.String)
	}
	if parentResourceID.Valid {
		a.ParentResourceID = parentResourceID.Int64
	}
	if actorID.Valid {
		a.ActorID = actorID.String
	}
	if message.Valid {
		a.Message = message.String
	}
	if payload.Valid {
		a.Payload = payload.String
	}
	if readAt.Valid {
		a.ReadAt = readAt.String
	}
	a.RequiresAttention = requiresAttention == 1

	return a, nil
}

// List retrieves activity entries with cursor-based pagination and filtering.
// Returns entries, next cursor (empty string if no more), and error.
func (s *Store) List(ctx context.Context, filter ListFilter) ([]Activity, string, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Build dynamic query
	query := strings.Builder{}
	query.WriteString(`SELECT id, event_type, category, level,
		resource_type, resource_id,
		parent_resource_type, parent_resource_id,
		actor_type, actor_id,
		title, message, payload,
		requires_attention, read_at, created_at
	FROM activity
	WHERE 1=1`)

	args := make([]any, 0)

	// Cursor pagination (descending by id)
	if filter.Cursor > 0 {
		query.WriteString(" AND id < ?")
		args = append(args, filter.Cursor)
	}

	// Category filter
	if filter.Category != "" {
		query.WriteString(" AND category = ?")
		args = append(args, filter.Category)
	}

	// Resource filter
	if filter.ResourceType != "" {
		query.WriteString(" AND resource_type = ?")
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID > 0 {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if filter.ParentResourceID > 0 {
		query.WriteString(" AND parent_resource_id = ?")
		args = append(args, filter.ParentResourceID)
	}

	// Requires attention filter
	if filter.RequiresAttention != nil {
		if *filter.RequiresAttention {
			query.WriteString(" AND requires_attention = 1")
		} else {
			query.WriteString(" AND requires_attention = 0")
		}
	}

	// Unread only filter
	if filter.UnreadOnly {
		query.WriteString(" AND read_at IS NULL")
	}

	query.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit+1) // Fetch one extra to determine if there's a next page

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list activity: %w", err)
	}
	defer rows.Close()

	out := make([]Activity, 0, limit)
	for rows.Next() {
		var (
			a                  Activity
			resourceType       sql.NullString
			resourceID         sql.NullInt64
			parentResourceType sql.NullString
			parentResourceID   sql.NullInt64
			actorID            sql.NullString
			message            sql.NullString
			payload            sql.NullString
			readAt             sql.NullString
			requiresAttention  int
		)

		if err := rows.Scan(
			&a.ID,
			&a.EventType,
			&a.Category,
			&a.Level,
			&resourceType,
			&resourceID,
			&parentResourceType,
			&parentResourceID,
			&a.ActorType,
			&actorID,
			&a.Title,
			&message,
			&payload,
			&requiresAttention,
			&readAt,
			&a.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan activity: %w", err)
		}

		if resourceType.Valid {
			a.ResourceType = ResourceType(resourceType.String)
		}
		if resourceID.Valid {
			a.ResourceID = resourceID.Int64
		}
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		if parentResourceID.Valid {
			a.ParentResourceID = parentResourceID.Int64
		}
		if actorID.Valid {
			a.ActorID = actorID.String
		}
		if message.Valid {
			a.Message = message.String
		}
		if payload.Valid {
			a.Payload = payload.String
		}
		if readAt.Valid {
			a.ReadAt = readAt.String
		}
		a.RequiresAttention = requiresAttention == 1

		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate activity: %w", err)
	}

	// Determine next cursor
	nextCursor := ""
	if len(out) > limit {
		// We have more results, return cursor for next page
		out = out[:limit]
		nextCursor = fmt.Sprintf("%d", out[len(out)-1].ID)
	}

	return out, nextCursor, nil
}

// ListForServer retrieves activity entries for a server, including server events
// and related child events (e.g., jobs attached to the server).
func (s *Store) ListForServer(ctx context.Context, serverID int64, filter ListFilter) ([]Activity, string, error) {
	if serverID <= 0 {
		return nil, "", fmt.Errorf("server id must be greater than zero")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := strings.Builder{}
	query.WriteString(`SELECT id, event_type, category, level,
		resource_type, resource_id,
		parent_resource_type, parent_resource_id,
		actor_type, actor_id,
		title, message, payload,
		requires_attention, read_at, created_at
	FROM activity
	WHERE ((resource_type = ? AND resource_id = ?) OR (parent_resource_type = ? AND parent_resource_id = ?))`)

	args := []any{ResourceServer, serverID, ResourceServer, serverID}

	if filter.Cursor > 0 {
		query.WriteString(" AND id < ?")
		args = append(args, filter.Cursor)
	}

	if filter.Category != "" {
		query.WriteString(" AND category = ?")
		args = append(args, filter.Category)
	}

	if filter.RequiresAttention != nil {
		if *filter.RequiresAttention {
			query.WriteString(" AND requires_attention = 1")
		} else {
			query.WriteString(" AND requires_attention = 0")
		}
	}

	if filter.UnreadOnly {
		query.WriteString(" AND read_at IS NULL")
	}

	query.WriteString(" ORDER BY id DESC LIMIT ?")
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list activity for server: %w", err)
	}
	defer rows.Close()

	out := make([]Activity, 0, limit)
	for rows.Next() {
		var (
			a                  Activity
			resourceType       sql.NullString
			resourceID         sql.NullInt64
			parentResourceType sql.NullString
			parentResourceID   sql.NullInt64
			actorID            sql.NullString
			message            sql.NullString
			payload            sql.NullString
			readAt             sql.NullString
			requiresAttention  int
		)

		if err := rows.Scan(
			&a.ID,
			&a.EventType,
			&a.Category,
			&a.Level,
			&resourceType,
			&resourceID,
			&parentResourceType,
			&parentResourceID,
			&a.ActorType,
			&actorID,
			&a.Title,
			&message,
			&payload,
			&requiresAttention,
			&readAt,
			&a.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan activity: %w", err)
		}

		if resourceType.Valid {
			a.ResourceType = ResourceType(resourceType.String)
		}
		if resourceID.Valid {
			a.ResourceID = resourceID.Int64
		}
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		if parentResourceID.Valid {
			a.ParentResourceID = parentResourceID.Int64
		}
		if actorID.Valid {
			a.ActorID = actorID.String
		}
		if message.Valid {
			a.Message = message.String
		}
		if payload.Valid {
			a.Payload = payload.String
		}
		if readAt.Valid {
			a.ReadAt = readAt.String
		}
		a.RequiresAttention = requiresAttention == 1

		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate activity: %w", err)
	}

	nextCursor := ""
	if len(out) > limit {
		out = out[:limit]
		nextCursor = fmt.Sprintf("%d", out[len(out)-1].ID)
	}

	return out, nextCursor, nil
}

// MarkRead marks a single activity entry as read.
func (s *Store) MarkRead(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id must be greater than zero")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE activity SET read_at = ? WHERE id = ? AND read_at IS NULL`,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		// Check if the activity exists
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM activity WHERE id = ?`, id).Scan(&exists)
		if err == sql.ErrNoRows {
			return fmt.Errorf("activity %d not found", id)
		}
		// Entry exists but was already read - that's fine
	}

	return nil
}

// MarkAllRead marks all matching activity entries as read.
func (s *Store) MarkAllRead(ctx context.Context, filter ListFilter) error {
	query := strings.Builder{}
	query.WriteString(`UPDATE activity SET read_at = ? WHERE read_at IS NULL`)

	now := time.Now().UTC().Format(time.RFC3339)
	args := []any{now}

	// Category filter
	if filter.Category != "" {
		query.WriteString(" AND category = ?")
		args = append(args, filter.Category)
	}

	// Resource filter
	if filter.ResourceType != "" {
		query.WriteString(" AND resource_type = ?")
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID > 0 {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if filter.ParentResourceID > 0 {
		query.WriteString(" AND parent_resource_id = ?")
		args = append(args, filter.ParentResourceID)
	}

	// Requires attention filter
	if filter.RequiresAttention != nil && *filter.RequiresAttention {
		query.WriteString(" AND requires_attention = 1")
	}

	_, err := s.db.ExecContext(ctx, query.String(), args...)
	if err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}

	return nil
}

// CountUnread counts unread activity entries matching the filter.
func (s *Store) CountUnread(ctx context.Context, filter ListFilter) (int64, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT COUNT(*) FROM activity WHERE read_at IS NULL`)

	args := make([]any, 0)

	// Category filter
	if filter.Category != "" {
		query.WriteString(" AND category = ?")
		args = append(args, filter.Category)
	}

	// Resource filter
	if filter.ResourceType != "" {
		query.WriteString(" AND resource_type = ?")
		args = append(args, filter.ResourceType)
	}
	if filter.ResourceID > 0 {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if filter.ParentResourceID > 0 {
		query.WriteString(" AND parent_resource_id = ?")
		args = append(args, filter.ParentResourceID)
	}

	// Requires attention filter (for unread count, usually we want attention items)
	if filter.RequiresAttention != nil && *filter.RequiresAttention {
		query.WriteString(" AND requires_attention = 1")
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query.String(), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}

	return count, nil
}

// GetLatestID returns the ID of the most recent activity entry.
// Returns 0 if no entries exist (not an error).
func (s *Store) GetLatestID(ctx context.Context) (int64, error) {
	var id sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(id) FROM activity`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get latest id: %w", err)
	}
	if !id.Valid {
		return 0, nil
	}
	return id.Int64, nil
}

// ListSince returns activity entries with ID greater than sinceID.
// Used for SSE streaming to poll for new entries.
func (s *Store) ListSince(ctx context.Context, sinceID int64, limit int) ([]Activity, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, event_type, category, level,
			resource_type, resource_id,
			parent_resource_type, parent_resource_id,
			actor_type, actor_id,
			title, message, payload,
			requires_attention, read_at, created_at
		FROM activity
		WHERE id > ?
		ORDER BY id ASC
		LIMIT ?`,
		sinceID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list since: %w", err)
	}
	defer rows.Close()

	out := make([]Activity, 0, limit)
	for rows.Next() {
		var (
			a                  Activity
			resourceType       sql.NullString
			resourceID         sql.NullInt64
			parentResourceType sql.NullString
			parentResourceID   sql.NullInt64
			actorID            sql.NullString
			message            sql.NullString
			payload            sql.NullString
			readAt             sql.NullString
			requiresAttention  int
		)

		if err := rows.Scan(
			&a.ID,
			&a.EventType,
			&a.Category,
			&a.Level,
			&resourceType,
			&resourceID,
			&parentResourceType,
			&parentResourceID,
			&a.ActorType,
			&actorID,
			&a.Title,
			&message,
			&payload,
			&requiresAttention,
			&readAt,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}

		if resourceType.Valid {
			a.ResourceType = ResourceType(resourceType.String)
		}
		if resourceID.Valid {
			a.ResourceID = resourceID.Int64
		}
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		if parentResourceID.Valid {
			a.ParentResourceID = parentResourceID.Int64
		}
		if actorID.Valid {
			a.ActorID = actorID.String
		}
		if message.Valid {
			a.Message = message.String
		}
		if payload.Valid {
			a.Payload = payload.String
		}
		if readAt.Valid {
			a.ReadAt = readAt.String
		}
		a.RequiresAttention = requiresAttention == 1

		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity: %w", err)
	}

	return out, nil
}

func nullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}
