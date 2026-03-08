package ws

import (
	"encoding/json"
	"time"

	"pressluft/internal/platform"
)

type MessageType string

const (
	TypeHeartbeat     MessageType = "heartbeat"
	TypeHeartbeatAck  MessageType = "heartbeat_ack"
	TypeCommand       MessageType = "command"
	TypeCommandResult MessageType = "command_result"
	TypeLogEntry      MessageType = "log_entry"
)

type Envelope struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Heartbeat struct {
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"version"`
	CPUPercent float64   `json:"cpu_percent"`
	MemUsedMB  int64     `json:"mem_used_mb"`
	MemTotalMB int64     `json:"mem_total_mb"`
}

// AgentInfo holds real-time agent state including connection and metrics.
type AgentInfo struct {
	Connected  bool                `json:"connected"`
	Status     platform.NodeStatus `json:"status"`
	LastSeen   time.Time           `json:"last_seen,omitempty"`
	Version    string              `json:"version,omitempty"`
	CPUPercent float64             `json:"cpu_percent,omitempty"`
	MemUsedMB  int64               `json:"mem_used_mb,omitempty"`
	MemTotalMB int64               `json:"mem_total_mb,omitempty"`
}

type Command struct {
	ID       string          `json:"id"`
	JobID    int64           `json:"job_id"`
	ServerID int64           `json:"server_id,omitempty"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

type CommandResult struct {
	CommandID string          `json:"command_id"`
	JobID     int64           `json:"job_id,omitempty"`
	ServerID  int64           `json:"server_id,omitempty"`
	Success   bool            `json:"success"`
	Error     string          `json:"error,omitempty"`
	ErrorCode string          `json:"error_code,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Output    string          `json:"output,omitempty"`
}

type LogEntry struct {
	CommandID string    `json:"command_id"`
	JobID     int64     `json:"job_id,omitempty"`
	ServerID  int64     `json:"server_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

func SuccessResult(commandID string, payload any, output string) CommandResult {
	result := CommandResult{
		CommandID: commandID,
		Success:   true,
		Output:    output,
	}
	data, err := marshalJSON(payload)
	if err == nil && len(data) > 0 && string(data) != "null" {
		result.Payload = data
	}
	if err != nil {
		return FailureResult(commandID, "serialization_failed", "failed to encode command payload", nil, output)
	}
	return result
}

func FailureResult(commandID, code, message string, payload any, output string) CommandResult {
	result := CommandResult{
		CommandID: commandID,
		Success:   false,
		Error:     message,
		ErrorCode: code,
		Output:    output,
	}
	data, err := marshalJSON(payload)
	if err == nil && len(data) > 0 && string(data) != "null" {
		result.Payload = data
	}
	if err != nil {
		result.Payload = nil
		result.ErrorCode = "serialization_failed"
		if result.Error == "" {
			result.Error = "failed to encode command payload"
		}
	}
	return result
}
