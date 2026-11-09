package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestHostTaskBusPreservesExplicitAPIKey(t *testing.T) {
	for _, mode := range []string{"local", "delegated"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "ApiKey fixture-key" {
					t.Error("TASKBUS_AUTH_SCHEME_CHANGED")
				}
				_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
			}))
			defer server.Close()
			cfg := &config.Config{Gateway: &config.GatewayConfig{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture-key"}, Context: &config.ContextConfig{ProviderMode: mode}}
			emitter, err := newHostTaskBusProvider(cfg, nil).NewEmitter()
			if err != nil {
				t.Fatal(err)
			}
			err = emitter.Emit(context.Background(), event.Event{Topic: "fixture.event.v1", Meta: event.Meta{TenantUUID: "00000000-0000-0000-0000-000000000001", RequestID: "fixture-request", TraceID: "fixture-trace"}, Payload: json.RawMessage(`{}`)})
			if err != nil || calls != 1 {
				t.Fatal("TASKBUS_EXPLICIT_API_KEY_FAILED", err, calls)
			}
			cfg.Gateway.APIKey = ""
			if _, err := newHostTaskBusProvider(cfg, nil).NewEmitter(); err == nil {
				t.Fatal("TASKBUS_MISSING_API_KEY_ACCEPTED")
			}
		})
	}
}

func TestValidateTaskBusProviderConflict(t *testing.T) {
	tests := []struct {
		name           string
		proxy          string
		provider       string
		expectProvider string
	}{
		{name: "standalone keeps redis", proxy: "0", provider: "redis", expectProvider: "redis"},
		{name: "delegated keeps host", proxy: "1", provider: "host", expectProvider: "host"},
		{name: "standalone keeps host override", proxy: "0", provider: "host", expectProvider: "host"},
		{name: "delegated overrides redis to host", proxy: "1", provider: "redis", expectProvider: "host"},
		{name: "empty proxy uses config value", proxy: "", provider: "redis", expectProvider: "redis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				EventBridge: &config.EventBridgeConfig{TaskBusProvider: tt.provider},
			}
			err := validateTaskBusProviderConflict(cfg, tt.proxy)
			if err != nil {
				t.Fatalf("期望校验成功，实际失败: %v", err)
			}
			if got := cfg.EventBridge.TaskBusProvider; got != tt.expectProvider {
				t.Fatalf("provider mismatch: got=%s want=%s", got, tt.expectProvider)
			}
		})
	}
}
