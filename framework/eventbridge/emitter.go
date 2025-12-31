package eventbridge

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
)

type Emitter interface {
	Emit(ctx context.Context, e event.Event) error
}

type Factory struct {
	cfg Config

	taskBusProvider func() (Emitter, error)
	metrics         MetricsRecorder
}

func NewFactory(cfg Config) (*Factory, error) {
	normalized, err := cfg.Normalized()
	if err != nil {
		return nil, err
	}
	return &Factory{cfg: normalized}, nil
}

func (f *Factory) WithTaskBusProvider(provider func() (Emitter, error)) *Factory {
	f.taskBusProvider = provider
	return f
}

func (f *Factory) WithMetrics(recorder MetricsRecorder) *Factory {
	f.metrics = recorder
	return f
}

func (f *Factory) NewEmitter() (Emitter, error) {
	if f == nil {
		return nil, ErrNotConfigured
	}

	local := NewLocalEmitter(f.cfg.LocalQueueSize)
	metrics := f.metrics

	if !f.cfg.Enabled || f.cfg.Mode == "local" {
		return newInstrumentedEmitter(local, metrics), nil
	}

	if f.taskBusProvider == nil {
		if f.cfg.FallbackToLocal {
			return newInstrumentedEmitter(local, metrics), nil
		}
		return nil, ErrTaskBusRequired
	}

	taskbus, err := f.taskBusProvider()
	if err != nil {
		if f.cfg.FallbackToLocal {
			return newInstrumentedEmitter(local, metrics), nil
		}
		return nil, err
	}

	if f.cfg.Mode == "dual" {
		return newInstrumentedEmitter(&dualEmitter{primary: taskbus, secondary: local}, metrics), nil
	}
	return newInstrumentedEmitter(taskbus, metrics), nil
}

type dualEmitter struct {
	primary   Emitter
	secondary Emitter
}

func (d *dualEmitter) Emit(ctx context.Context, e event.Event) error {
	if d == nil || d.primary == nil {
		return ErrNotConfigured
	}
	if err := d.primary.Emit(ctx, e); err != nil {
		return err
	}
	if d.secondary == nil {
		return nil
	}
	_ = d.secondary.Emit(ctx, e)
	return nil
}

// Drain proxies LocalEmitter.Drain when the underlying emitter supports it.
// This is primarily used by tests to assert fallback behavior without relying on concrete types.
func Drain(e Emitter) []event.Event {
	if e == nil {
		return nil
	}
	if d, ok := e.(interface{ Drain() []event.Event }); ok {
		return d.Drain()
	}
	if d, ok := e.(*instrumentedEmitter); ok && d != nil && d.inner != nil {
		if innerDrain, ok := d.inner.(interface{ Drain() []event.Event }); ok {
			return innerDrain.Drain()
		}
	}
	return nil
}

type instrumentedEmitter struct {
	inner   Emitter
	metrics MetricsRecorder
}

func newInstrumentedEmitter(inner Emitter, metrics MetricsRecorder) Emitter {
	if inner == nil {
		return nil
	}
	if metrics == nil {
		return inner
	}
	return &instrumentedEmitter{inner: inner, metrics: metrics}
}

func (e *instrumentedEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.inner == nil {
		return ErrNotConfigured
	}

	start := time.Now()
	err := e.inner.Emit(ctx, ev)
	latencyMs := float64(time.Since(start).Milliseconds())

	result := "success"
	if err != nil {
		result = "error"
	}

	e.metrics.RecordEmit(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), result)
	e.metrics.ObserveLatencyMs(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), "emit", latencyMs)
	return err
}
