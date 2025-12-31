package eventbridge

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
)

type LocalEmitter struct {
	queue chan event.Event
}

func NewLocalEmitter(queueSize int) *LocalEmitter {
	if queueSize <= 0 {
		queueSize = 1024
	}
	return &LocalEmitter{
		queue: make(chan event.Event, queueSize),
	}
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

