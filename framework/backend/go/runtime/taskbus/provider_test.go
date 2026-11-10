package taskbus

import (
	"context"
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const fixtureTenant = "22222222-2222-4222-8222-222222222222"

func hostFixtureEvent() event.Event {
	return event.Event{Topic: "fixture.event.v1", Meta: event.Meta{TenantUUID: fixtureTenant, SourcePlugin: "com.powerx.plugins.fixture", RequestID: "request", TraceID: "business-trace", OccurredAt: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC), PayloadVersion: "1"}, Payload: json.RawMessage(`{"endpoint_uuid":"55555555-5555-4555-8555-555555555555"}`), Reliability: &event.ReliableIdentity{EventID: "stable-event", IdempotencyKey: "stable-key"}}
}
func fixtureHostConfig(base string) HostProviderConfig {
	return HostProviderConfig{BaseURL: base, TenantUUID: fixtureTenant, SourcePlugin: "com.powerx.plugins.fixture", TopicBindings: map[string]string{"fixture.event.v1": fixtureTenant + "._topic.fixture.changed"}}
}
func hostSuccess(w http.ResponseWriter) {
	_, _ = w.Write([]byte(`{"code":200,"data":{"status":"completed","protocol_used":"core_internal","fallback_used":false,"result":{"contract_version":"event_fabric.service.v1","accepted":true,"event_id":"stable-event","trace_id":"core-trace"}}}`))
}
func TestHostProviderUsesReliableServiceAndExplicitIdentity(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
		require.Equal(t, "Bearer fixture-sts", r.Header.Get("Authorization"))
		var body struct {
			Capability string `json:"capability_id"`
			Protocol   string `json:"preferred_protocol"`
			Payload    struct {
				Body struct {
					Event struct {
						ID      string      `json:"event_id"`
						Key     string      `json:"idempotency_key"`
						Payload event.Event `json:"payload"`
					} `json:"event"`
				} `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "com.corex.eventfabric.service_publish", body.Capability)
		require.Equal(t, "core_internal", body.Protocol)
		require.Equal(t, "stable-event", body.Payload.Body.Event.ID)
		require.Equal(t, "stable-key", body.Payload.Body.Event.Key)
		require.Equal(t, "business-trace", body.Payload.Body.Event.Payload.Meta.TraceID)
		hostSuccess(w)
	}))
	defer server.Close()
	cfg := fixtureHostConfig(server.URL)
	cfg.TokenProvider = func(context.Context) (string, error) { return "fixture-sts", nil }
	emitter, err := NewHostProvider(cfg).NewEmitter()
	require.NoError(t, err)
	ev := hostFixtureEvent()
	require.NoError(t, emitter.Emit(middleware.WithBearerToken(context.Background(), "fixture-admin"), ev))
	require.Equal(t, 1, calls)
	ev.Reliability = nil
	require.Error(t, emitter.Emit(context.Background(), ev))
	require.Equal(t, 1, calls)
}
func TestHostProviderRequiresExplicitBindings(t *testing.T) {
	cfg := fixtureHostConfig("http://core.invalid")
	cfg.AuthScheme = "apikey"
	cfg.APIKey = "fixture"
	cfg.TopicBindings = nil
	_, err := NewHostProvider(cfg).NewEmitter()
	require.Error(t, err)
}
