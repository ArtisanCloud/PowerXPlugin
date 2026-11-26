package eventbridge

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
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
	if !f.cfg.Enabled {
		var opts []LocalEmitterOption
		if recorder, ok := f.metrics.(DropRecorder); ok && recorder != nil {
			opts = append(opts, WithDropRecorder(recorder))
		}
		return newInstrumentedEmitter(NewLocalEmitter(f.cfg.LocalQueueSize, opts...), f.metrics), nil
	}
	if f.taskBusProvider == nil {
		return nil, ErrTaskBusRequired
	}
	selected, err := f.taskBusProvider()
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, ErrTaskBusRequired
	}
	reliable, ok := selected.(interface{ ReliableHost() bool })
	isHost := ok && reliable.ReliableHost()
	if isHost != (f.cfg.RuntimeMode.TaskBusProvider() == "host") {
		return nil, errors.Join(ErrReliableHostRequired, Close(selected))
	}
	slog.Info("EVENT_BRIDGE_RUNTIME_SELECTED", "provider_mode", f.cfg.RuntimeMode.ProviderMode, "taskbus_provider", f.cfg.RuntimeMode.TaskBusProvider())
	return newInstrumentedEmitter(selected, f.metrics), nil
}

// Drain proxies LocalEmitter.Drain when the underlying emitter supports it.
// This is primarily used to inspect an explicitly selected local emitter.
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

// Close releases the selected transport even when metrics wrap the emitter.
func Close(e Emitter) error {
	if wrapped, ok := e.(*instrumentedEmitter); ok && wrapped != nil {
		return Close(wrapped.inner)
	}
	if closer, ok := e.(interface{ Close() error }); ok {
		return closer.Close()
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
