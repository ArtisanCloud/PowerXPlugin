package eventbridge

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
)

const localQueueFullReason = "queue_full"

type DropRecorder interface {
	RecordDrop(pluginID, tenantUUID, topic, reason string)
}

type LocalEmitterOption func(*LocalEmitter)

func WithDropRecorder(recorder DropRecorder) LocalEmitterOption {
	return func(e *LocalEmitter) {
		if e == nil {
			return
		}
		e.dropRecorder = recorder
	}
}

type LocalEmitter struct {
	queue        chan event.Event
	dropRecorder DropRecorder
}

func NewLocalEmitter(queueSize int, opts ...LocalEmitterOption) *LocalEmitter {
	if queueSize <= 0 {
		queueSize = 1024
	}
	emitter := &LocalEmitter{
		queue: make(chan event.Event, queueSize),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(emitter)
		}
	}
	return emitter
}

func (e *LocalEmitter) Emit(ctx context.Context, ev event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case e.queue <- ev:
		return nil
	default:
		e.recordDrop(ev, localQueueFullReason)
		return nil
	}
}

func (e *LocalEmitter) Drain() []event.Event {
	var out []event.Event
	for {
		select {
		case ev := <-e.queue:
			out = append(out, ev)
		default:
			return out
		}
	}
}

func (e *LocalEmitter) recordDrop(ev event.Event, reason string) {
	if e == nil || e.dropRecorder == nil {
		return
	}
	e.dropRecorder.RecordDrop(ev.Meta.SourcePlugin, ev.Meta.TenantUUID, string(ev.Topic), reason)
}
