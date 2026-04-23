package logging

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Event struct {
	Message   string
	Level     string
	Fields    Fields
	Timestamp time.Time
}

type Sink interface {
	Name() SinkType
	Emit(ctx context.Context, event Event) error
}

type SinkRegistry struct {
	mu    sync.RWMutex
	sinks map[SinkType]Sink
}

func NewSinkRegistry() *SinkRegistry {
	return &SinkRegistry{sinks: map[SinkType]Sink{}}
}

func (r *SinkRegistry) Register(s Sink) error {
	if r == nil {
		return fmt.Errorf("sink registry is nil")
	}
	if s == nil {
		return fmt.Errorf("sink is nil")
	}
	name := s.Name()
	if name == "" {
		return fmt.Errorf("sink name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sinks[name] = s
	return nil
}

func (r *SinkRegistry) Resolve(name SinkType) (Sink, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sinks[name]
	return s, ok
}
