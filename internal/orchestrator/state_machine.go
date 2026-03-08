package orchestrator

import "fmt"

var allowedTransitions = map[JobStatus]map[JobStatus]struct{}{
	JobStatusQueued: {
		JobStatusRunning: {},
		JobStatusFailed:  {},
	},
	JobStatusRunning: {
		JobStatusSucceeded: {},
		JobStatusFailed:    {},
	},
}

// CanTransition reports whether the requested lifecycle change is allowed.
func CanTransition(from, to JobStatus) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

// ValidateTransition returns an error when transition is invalid.
func ValidateTransition(from, to JobStatus) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid job status transition: %q -> %q", from, to)
	}
	return nil
}

// IsTerminalStatus reports whether the status can no longer transition.
func IsTerminalStatus(status JobStatus) bool {
	switch status {
	case JobStatusSucceeded, JobStatusFailed:
		return true
	default:
		return false
	}
}
