package orchestrator

// JobStatus is the durable lifecycle state for orchestration jobs.
type JobStatus string

const (
	JobStatusQueued        JobStatus = "queued"
	JobStatusPreparing     JobStatus = "preparing"
	JobStatusRunning       JobStatus = "running"
	JobStatusWaitingReboot JobStatus = "waiting_reboot"
	JobStatusResuming      JobStatus = "resuming"
	JobStatusVerifying     JobStatus = "verifying"
	JobStatusRetrying      JobStatus = "retrying"
	JobStatusSucceeded     JobStatus = "succeeded"
	JobStatusFailed        JobStatus = "failed"
	JobStatusCancelled     JobStatus = "cancelled"
	JobStatusTimedOut      JobStatus = "timed_out"
)

// Job is the persisted orchestration unit.
type Job struct {
	ID          int64     `json:"id"`
	ServerID    int64     `json:"server_id,omitempty"`
	Kind        string    `json:"kind"`
	Status      JobStatus `json:"status"`
	CurrentStep string    `json:"current_step"`
	RetryCount  int       `json:"retry_count"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// JobEvent is an ordered event entry consumed by the dashboard.
type JobEvent struct {
	JobID      int64  `json:"job_id"`
	Seq        int64  `json:"seq"`
	EventType  string `json:"event_type"`
	Level      string `json:"level"`
	StepKey    string `json:"step_key,omitempty"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message"`
	Payload    string `json:"payload,omitempty"`
	OccurredAt string `json:"occurred_at"`
}

// CreateJobInput is the job creation payload.
type CreateJobInput struct {
	Kind     string
	ServerID int64
}

// TransitionInput updates a job lifecycle state.
type TransitionInput struct {
	ToStatus    JobStatus
	CurrentStep string
	LastError   string
	RetryCount  int
}

// CreateEventInput appends an event for a job timeline.
type CreateEventInput struct {
	EventType string
	Level     string
	StepKey   string
	Status    string
	Message   string
	Payload   string
}
