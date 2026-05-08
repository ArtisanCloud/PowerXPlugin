package eventbridge

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
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

	opts := make([]LocalEmitterOption, 0, 1)
	metrics := f.metrics
	if dropRecorder, ok := metrics.(DropRecorder); ok && dropRecorder != nil {
		opts = append(opts, WithDropRecorder(dropRecorder))
	}

	local := NewLocalEmitter(f.cfg.LocalQueueSize, opts...)
	hostManaged := strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("IAM_MODE")), "delegated") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("IAMMode")), "delegated")
	slog.Info("eventbridge factory decide",
		"enabled", f.cfg.Enabled,
		"mode", f.cfg.Mode,
		"fallback_to_local", f.cfg.FallbackToLocal,
		"has_taskbus_provider", f.taskBusProvider != nil,
		"local_queue_size", f.cfg.LocalQueueSize,
		"host_managed", hostManaged,
	)

	// 宿主模式硬约束：禁止 local/fallback，必须 taskbus。
	if hostManaged {
		if !f.cfg.Enabled || f.cfg.Mode == "local" {
			slog.Error("eventbridge host-mode guard failed",
				"reason", "host_mode_requires_taskbus_mode",
				"enabled", f.cfg.Enabled,
				"mode", f.cfg.Mode,
			)
			return nil, ErrTaskBusRequired
		}
		if f.taskBusProvider == nil {
			slog.Error("eventbridge host-mode guard failed",
				"reason", "host_mode_requires_taskbus_provider",
			)
			return nil, ErrTaskBusRequired
		}
	}

	if !f.cfg.Enabled || f.cfg.Mode == "local" {
		slog.Info("eventbridge factory chose local",
			"reason", "disabled_or_local_mode",
		)
		return newInstrumentedEmitter(local, metrics), nil
	}

	if f.taskBusProvider == nil {
		if f.cfg.FallbackToLocal {
			slog.Warn("eventbridge factory fallback local",
				"reason", "taskbus_provider_missing",
			)
			return newInstrumentedEmitter(local, metrics), nil
		}
		slog.Error("eventbridge factory failed",
			"reason", "taskbus_provider_missing",
		)
		return nil, ErrTaskBusRequired
	}

	taskbus, err := f.taskBusProvider()
	if err != nil {
		if f.cfg.FallbackToLocal {
			slog.Warn("eventbridge factory fallback local",
				"reason", "taskbus_provider_error",
				"error", err.Error(),
			)
			return newInstrumentedEmitter(local, metrics), nil
		}
		slog.Error("eventbridge factory failed",
			"reason", "taskbus_provider_error",
			"error", err.Error(),
		)
		return nil, err
	}

	if f.cfg.Mode == "dual" {
		slog.Info("eventbridge factory chose dual",
			"taskbus_type", typeName(taskbus),
		)
		return newInstrumentedEmitter(&dualEmitter{primary: taskbus, secondary: local}, metrics), nil
	}
	slog.Info("eventbridge factory chose taskbus",
		"taskbus_type", typeName(taskbus),
	)
	return newInstrumentedEmitter(taskbus, metrics), nil
}

func typeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%T", v)
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
