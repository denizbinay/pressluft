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
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
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
