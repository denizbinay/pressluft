package agentproto

// Heartbeat is emitted periodically by a target-side execution agent.
type Heartbeat struct {
	JobID       int64  `json:"job_id"`
	ServerID    int64  `json:"server_id"`
	Status      string `json:"status"`
	CurrentStep string `json:"current_step"`
	Timestamp   string `json:"timestamp"`
}

// Checkpoint describes a resumable execution boundary.
type Checkpoint struct {
	JobID         int64  `json:"job_id"`
	StepKey       string `json:"step_key"`
	CheckpointKey string `json:"checkpoint_key"`
	ResumeToken   string `json:"resume_token,omitempty"`
	Payload       string `json:"payload,omitempty"`
	Timestamp     string `json:"timestamp"`
}
