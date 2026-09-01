package wsbus

import (
	"context"
	"testing"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
)

type captureSubscriber struct {
	handler func(fwwsbus.Event)
}

func (s *captureSubscriber) Subscribe(_ string, handler func(fwwsbus.Event)) func() {
	s.handler = handler
	return func() {}
}

func TestSubscribeFiltersMemberScopedEvents(t *testing.T) {
	subscriber := &captureSubscriber{}
	conn := &wsConn{
		subs:       map[string]func(){},
		tenantUUID: "tenant-1",
		memberUUID: "11111111-1111-4111-8111-111111111111",
	}
	sent := 0
	conn.sendHook = func(wsResponse) {
		sent++
	}
	conn.subscribe(subscriber, []string{"_topic.notify.member.11111111-1111-4111-8111-111111111111"}, []frameworkrealtime.Descriptor{{Key: "_topic.notify.member.11111111-1111-4111-8111-111111111111", Protocols: []frameworkrealtime.Protocol{frameworkrealtime.ProtocolWS}, Actions: []frameworkrealtime.Action{frameworkrealtime.ActionSubscribe}, Scope: frameworkrealtime.ScopeMember, EventTypes: []string{"message"}}})
	if subscriber.handler == nil {
		t.Fatal("expected subscription handler")
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.member.22222222-2222-4222-8222-222222222222",
		TenantUUID: "tenant-1",
		MemberUUID: "22222222-2222-4222-8222-222222222222",
	})
	if sent != 0 {
		t.Fatalf("cross-member event was delivered, sent=%d", sent)
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.member.11111111-1111-4111-8111-111111111111",
		TenantUUID: "tenant-1",
		MemberUUID: "11111111-1111-4111-8111-111111111111",
	})
	if sent != 1 {
		t.Fatalf("own member event not delivered, sent=%d", sent)
	}

	subscriber.handler(fwwsbus.Event{
		Topic:      "_topic.notify.tenant.tenant-1",
		TenantUUID: "tenant-1",
	})
	if sent != 2 {
		t.Fatalf("tenant broadcast event not delivered, sent=%d", sent)
	}
}

func TestMemoryHubCarriesMemberUUID(t *testing.T) {
	hub := fwwsbus.NewMemoryHub()
	received := make(chan fwwsbus.Event, 1)
	hub.Subscribe("_topic.notify.member.11111111-1111-4111-8111-111111111111", func(ev fwwsbus.Event) {
		received <- ev
	})
	err := hub.Publish(context.Background(), "_topic.notify.member.11111111-1111-4111-8111-111111111111", map[string]any{"ok": true}, fwwsbus.PublishOptions{
		TenantUUID: "tenant-1",
		MemberUUID: "11111111-1111-4111-8111-111111111111",
		TraceID:    "trace-1",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	ev := <-received
	if ev.MemberUUID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("member_uuid=%q", ev.MemberUUID)
	}
}
