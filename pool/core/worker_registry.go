package core

import (
	"sync"
)

type WorkerRegistry struct {
	workers map[string]*Worker
	mu      sync.RWMutex
}

func NewWorkerRegistry() *WorkerRegistry {
	return &WorkerRegistry{
		workers: make(map[string]*Worker),
	}
}

func (wr *WorkerRegistry) Add(w *Worker) {
	if w == nil {
		return
	}
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.workers[w.ID] = w
}

func (wr *WorkerRegistry) Get(id string) (*Worker, bool) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	w, ok := wr.workers[id]
	return w, ok
}

func (wr *WorkerRegistry) Remove(id string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	delete(wr.workers, id)
}
