package ws

import (
	"sync"
	"time"
)

type ResultWaiter struct {
	mu       sync.Mutex
	pending  map[string]chan CommandResult
	canceled map[string]time.Time
	maxAge   time.Duration
}

func NewResultWaiter() *ResultWaiter {
	return &ResultWaiter{
		pending:  make(map[string]chan CommandResult),
		canceled: make(map[string]time.Time),
		maxAge:   2 * time.Minute,
	}
}

func (w *ResultWaiter) Register(commandID string) <-chan CommandResult {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan CommandResult, 1)
	w.pending[commandID] = ch
	w.pruneLocked(time.Now())
	return ch
}

func (w *ResultWaiter) Resolve(result CommandResult) bool {
	w.mu.Lock()
	if ch, ok := w.pending[result.CommandID]; ok {
		delete(w.pending, result.CommandID)
		w.mu.Unlock()
		ch <- result
		close(ch)
		return true
	}
	if _, ok := w.canceled[result.CommandID]; ok {
		delete(w.canceled, result.CommandID)
		w.mu.Unlock()
		return true
	}
	w.mu.Unlock()
	return false
}

func (w *ResultWaiter) Cancel(commandID string) {
	w.mu.Lock()
	if ch, ok := w.pending[commandID]; ok {
		delete(w.pending, commandID)
		w.canceled[commandID] = time.Now()
		w.mu.Unlock()
		close(ch)
		return
	}
	w.pruneLocked(time.Now())
	w.mu.Unlock()
}

func (w *ResultWaiter) pruneLocked(now time.Time) {
	if w.maxAge <= 0 {
		return
	}
	for id, ts := range w.canceled {
		if now.Sub(ts) > w.maxAge {
			delete(w.canceled, id)
		}
	}
}
