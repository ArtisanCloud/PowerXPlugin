package taskbus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric"
	"github.com/stretchr/testify/require"
)

func TestHostProviderResolvedTenantTopicsAndNoStaticMixing(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "ApiKey tenant-key", r.Header.Get("Authorization"))
		var request struct {
			Capability string `json:"capability_id"`
			Protocol   string `json:"preferred_protocol"`
			Payload    struct {
				Body struct {
					Operation string   `json:"operation"`
					Tenant    string   `json:"tenant_uuid"`
					Plugin    string   `json:"plugin_id"`
					Keys      []string `json:"topic_keys"`
				} `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, eventfabric.PublishCapability, request.Capability)
		require.Equal(t, eventfabric.Protocol, request.Protocol)
		require.Equal(t, "resolve_topics", request.Payload.Body.Operation)
		require.Equal(t, fixtureTenant, request.Payload.Body.Tenant)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"status": "completed", "protocol_used": eventfabric.Protocol, "fallback_used": false, "result": eventfabric.ResolvedTopics{ContractVersion: eventfabric.Contract, TenantUUID: fixtureTenant, PluginID: request.Payload.Body.Plugin, TopicBindings: map[string]string{"fixture.event.v1": fixtureTenant + ".powerx.fixture.changed"}}}})
	}))
	defer s.Close()
	cfg := fixtureHostConfig(s.URL)
	cfg.TopicBindings = nil
	cfg.AuthScheme = "apikey"
	cfg.APIKey = "tenant-key"
	cfg.PublishTopicKeys = []string{"fixture.event.v1"}
	emitter, err := NewHostProvider(cfg).NewEmitter()
	require.NoError(t, err)
	runtime, ok := emitter.(*eventfabric.Runtime)
	require.True(t, ok)
	defer runtime.Close()
	require.Equal(t, fixtureTenant+".powerx.fixture.changed", runtime.TopicBindings()["fixture.event.v1"])
	require.Equal(t, 1, calls)
	cfg.TopicBindings = map[string]string{"fixture.event.v1": fixtureTenant + ".powerx.fixture.system"}
	_, err = NewHostProvider(cfg).NewEmitter()
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
