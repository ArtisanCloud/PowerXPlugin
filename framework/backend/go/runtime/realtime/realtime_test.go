package realtime

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ssebus"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
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

func TestLoadDescriptorsAndDecide(t *testing.T) {
	descriptors, err := LoadDescriptors([]byte(`
events:
  channels:
    - key: _channel.job.progress
      protocols: [sse]
      actions: [publish, subscribe]
      scope: member
      event_types: [progress, done]
`))
	if err != nil {
		t.Fatalf("LoadDescriptors: %v", err)
	}
	decision := Decide(descriptors, ActionSubscribe, "_channel.job.progress", ProtocolSSE, "progress", Scope{
		TenantUUID: "tenant-1",
		MemberUUID: "member-1",
		TraceID:    "trace-1",
	})
	if !decision.Allowed || decision.Reason != "REALTIME_ALLOWED" {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestLoadDescriptorsRejectsIncompleteOrDuplicateDeclarations(t *testing.T) {
	_, err := LoadDescriptors([]byte(`
events:
  topics:
    - key: _topic.incomplete
      actions: [publish]
      scope: tenant
`))
	if err == nil {
		t.Fatal("expected incomplete descriptor to be rejected")
	}

	err = ValidateDescriptors([]Descriptor{
		{Key: "_topic.duplicate", Protocols: []Protocol{ProtocolWS}, Actions: []Action{ActionPublish}, Scope: ScopeGlobal},
		{Key: "_topic.duplicate", Protocols: []Protocol{ProtocolWS}, Actions: []Action{ActionPublish}, Scope: ScopeGlobal},
	})
	if err == nil {
		t.Fatal("expected duplicate descriptor to be rejected")
	}
}

func TestDecideDenyByDefaultForScopeAndAction(t *testing.T) {
	descriptors := []Descriptor{{
		Key:       "_topic.job.updated",
		Protocols: []Protocol{ProtocolWS},
		Actions:   []Action{ActionPublish},
		Scope:     ScopeTenant,
	}}
	denied := Decide(descriptors, ActionSubscribe, "_topic.job.updated", ProtocolWS, "", Scope{TenantUUID: "tenant-1"})
	if denied.Allowed || denied.Reason != "REALTIME_ACTION_NOT_ALLOWED" {
		t.Fatalf("action decision=%+v", denied)
	}
	denied = Decide(descriptors, ActionPublish, "_topic.job.updated", ProtocolWS, "", Scope{})
	if denied.Allowed || denied.Reason != "REALTIME_SCOPE_INVALID" {
		t.Fatalf("scope decision=%+v", denied)
	}
}

func TestDecideMatchesOnlyExactScopeTemplate(t *testing.T) {
	descriptors := []Descriptor{{
		Key:        "_topic.notify.tenant.{{tenant_uuid}}",
		Protocols:  []Protocol{ProtocolWS},
		Actions:    []Action{ActionPublish, ActionSubscribe},
		Scope:      ScopeTenant,
		EventTypes: []string{"message"},
	}}
	allowed := Decide(descriptors, ActionSubscribe, "_topic.notify.tenant.tenant-1", ProtocolWS, "message", Scope{TenantUUID: "tenant-1"})
	if !allowed.Allowed {
		t.Fatalf("template decision=%+v", allowed)
	}
	denied := Decide(descriptors, ActionSubscribe, "_topic.notify.tenant.tenant-2", ProtocolWS, "message", Scope{TenantUUID: "tenant-1"})
	if denied.Allowed || denied.Reason != "REALTIME_DESCRIPTOR_NOT_FOUND" {
		t.Fatalf("cross-tenant decision=%+v", denied)
	}
}

func TestAuthorizedWSPublisherRejectsUndeclaredTopic(t *testing.T) {
	hub := fwwsbus.NewMemoryHub()
	publisher := NewAuthorizedWSPublisher(fwwsbus.NewLocalPublisher(hub, nil), []Descriptor{{
		Key:        "_topic.job.progress",
		Protocols:  []Protocol{ProtocolWS},
		Actions:    []Action{ActionPublish},
		Scope:      ScopeTenant,
		EventTypes: []string{"message"},
	}}, "message")
	result := publisher.Publish(context.Background(), "_topic.unknown", map[string]bool{"ok": true}, fwwsbus.PublishOptions{TenantUUID: "tenant-1"})
	if result.OK || result.ErrorCode != "REALTIME_DESCRIPTOR_NOT_FOUND" {
		t.Fatalf("result=%+v", result)
	}
}

func TestManagedSSERejectsUndeclaredAndDeliversDeclaredChannel(t *testing.T) {
	broker := ssebus.NewBroker()
	descriptors := []Descriptor{{
		Key:        "_channel.job.progress",
		Protocols:  []Protocol{ProtocolSSE},
		Actions:    []Action{ActionPublish, ActionSubscribe},
		Scope:      ScopeMember,
		EventTypes: []string{"progress"},
	}}
	scope := Scope{TenantUUID: "tenant-1", MemberUUID: "member-1", TraceID: "trace-1"}
	deniedEvents, _, denied := SubscribeManagedSSE(broker, descriptors, "_channel.unknown", "progress", scope)
	if denied.Allowed || denied.Reason != "REALTIME_DESCRIPTOR_NOT_FOUND" {
		t.Fatalf("denied=%+v", denied)
	}
	if _, ok := <-deniedEvents; ok {
		t.Fatal("denied subscription must be closed")
	}

	events, cancel, allowed := SubscribeManagedSSE(broker, descriptors, "_channel.job.progress", "progress", scope)
	defer cancel()
	if !allowed.Allowed {
		t.Fatalf("allowed=%+v", allowed)
	}
	decision, err := PublishManagedSSE(context.Background(), broker, descriptors, "_channel.job.progress", "progress", map[string]int{"percent": 50}, scope)
	if err != nil || !decision.Allowed {
		t.Fatalf("publish decision=%+v err=%v", decision, err)
	}
	select {
	case event := <-events:
		if event.EventType != "progress" || event.Channel != "_channel.job.progress" {
			t.Fatalf("event=%+v", event)
		}
	case <-context.Background().Done():
		t.Fatal("unreachable")
	}
}
