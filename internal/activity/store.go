package activity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/idutil"
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

	publicID, err := idutil.New()
	if err != nil {
		return Activity{}, err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO activity (
			id,
			event_type, category, level,
			resource_type, resource_id,
			parent_resource_type, parent_resource_id,
			actor_type, actor_id,
			title, message, payload,
			requires_attention, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		publicID,
		in.EventType,
		in.Category,
		in.Level,
		nullableString(string(in.ResourceType)),
		nullableString(in.ResourceID),
		nullableString(string(in.ParentResourceType)),
		nullableString(in.ParentResourceID),
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

	return s.GetByID(ctx, publicID)
}

// GetByID retrieves a single activity entry by ID.
func (s *Store) GetByID(ctx context.Context, id string) (Activity, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return Activity{}, err
	}

	var (
		a                  Activity
		resourceType       sql.NullString
		resourceID         sql.NullString
		parentResourceType sql.NullString
		parentResourceID   sql.NullString
		actorID            sql.NullString
		message            sql.NullString
		payload            sql.NullString
		readAt             sql.NullString
		requiresAttention  int
	)

	err = s.db.QueryRowContext(ctx,
		`SELECT id, event_type, category, level,
			resource_type, resource_id,
			parent_resource_type, parent_resource_id,
			actor_type, actor_id,
			title, message, payload,
			requires_attention, read_at, created_at
		FROM activity
		WHERE id = ?`,
		publicID,
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
			return Activity{}, fmt.Errorf("activity %s not found", publicID)
		}
		return Activity{}, fmt.Errorf("query activity: %w", err)
	}

	if resourceType.Valid {
		a.ResourceType = ResourceType(resourceType.String)
	}
	a.ResourceID = nullString(resourceID)
	if parentResourceType.Valid {
		a.ParentResourceType = ResourceType(parentResourceType.String)
	}
	a.ParentResourceID = nullString(parentResourceID)
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
	if cursor := parseCursor(filter.Cursor); cursor != "" {
		query.WriteString(" AND id < ?")
		args = append(args, cursor)
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
	if strings.TrimSpace(filter.ResourceID) != "" {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if strings.TrimSpace(filter.ParentResourceID) != "" {
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
			resourceID         sql.NullString
			parentResourceType sql.NullString
			parentResourceID   sql.NullString
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
		a.ResourceID = nullString(resourceID)
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		a.ParentResourceID = nullString(parentResourceID)
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
		nextCursor = out[len(out)-1].ID
	}

	return out, nextCursor, nil
}

// ListForServer retrieves activity entries for a server, including server events
// and related child events (e.g., jobs attached to the server).
func (s *Store) ListForServer(ctx context.Context, serverID string, filter ListFilter) ([]Activity, string, error) {
	if strings.TrimSpace(serverID) == "" {
		return nil, "", fmt.Errorf("server id is required")
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

	if cursor := parseCursor(filter.Cursor); cursor != "" {
		query.WriteString(" AND id < ?")
		args = append(args, cursor)
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
			resourceID         sql.NullString
			parentResourceType sql.NullString
			parentResourceID   sql.NullString
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
		a.ResourceID = nullString(resourceID)
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		a.ParentResourceID = nullString(parentResourceID)
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
		nextCursor = out[len(out)-1].ID
	}

	return out, nextCursor, nil
}

// ListForSite retrieves activity entries for a site, including site events
// and related child events such as domain assignment changes.
func (s *Store) ListForSite(ctx context.Context, siteID string, filter ListFilter) ([]Activity, string, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, "", fmt.Errorf("site id is required")
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

	args := []any{ResourceSite, siteID, ResourceSite, siteID}

	if cursor := parseCursor(filter.Cursor); cursor != "" {
		query.WriteString(" AND id < ?")
		args = append(args, cursor)
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
		return nil, "", fmt.Errorf("list activity for site: %w", err)
	}
	defer rows.Close()

	out := make([]Activity, 0, limit)
	for rows.Next() {
		var (
			a                  Activity
			resourceType       sql.NullString
			resourceID         sql.NullString
			parentResourceType sql.NullString
			parentResourceID   sql.NullString
			actorID            sql.NullString
			message            sql.NullString
			payload            sql.NullString
			readAt             sql.NullString
			requiresAttention  int
		)
		if err := rows.Scan(&a.ID, &a.EventType, &a.Category, &a.Level, &resourceType, &resourceID, &parentResourceType, &parentResourceID, &a.ActorType, &actorID, &a.Title, &message, &payload, &requiresAttention, &readAt, &a.CreatedAt); err != nil {
			return nil, "", fmt.Errorf("scan activity: %w", err)
		}
		if resourceType.Valid {
			a.ResourceType = ResourceType(resourceType.String)
		}
		a.ResourceID = nullString(resourceID)
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		a.ParentResourceID = nullString(parentResourceID)
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
		nextCursor = out[len(out)-1].ID
	}
	return out, nextCursor, nil
}

// MarkRead marks a single activity entry as read.
func (s *Store) MarkRead(ctx context.Context, id string) error {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`UPDATE activity SET read_at = ? WHERE id = ? AND read_at IS NULL`,
		now,
		publicID,
	)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		// Check if the activity exists
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM activity WHERE id = ?`, publicID).Scan(&exists)
		if err == sql.ErrNoRows {
			return fmt.Errorf("activity %s not found", publicID)
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
	if strings.TrimSpace(filter.ResourceID) != "" {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if strings.TrimSpace(filter.ParentResourceID) != "" {
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
	if strings.TrimSpace(filter.ResourceID) != "" {
		query.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	// Parent resource filter
	if filter.ParentResourceType != "" {
		query.WriteString(" AND parent_resource_type = ?")
		args = append(args, filter.ParentResourceType)
	}
	if strings.TrimSpace(filter.ParentResourceID) != "" {
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
// Returns empty string if no entries exist (not an error).
func (s *Store) GetLatestID(ctx context.Context) (string, error) {
	var id sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id FROM activity ORDER BY id DESC LIMIT 1`).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("get latest id: %w", err)
	}
	if !id.Valid {
		return "", nil
	}
	return id.String, nil
}

// ListSince returns activity entries with ID greater than sinceID.
// Used for SSE streaming to poll for new entries.
func (s *Store) ListSince(ctx context.Context, sinceID string, limit int) ([]Activity, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	if strings.TrimSpace(sinceID) == "" {
		sinceID = "00000000-0000-7000-8000-000000000000"
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
			resourceID         sql.NullString
			parentResourceType sql.NullString
			parentResourceID   sql.NullString
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
		a.ResourceID = nullString(resourceID)
		if parentResourceType.Valid {
			a.ParentResourceType = ResourceType(parentResourceType.String)
		}
		a.ParentResourceID = nullString(parentResourceID)
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

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func nullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func parseCursor(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	normalized, err := idutil.Normalize(v)
	if err != nil {
		return ""
	}
	return normalized
}
