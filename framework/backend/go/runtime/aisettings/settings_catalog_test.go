package aisettings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

func TestSettingsCatalogHTTPUsesTenantInvocationsForAPIKeyAndSTS(t *testing.T) {
	for _, scheme := range []string{"apikey", "sts"} {
		t.Run(scheme, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
				expectedAuth := "ApiKey fixture.key"
				if scheme == "sts" {
					expectedAuth = "Bearer fixture.sts"
				}
				require.Equal(t, expectedAuth, r.Header.Get("Authorization"))
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, CapabilityAISettingsCatalogProviders, request["capability_id"])
				require.Equal(t, "core_internal", request["preferred_protocol"])
				payload := request["payload"].(map[string]any)
				require.Len(t, payload, 1)
				require.Equal(t, map[string]any{"operation": "list_settings_providers", "modality": "llm"}, payload["body"])
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"result": map[string]any{"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "providers": []any{}}}}})
			}))
			defer server.Close()
			cfg := gateway.Config{BaseURL: server.URL, APIPrefix: "/api/v1", TenantUUID: "11111111-1111-4111-8111-111111111111", HTTPClient: server.Client(), AuthScheme: "apikey", APIKey: "fixture.key"}
			if scheme == "sts" {
				cfg.AuthScheme = "bearer"
				cfg.APIKey = ""
				cfg.TokenProvider = func(context.Context) (string, error) { return "fixture.sts", nil }
			}
			invoker, err := gateway.NewClient(cfg)
			require.NoError(t, err)
			client, err := NewClient(Config{Invoker: invoker})
			require.NoError(t, err)
			_, err = client.SettingsCatalogProviders(context.Background(), "fixture.request")
			require.NoError(t, err)
			require.Equal(t, 1, calls)
		})
	}
}

func TestSettingsCatalogUsesCoreSettingsOperationsAndPreservesProviderIdentity(t *testing.T) {
	stub := &catalogInvokerStub{payload: map[string]any{
		"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm",
		"providers": []any{map[string]any{"id": "ollama", "name": "Ollama (Local)", "apps": []any{}}},
	}}
	client, err := NewClient(Config{Invoker: stub, TenantUUID: "11111111-1111-4111-8111-111111111111"})
	require.NoError(t, err)
	providers, err := client.SettingsCatalogProviders(context.Background(), "fixture.request")
	require.NoError(t, err)
	require.Equal(t, "Ollama (Local)", providers.Providers[0].Name)
	require.Equal(t, CapabilityAISettingsCatalogProviders, stub.request.CapabilityID)
	require.Equal(t, "core_internal", stub.request.PreferredProtocol)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", stub.request.TenantUUID)
	raw, err := json.Marshal(stub.request.Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"body":{"operation":"list_settings_providers","modality":"llm"}}`, string(raw))

	stub.payload = map[string]any{
		"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "provider": "ollama", "app": "",
		"items": []any{map[string]string{"model_key": "ollama/private-instance:tag", "provider": "ollama", "model": "private-instance:tag", "label": "private-instance:tag"}},
	}
	models, err := client.SettingsCatalogModels(context.Background(), "ollama", "", "fixture.request")
	require.NoError(t, err)
	require.Equal(t, "private-instance:tag", models.Items[0].Model)
	require.Equal(t, "private-instance:tag", models.Items[0].Label)
	require.Equal(t, CapabilityAISettingsCatalogModels, stub.request.CapabilityID)
	raw, err = json.Marshal(stub.request.Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"body":{"operation":"list_settings_models","modality":"llm","provider":"ollama","app":""}}`, string(raw))
}

func TestSettingsCatalogRejectsGenerativeCatalogAndWrongScope(t *testing.T) {
	stub := &catalogInvokerStub{}
	client, err := NewClient(Config{Invoker: stub})
	require.NoError(t, err)
	for _, payload := range []any{
		map[string]any{"items": []any{map[string]string{"model_key": "ollama/static", "label": "fixture.static"}}},
		map[string]any{"contract_version": 2, "providers": []any{map[string]string{"provider": "ollama", "label": "fixture.media"}}},
		map[string]any{"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "providers": []any{map[string]string{"id": "ollama", "name": ""}}},
	} {
		stub.payload = payload
		_, err = client.SettingsCatalogProviders(context.Background(), "")
		require.EqualError(t, err, "AI_SETTINGS_CATALOG_RESPONSE_INVALID")
	}
	for _, provider := range []string{"other-provider", "ollama"} {
		stub.payload = map[string]any{
			"contract_version": 1, "catalog_kind": "ai_settings", "env": "dev", "modality": "llm", "provider": provider, "app": "",
			"items": []any{map[string]string{"model_key": "other-provider/private", "provider": provider, "model": "private", "label": "fixture.model"}},
		}
		_, err = client.SettingsCatalogModels(context.Background(), "ollama", "", "")
		require.EqualError(t, err, "AI_SETTINGS_CATALOG_RESPONSE_INVALID")
	}
}
