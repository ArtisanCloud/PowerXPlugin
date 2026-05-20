package scheduler

import (
	"context"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[string]HandlerFunc{}}
}

func (r *Registry) RegisterHandler(jobName string, handler HandlerFunc) {
	if r == nil || handler == nil {
		return
	}
	key := strings.TrimSpace(jobName)
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[key] = handler
}

func (r *Registry) HandleTriggered(ctx context.Context, job TriggeredJob) error {
	if r == nil {
		return ErrHandlerNotFound
	}
	key := strings.TrimSpace(job.JobName)
	if key == "" {
		key = strings.TrimSpace(job.BusinessAction)
	}
	r.mu.RLock()
	handler := r.handlers[key]
	r.mu.RUnlock()
	if handler == nil {
		return ErrHandlerNotFound
	}
	return handler(ctx, job)
}
