package orchestrator

import "fmt"

var allowedTransitions = map[JobStatus]map[JobStatus]struct{}{
	JobStatusQueued: {
		JobStatusPreparing: {},
		JobStatusCancelled: {},
		JobStatusFailed:    {},
		JobStatusTimedOut:  {},
	},
	JobStatusPreparing: {
		JobStatusRunning:   {},
		JobStatusFailed:    {},
		JobStatusCancelled: {},
		JobStatusTimedOut:  {},
	},
	JobStatusRunning: {
		JobStatusWaitingReboot: {},
		JobStatusVerifying:     {},
		JobStatusRetrying:      {},
		JobStatusFailed:        {},
		JobStatusCancelled:     {},
		JobStatusTimedOut:      {},
	},
	JobStatusWaitingReboot: {
		JobStatusResuming:  {},
		JobStatusFailed:    {},
		JobStatusCancelled: {},
		JobStatusTimedOut:  {},
	},
	JobStatusResuming: {
		JobStatusRunning:   {},
		JobStatusVerifying: {},
		JobStatusRetrying:  {},
		JobStatusFailed:    {},
		JobStatusCancelled: {},
		JobStatusTimedOut:  {},
	},
	JobStatusVerifying: {
		JobStatusSucceeded: {},
		JobStatusRetrying:  {},
		JobStatusFailed:    {},
		JobStatusCancelled: {},
		JobStatusTimedOut:  {},
	},
	JobStatusRetrying: {
		JobStatusRunning:   {},
		JobStatusFailed:    {},
		JobStatusCancelled: {},
		JobStatusTimedOut:  {},
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
		return fmt.Errorf("status is already %q", from)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid job status transition: %q -> %q", from, to)
	}
	return nil
}

// IsTerminalStatus reports whether the status can no longer transition.
func IsTerminalStatus(status JobStatus) bool {
	switch status {
	case JobStatusSucceeded, JobStatusFailed, JobStatusCancelled, JobStatusTimedOut:
		return true
	default:
		return false
	}
}
