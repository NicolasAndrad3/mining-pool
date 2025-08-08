package core

import (
	"context"
	"errors"
	"sync"
)

type Worker struct {
	ID        string
	connected bool
	job       *Job
	shares    []*Share
	mu        sync.RWMutex
}

func NewWorker(id string) *Worker {
	return &Worker{
		ID:        id,
		connected: true,
		shares:    []*Share{},
	}
}

func (w *Worker) AssignJob(job *Job) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.job = job
}

func (w *Worker) CurrentJob() *Job {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.job
}

func (w *Worker) AddShare(ctx context.Context, s *Share) error {
	if s == nil {
		return errors.New("invalid share")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.shares = append(w.shares, s)
	return nil
}

func (w *Worker) IsActive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connected
}

func (w *Worker) Disconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connected = false
}
