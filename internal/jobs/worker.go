package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/audit"
)

const (
	defaultBackoff1 = time.Minute
	defaultBackoff2 = 5 * time.Minute
	defaultBackoff3 = 15 * time.Minute
)

type Handler func(ctx context.Context, job Job) error

type Worker struct {
	store       QueueStore
	handlers    map[string]Handler
	workerID    string
	auditor     audit.Recorder
	now         func() time.Time
	backoffFunc func(attempt int) time.Duration
}

func NewWorker(store QueueStore, workerID string, handlers map[string]Handler, auditRecorder audit.Recorder) *Worker {
	copyHandlers := make(map[string]Handler, len(handlers))
	for jobType, handler := range handlers {
		copyHandlers[jobType] = handler
	}

	return &Worker{
		store:       store,
		handlers:    copyHandlers,
		workerID:    workerID,
		auditor:     auditRecorder,
		now:         func() time.Time { return time.Now().UTC() },
		backoffFunc: retryBackoff,
	}
}

func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	now := w.now()
	job, ok, err := w.store.ClaimNextRunnable(ctx, w.workerID, now)
	if err != nil {
		return false, fmt.Errorf("claim next runnable job: %w", err)
	}
	if !ok {
		return false, nil
	}

	if job.AttemptCount == 1 {
		if err := w.recordAsyncAccepted(ctx, job); err != nil {
			if _, completeErr := w.store.CompleteFailure(ctx, job.ID, "AUDIT_WRITE_FAILED", "audit write failed", w.now()); completeErr != nil {
				return true, fmt.Errorf("record accepted audit and fail job: %w", completeErr)
			}
			return true, nil
		}
	}

	handler, found := w.handlers[job.JobType]
	if !found {
		handler = nil
	}

	if err := w.processJob(ctx, job, handler); err != nil {
		return true, err
	}

	return true, nil
}

func (w *Worker) processJob(ctx context.Context, job Job, handler Handler) error {
	if handler == nil {
		_, err := w.store.CompleteFailure(ctx, job.ID, "ANSIBLE_UNKNOWN_EXIT", truncateError("handler missing for job type: "+job.JobType), w.now())
		if err != nil {
			return fmt.Errorf("mark missing-handler failure: %w", err)
		}
		if err := w.updateAsyncResult(ctx, job, "failed"); err != nil {
			return fmt.Errorf("update async audit failure result: %w", err)
		}
		return nil
	}

	err := handler(ctx, job)
	if err == nil {
		if _, completeErr := w.store.CompleteSuccess(ctx, job.ID, w.now()); completeErr != nil {
			return fmt.Errorf("complete success: %w", completeErr)
		}
		if auditErr := w.updateAsyncResult(ctx, job, "succeeded"); auditErr != nil {
			return fmt.Errorf("update async audit success result: %w", auditErr)
		}
		return nil
	}

	jobErr := classifyError(err)
	message := truncateError(jobErr.Message)
	now := w.now()

	if jobErr.Retryable && job.AttemptCount < job.MaxAttempts {
		runAfter := now.Add(w.backoffFunc(job.AttemptCount))
		if _, requeueErr := w.store.Requeue(ctx, job.ID, runAfter, jobErr.Code, message, now); requeueErr != nil {
			return fmt.Errorf("requeue job: %w", requeueErr)
		}
		if auditErr := w.updateAsyncResult(ctx, job, "retrying"); auditErr != nil {
			return fmt.Errorf("update async audit retry result: %w", auditErr)
		}
		return nil
	}

	if _, completeErr := w.store.CompleteFailure(ctx, job.ID, jobErr.Code, message, now); completeErr != nil {
		return fmt.Errorf("complete failure: %w", completeErr)
	}
	if auditErr := w.updateAsyncResult(ctx, job, "failed"); auditErr != nil {
		return fmt.Errorf("update async audit failed result: %w", auditErr)
	}

	return nil
}

func (w *Worker) recordAsyncAccepted(ctx context.Context, job Job) error {
	if w.auditor == nil {
		return nil
	}

	return w.auditor.RecordAsyncAccepted(ctx, audit.Entry{
		UserID:       "admin",
		Action:       job.JobType,
		ResourceType: "job",
		ResourceID:   job.ID,
		Result:       "accepted",
		CreatedAt:    w.now(),
	})
}

func (w *Worker) updateAsyncResult(ctx context.Context, job Job, result string) error {
	if w.auditor == nil {
		return nil
	}

	return w.auditor.UpdateAsyncResult(ctx, job.JobType, "job", job.ID, result)
}

func retryBackoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return defaultBackoff1
	case 2:
		return defaultBackoff2
	default:
		return defaultBackoff3
	}
}

type ExecutionError struct {
	Code      string
	Message   string
	Retryable bool
}

func (e ExecutionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func classifyError(err error) ExecutionError {
	var execErr ExecutionError
	if errors.As(err, &execErr) {
		if execErr.Code == "" {
			execErr.Code = "ANSIBLE_UNKNOWN_EXIT"
		}
		if execErr.Message == "" {
			execErr.Message = execErr.Code
		}
		return execErr
	}

	return ExecutionError{
		Code:      "ANSIBLE_UNKNOWN_EXIT",
		Message:   err.Error(),
		Retryable: false,
	}
}

func truncateError(msg string) string {
	const maxLen = 10 * 1024
	if len(msg) <= maxLen {
		return msg
	}
	return msg[len(msg)-maxLen:]
}
