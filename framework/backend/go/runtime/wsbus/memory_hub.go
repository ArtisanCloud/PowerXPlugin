package wsbus

import (
	"context"
	"sync"
	"sync/atomic"
)

type Event struct {
	Topic      string `json:"topic"`
	Payload    any    `json:"payload"`
	TenantUUID string `json:"tenant_uuid"`
	MemberUUID string `json:"member_uuid,omitempty"`
	TraceID    string `json:"trace_id"`
}

type MemoryHub struct {
	mu       sync.RWMutex
	nextID   uint64
	handlers map[string]map[uint64]func(Event)
}

func NewMemoryHub() *MemoryHub {
	return &MemoryHub{
		handlers: make(map[string]map[uint64]func(Event)),
	}
}

// Publish broadcasts the event to all subscribers of the topic.
func (h *MemoryHub) Publish(_ context.Context, topic string, payload any, opts PublishOptions) error {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	subs := h.handlers[topic]
	if len(subs) == 0 {
		return nil
	}
	ev := Event{
		Topic:      topic,
		Payload:    payload,
		TenantUUID: opts.TenantUUID,
		MemberUUID: opts.MemberUUID,
		TraceID:    opts.TraceID,
	}
	for _, handler := range subs {
		if handler != nil {
			handler(ev)
		}
	}
	return nil
}

// Subscribe registers a handler for the topic. It returns an unsubscribe function.
func (h *MemoryHub) Subscribe(topic string, handler func(Event)) func() {
	if h == nil || handler == nil {
		return func() {}
	}
	id := atomic.AddUint64(&h.nextID, 1)
	h.mu.Lock()
	if h.handlers[topic] == nil {
		h.handlers[topic] = make(map[uint64]func(Event))
	}
	h.handlers[topic][id] = handler
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		if subs := h.handlers[topic]; subs != nil {
			delete(subs, id)
			if len(subs) == 0 {
				delete(h.handlers, topic)
			}
		}
		h.mu.Unlock()
	}
}

func (h *MemoryHub) Subscribers(topic string) int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.handlers[topic])
}
