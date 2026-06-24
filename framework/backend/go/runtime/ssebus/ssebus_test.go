package ssebus

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"
)

type flushRecorder struct {
	*bytes.Buffer
	header http.Header
	code   int
}

func (r *flushRecorder) Header() http.Header {
	return r.header
}

func (r *flushRecorder) WriteHeader(code int) {
	r.code = code
}

func (r *flushRecorder) Flush() {}

func TestBrokerPublishSubscribe(t *testing.T) {
	b := NewBroker()
	ch, cancel := b.Subscribe("tenant.member")
	defer cancel()

	if err := b.Publish(context.Background(), "tenant.member", "progress", map[string]int{"pct": 80}, PublishOptions{TenantUUID: "tenant-a", MemberUUID: "member-a"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Channel != "tenant.member" || ev.EventType != "progress" || ev.TenantUUID != "tenant-a" || ev.MemberUUID != "member-a" {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("expected event")
	}
}

func TestWriteEvent(t *testing.T) {
	buf := &bytes.Buffer{}
	rec := &flushRecorder{Buffer: buf, header: make(http.Header)}

	if err := WriteEvent(rec, "done", Event{EventType: "done", Payload: map[string]string{"ok": "true"}}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	got := buf.String()
	if got == "" || !bytes.Contains(buf.Bytes(), []byte("event: done\n")) || !bytes.Contains(buf.Bytes(), []byte("data:")) {
		t.Fatalf("unexpected sse payload: %q", got)
	}
}

func TestServeStreamWritesHeadersAndEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan Event, 1)
	events <- Event{EventType: "message", Payload: "hello"}
	close(events)

	buf := &bytes.Buffer{}
	rec := &flushRecorder{Buffer: buf, header: make(http.Header)}
	ServeStream(ctx, rec, events, StreamOptions{HeartbeatEvery: time.Hour})

	if rec.code != http.StatusOK {
		t.Fatalf("status code = %d", rec.code)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
	}
	if !bytes.Contains(buf.Bytes(), []byte("event: message\n")) {
		t.Fatalf("expected event output, got %q", buf.String())
	}
}
