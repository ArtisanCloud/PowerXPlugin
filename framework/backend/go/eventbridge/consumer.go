package eventbridge

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
)

type HandlerFunc func(ctx context.Context, ev event.Event) error

type Dispatcher struct {
	mu          sync.RWMutex
	handlers    map[event.Topic][]HandlerFunc
	idempotency *IdempotencyFilter
	metrics     MetricsRecorder
}

func NewDispatcher(idempotency *IdempotencyFilter) *Dispatcher {
	if idempotency == nil {
		idempotency = NewIdempotencyFilter(10000)
	}
	return &Dispatcher{
		handlers:    map[event.Topic][]HandlerFunc{},
		idempotency: idempotency,
	}
}

func (d *Dispatcher) WithMetrics(recorder MetricsRecorder) *Dispatcher {
	if d == nil {
		return nil
	}
	d.metrics = recorder
	return d
}

func (d *Dispatcher) Register(topic event.Topic, handler HandlerFunc) {
	if d == nil || handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[topic] = append(d.handlers[topic], handler)
}

func (d *Dispatcher) Dispatch(ctx context.Context, ev event.Event) error {
	if d == nil {
		return errors.New("dispatcher is nil")
	}

	if d.idempotency != nil && d.idempotency.SeenBefore(ev) {
		return nil
	}

	d.mu.RLock()
	hs := append([]HandlerFunc(nil), d.handlers[ev.Topic]...)
	d.mu.RUnlock()

	start := time.Now()
	result := "success"
	for _, h := range hs {
		if err := h(ctx, ev); err != nil {
			result = "error"
			if d.metrics != nil {
				d.metrics.RecordConsume(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
				d.metrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "consume", float64(time.Since(start).Milliseconds()))
			}
			return err
		}
	}
	if d.metrics != nil {
		d.metrics.RecordConsume(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
		d.metrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "consume", float64(time.Since(start).Milliseconds()))
	}
	return nil
}
