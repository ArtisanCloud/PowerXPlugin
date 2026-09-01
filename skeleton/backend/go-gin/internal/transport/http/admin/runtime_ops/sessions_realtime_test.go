package runtime_ops

import (
	"testing"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
)

func TestSessionsHandlerRejectsUndeclaredMCPPublish(t *testing.T) {
	handler := &SessionsHandler{broker: stream.NewBroker()}
	if handler.publishEvent("tenant-1", "session-1", "session.ready", map[string]any{"ok": true}) {
		t.Fatal("undeclared MCP publish must be rejected")
	}
}

func TestSessionsHandlerPublishesDeclaredMCPEvent(t *testing.T) {
	broker := stream.NewBroker()
	events, cancel := broker.Subscribe("session-1")
	defer cancel()
	handler := &SessionsHandler{
		broker: broker,
		descriptors: []frameworkrealtime.Descriptor{{
			Key:        "_channel.mcp.session",
			Protocols:  []frameworkrealtime.Protocol{frameworkrealtime.ProtocolSSE},
			Actions:    []frameworkrealtime.Action{frameworkrealtime.ActionPublish},
			Scope:      frameworkrealtime.ScopeTenant,
			EventTypes: []string{"session.ready"},
		}},
	}
	if !handler.publishEvent("tenant-1", "session-1", "session.ready", map[string]any{"ok": true}) {
		t.Fatal("declared MCP publish must be accepted")
	}
	event := <-events
	if event.Type != "session.ready" || event.SessionID != "session-1" {
		t.Fatalf("unexpected event: %#v", event)
	}
}
