package ssebus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultHeartbeatEvent = "ping"
	DefaultHeartbeatEvery = 25 * time.Second
)

type Event struct {
	Channel    string    `json:"channel,omitempty"`
	EventType  string    `json:"event_type"`
	Payload    any       `json:"payload,omitempty"`
	TenantUUID string    `json:"tenant_uuid,omitempty"`
	MemberUUID string    `json:"member_uuid,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type PublishOptions struct {
	TenantUUID string
	MemberUUID string
	TraceID    string
}

type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[string]map[chan Event]struct{})}
}

func NormalizeChannel(channel string) string {
	return strings.TrimSpace(channel)
}

func ValidateChannel(channel string) error {
	if NormalizeChannel(channel) == "" {
		return errors.New("sse channel is required")
	}
	return nil
}

func (b *Broker) Subscribe(channel string) (<-chan Event, func()) {
	channel = NormalizeChannel(channel)
	if b == nil || channel == "" {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}

	ch := make(chan Event, 16)
	b.mu.Lock()
	if b.subs[channel] == nil {
		b.subs[channel] = make(map[chan Event]struct{})
	}
	b.subs[channel][ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if subs := b.subs[channel]; subs != nil {
			if _, ok := subs[ch]; ok {
				delete(subs, ch)
				close(ch)
			}
			if len(subs) == 0 {
				delete(b.subs, channel)
			}
		}
		b.mu.Unlock()
	}
}

func (b *Broker) Publish(_ context.Context, channel string, eventType string, payload any, opts PublishOptions) error {
	channel = NormalizeChannel(channel)
	if err := ValidateChannel(channel); err != nil {
		return err
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = "message"
	}
	if b == nil {
		return nil
	}
	ev := Event{
		Channel:    channel,
		EventType:  eventType,
		Payload:    payload,
		TenantUUID: strings.TrimSpace(opts.TenantUUID),
		MemberUUID: strings.TrimSpace(opts.MemberUUID),
		TraceID:    strings.TrimSpace(opts.TraceID),
		Timestamp:  time.Now().UTC(),
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[channel] {
		select {
		case ch <- ev:
		default:
		}
	}
	return nil
}

func (b *Broker) Subscribers(channel string) int {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[NormalizeChannel(channel)])
}

type StreamOptions struct {
	HeartbeatEvery time.Duration
	HeartbeatEvent string
	HeartbeatData  any
	Heartbeat      func(time.Time) any
}

func ServeStream(ctx context.Context, w http.ResponseWriter, events <-chan Event, opts StreamOptions) {
	if ctx == nil {
		ctx = context.Background()
	}
	if w == nil {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	heartbeatEvery := opts.HeartbeatEvery
	if heartbeatEvery <= 0 {
		heartbeatEvery = DefaultHeartbeatEvery
	}
	heartbeatEvent := strings.TrimSpace(opts.HeartbeatEvent)
	if heartbeatEvent == "" {
		heartbeatEvent = DefaultHeartbeatEvent
	}
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Timestamp.IsZero() {
				ev.Timestamp = time.Now().UTC()
			}
			if strings.TrimSpace(ev.EventType) == "" {
				ev.EventType = "message"
			}
			if err := WriteEvent(w, ev.EventType, ev); err != nil {
				return
			}
			flusher.Flush()
		case now := <-ticker.C:
			data := opts.HeartbeatData
			if opts.Heartbeat != nil {
				data = opts.Heartbeat(now.UTC())
			}
			if data == nil {
				data = Event{EventType: heartbeatEvent, Timestamp: now.UTC()}
			}
			if err := WriteEvent(w, heartbeatEvent, data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func WriteEvent(w http.ResponseWriter, eventType string, data any) error {
	if w == nil {
		return errors.New("response writer is nil")
	}
	eventType = strings.TrimSpace(eventType)
	if eventType != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, "\n")
	return err
}
