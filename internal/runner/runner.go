package runner

import "context"

// Request defines the normalized runner execution input.
type Request struct {
	JobID         int64
	InventoryPath string
	PlaybookPath  string
	ExtraVars     map[string]string
	CheckOnly     bool
}

// Event describes a runner lifecycle event.
type Event struct {
	Type    string
	Level   string
	StepKey string
	Message string
	Payload string
}

// EventSink receives runner events and forwards them to orchestration storage.
type EventSink interface {
	Emit(ctx context.Context, event Event) error
}

// Runner executes a convergence task with a backend-specific adapter.
type Runner interface {
	Name() string
	Run(ctx context.Context, req Request, sink EventSink) error
}
