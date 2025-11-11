package watch

import (
	"sync"
	"time"
)

// Debouncer handles debouncing of events
type Debouncer struct {
	mu       sync.Mutex
	events   []FileEvent
	timer    *time.Timer
	delay    time.Duration
	callback func([]FileEvent)
}

// NewDebouncer creates a new debouncer
func NewDebouncer(delay time.Duration, callback func([]FileEvent)) *Debouncer {
	return &Debouncer{
		events:   []FileEvent{},
		delay:    delay,
		callback: callback,
	}
}

// AddEvent adds an event to the debouncer
func (d *Debouncer) AddEvent(event FileEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.events = append(d.events, event)

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		events := make([]FileEvent, len(d.events))
		copy(events, d.events)
		d.events = []FileEvent{}
		d.mu.Unlock()

		if d.callback != nil && len(events) > 0 {
			d.callback(events)
		}
	})
}

// Flush flushes all pending events immediately
func (d *Debouncer) Flush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	if len(d.events) > 0 && d.callback != nil {
		events := make([]FileEvent, len(d.events))
		copy(events, d.events)
		d.events = []FileEvent{}
		d.callback(events)
	}
}

// Stop stops the debouncer
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.events = []FileEvent{}
}
