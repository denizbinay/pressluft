package observability

import "encoding/json"

type Correlation struct {
	JobID     string `json:"job_id,omitempty"`
	ServerID  string `json:"server_id,omitempty"`
	CommandID string `json:"command_id,omitempty"`
}

func (c Correlation) LogArgs(args ...any) []any {
	out := make([]any, 0, len(args)+6)
	if c.JobID != "" {
		out = append(out, "job_id", c.JobID)
	}
	if c.ServerID != "" {
		out = append(out, "server_id", c.ServerID)
	}
	if c.CommandID != "" {
		out = append(out, "command_id", c.CommandID)
	}
	return append(out, args...)
}

func (c Correlation) Payload(fields map[string]any) string {
	payload := make(map[string]any, len(fields)+3)
	if c.JobID != "" {
		payload["job_id"] = c.JobID
	}
	if c.ServerID != "" {
		payload["server_id"] = c.ServerID
	}
	if c.CommandID != "" {
		payload["command_id"] = c.CommandID
	}
	for key, value := range fields {
		if value == nil {
			continue
		}
		payload[key] = value
	}
	if len(payload) == 0 {
		return ""
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(encoded)
}
