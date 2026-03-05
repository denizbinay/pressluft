package ws

import (
	"encoding/json"
	"time"
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

// AgentStatus represents the connection status of an agent.
type AgentStatus string

const (
	AgentStatusOnline    AgentStatus = "online"
	AgentStatusUnhealthy AgentStatus = "unhealthy"
	AgentStatusOffline   AgentStatus = "offline"
	AgentStatusUnknown   AgentStatus = "unknown"
)

// AgentInfo holds real-time agent state including connection and metrics.
type AgentInfo struct {
	Connected  bool        `json:"connected"`
	Status     AgentStatus `json:"status"`
	LastSeen   time.Time   `json:"last_seen,omitempty"`
	Version    string      `json:"version,omitempty"`
	CPUPercent float64     `json:"cpu_percent,omitempty"`
	MemUsedMB  int64       `json:"mem_used_mb,omitempty"`
	MemTotalMB int64       `json:"mem_total_mb,omitempty"`
}

type Command struct {
	ID      string          `json:"id"`
	JobID   int64           `json:"job_id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type CommandResult struct {
	CommandID string `json:"command_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Output    string `json:"output,omitempty"`
}

type LogEntry struct {
	CommandID string    `json:"command_id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}
