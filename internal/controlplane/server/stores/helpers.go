package stores

import (
	"database/sql"
	"strings"

	"pressluft/internal/platform"
)

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func nullableString(v string) any {
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

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
