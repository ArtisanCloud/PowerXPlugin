package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHostProviderForwardsTenantResolvedTopicKeys(t *testing.T) {
	tenant := "22222222-2222-4222-8222-222222222222"
	plugin := "com.powerx.plugins.fixture"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "ApiKey tenant-key", r.Header.Get("Authorization"))
		var request struct {
			Payload struct {
				Body struct {
					Operation string   `json:"operation"`
					Tenant    string   `json:"tenant_uuid"`
					Plugin    string   `json:"plugin_id"`
					Keys      []string `json:"topic_keys"`
				} `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		body := request.Payload.Body
		require.Equal(t, "resolve_topics", body.Operation)
		require.Equal(t, tenant, body.Tenant)
		require.Equal(t, plugin, body.Plugin)
		require.Equal(t, []string{"fixture.changed"}, body.Keys)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "result": eventfabric.ResolvedTopics{ContractVersion: eventfabric.Contract, TenantUUID: tenant, PluginID: plugin, TopicBindings: map[string]string{"fixture.changed": tenant + ".powerx.fixture.changed"}}}})
	}))
	defer s.Close()
	cfg := &config.Config{Gateway: &config.GatewayConfig{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "tenant-key"}, Context: &config.ContextConfig{ProviderMode: "local"}, GRPCUpstream: &config.GRPCUpstream{TenantUUID: tenant}, EventBridge: &config.EventBridgeConfig{SourcePlugin: plugin, PublishTopicKeys: []string{"fixture.changed"}}}
	emitter, err := newHostTaskBusProvider(cfg, nil).NewEmitter()
	require.NoError(t, err)
	runtime, ok := emitter.(*eventfabric.Runtime)
	require.True(t, ok)
	defer runtime.Close()
	require.Equal(t, tenant+".powerx.fixture.changed", runtime.TopicBindings()["fixture.changed"])
}
