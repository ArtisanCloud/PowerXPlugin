package realtime

import (
	"bytes"
	"context"
	"net/http"
	"testing"
)

type recorder struct {
	*bytes.Buffer
	header http.Header
	code   int
}

func (r *recorder) Header() http.Header  { return r.header }
func (r *recorder) WriteHeader(code int) { r.code = code }
func (r *recorder) Flush()               {}

func TestMemberKey(t *testing.T) {
	got, err := MemberKey("_topic.notify", "tenant-1", "member-1")
	if err != nil {
		t.Fatalf("MemberKey: %v", err)
	}
	if got != "_topic.notify.tenant.tenant-1.member.member-1" {
		t.Fatalf("key=%q", got)
	}
}

func TestWriteSSEEnvelope(t *testing.T) {
	rec := &recorder{Buffer: &bytes.Buffer{}, header: make(http.Header)}
	err := WriteSSEEnvelope(rec, NewSSEEnvelope("_channel.test", "progress", map[string]int{"pct": 10}, Scope{TenantUUID: "tenant-1"}))
	if err != nil {
		t.Fatalf("WriteSSEEnvelope: %v", err)
	}
	if !bytes.Contains(rec.Bytes(), []byte("event: progress\n")) {
		t.Fatalf("missing event: %q", rec.String())
	}
	if !bytes.Contains(rec.Bytes(), []byte(`"channel":"_channel.test"`)) {
		t.Fatalf("missing channel: %q", rec.String())
	}
}

func TestProxySSEStreamPreservesRawEvents(t *testing.T) {
	rec := &recorder{Buffer: &bytes.Buffer{}, header: make(http.Header)}
	body := bytes.NewBufferString("event: token\ndata: {\"text\":\"hi\"}\n\n")
	err := ProxySSEStream(context.Background(), rec, StreamThroughOptions{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       body,
		TraceID:    "trace-1",
	})
	if err != nil {
		t.Fatalf("ProxySSEStream: %v", err)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	if !bytes.Contains(rec.Bytes(), []byte("event: token\n")) {
		t.Fatalf("raw event not preserved: %q", rec.String())
	}
	if rec.Header().Get("X-Trace-ID") != "trace-1" {
		t.Fatalf("trace header=%q", rec.Header().Get("X-Trace-ID"))
	}
}
