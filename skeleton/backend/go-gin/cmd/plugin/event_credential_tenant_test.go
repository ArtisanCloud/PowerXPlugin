package main

import (
	"encoding/json"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostEventTenantComesFromServiceCredential(t *testing.T) {
	for _, profile := range []struct {
		name, prefix string
		prefixedBase bool
	}{
		{"origin", "/api/v1", false},
		{"api_root", "/api/v1", true},
		{"custom_prefix", "/custom", true},
	} {
		t.Run(profile.name, func(t *testing.T) {
			tenant := "22222222-2222-4222-8222-222222222222"
			reads, resolves := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "ApiKey fixture-key" {
					t.Error("SERVICE_CREDENTIAL_CHANGED")
				}
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == profile.prefix+"/tenant/iam/tenant" {
					reads++
					json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"tenant_uuid": tenant, "name": "fixture"}})
					return
				}
				if r.URL.Path != profile.prefix+"/tenant/invocations" {
					t.Error("UNEXPECTED_EVENT_PATH")
					w.WriteHeader(404)
					return
				}
				resolves++
				var in struct {
					Payload struct {
						Body map[string]any `json:"body"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					t.Error(err)
				}
				if in.Payload.Body["tenant_uuid"] != tenant || in.Payload.Body["operation"] != "resolve_topics" {
					t.Error("CREDENTIAL_TENANT_NOT_USED")
				}
				json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "result": map[string]any{"contract_version": "event_fabric.service.v1", "tenant_uuid": tenant, "plugin_id": "com.example.fixture", "topic_bindings": map[string]string{"fixture.changed": tenant + ".com.example.fixture.events.changed"}}}})
			}))
			defer server.Close()
			base := server.URL
			if profile.prefixedBase {
				base += profile.prefix
			}
			cfg := &config.Config{Gateway: &config.GatewayConfig{BaseURL: base, APIPrefix: profile.prefix, AuthScheme: "apikey", APIKey: "fixture-key"}, EventBridge: &config.EventBridgeConfig{Enabled: true, SourcePlugin: "com.example.fixture", PublishTopicKeys: []string{"fixture.changed"}}}
			mode := fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeDelegated, ProxyEnabled: true}
			selected := resolveTaskBusRuntime(cfg, mode, nil, nil, nil, nil)
			emitter, runtime, err := newEventBridgeEmitter(cfg, mode, selected)
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Close()
			if emitter == nil || reads != 1 || resolves != 1 || cfg.EventBridge.TenantUUID != "" {
				t.Fatal("CREDENTIAL_SCOPE_RESOLUTION_FAILED")
			}
		})
	}
}

func TestHostEventRejectsConflictingTenantConfiguration(t *testing.T) {
	cfg := &config.Config{EventBridge: &config.EventBridgeConfig{TenantUUID: "22222222-2222-4222-8222-222222222222"}, GRPCUpstream: &config.GRPCUpstream{TenantUUID: "33333333-3333-4333-8333-333333333333"}}
	_, err := newHostTaskBusProvider(cfg, fwprovider.RuntimeMode{ProviderMode: fwprovider.ModeLocal, ProxyEnabled: true}, nil).NewEmitter()
	if err == nil || err.Error() != "EVENT_FABRIC_TENANT_CONFIG_CONFLICT" {
		t.Fatal("CONFLICTING_TENANT_ACCEPTED", err)
	}
}
