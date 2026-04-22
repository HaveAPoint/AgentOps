package executor

import (
	"context"
	"sync"
)

type CancelRegistry struct {
	mu      sync.Mutex
	cancels map[int64]context.CancelFunc
}

func NewCancelRegistry() *CancelRegistry {
	return &CancelRegistry{
		cancels: make(map[int64]context.CancelFunc),
	}
}

func (r *CancelRegistry) Register(taskID int64, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cancels[taskID] = cancel
}

func (r *CancelRegistry) Cancel(taskID int64) bool {
	r.mu.Lock()
	cancel, ok := r.cancels[taskID]
	r.mu.Unlock()

	if !ok {
		return false
	}

	cancel()
	return true
}

func (r *CancelRegistry) Unregister(taskID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cancels, taskID)
}
