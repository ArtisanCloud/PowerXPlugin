package aisettings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/stretchr/testify/require"
)

func connectionFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("testdata/ai-connection-test.v1.json")
	require.NoError(t, err)
	var fixture map[string]any
	require.NoError(t, json.Unmarshal(raw, &fixture))
	return fixture
}
func connectionData(result any) map[string]any {
	return map[string]any{"protocol_used": "core_internal", "status": "completed", "fallback_used": false, "payload": map[string]any{"result": result}}
}

type connectionInvoker struct {
	data    map[string]any
	request gateway.InvokeRequest
	calls   int
}

func (i *connectionInvoker) Invoke(_ context.Context, in gateway.InvokeRequest) (*gateway.Response, error) {
	i.request = in
	i.calls++
	return &gateway.Response{Data: i.data}, nil
}

func TestConnectionUsesSharedFixtureForBothCredentialsAndConfiguredPrefix(t *testing.T) {
	for _, scheme := range []string{"apikey", "sts"} {
		for _, prefix := range []string{"/api/v1", "/custom/v2"} {
			t.Run(scheme+prefix, func(t *testing.T) {
				fixture := connectionFixture(t)
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					require.Equal(t, http.MethodPost, r.Method)
					require.Equal(t, prefix+"/tenant/invocations", r.URL.Path)
					auth := "ApiKey fixture.key"
					if scheme == "sts" {
						auth = "Bearer fixture.sts"
					}
					require.Equal(t, auth, r.Header.Get("Authorization"))
					var request map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Equal(t, fixture["capability_id"], request["capability_id"])
					require.Equal(t, fixture["preferred_protocol"], request["preferred_protocol"])
					require.Equal(t, fixture["payload"], request["payload"])
					_ = json.NewEncoder(w).Encode(map[string]any{"data": connectionData(fixture["expected_result"])})
				}))
				defer server.Close()
				cfg := gateway.Config{BaseURL: server.URL, APIPrefix: prefix, AuthScheme: "apikey", APIKey: "fixture.key", HTTPClient: server.Client()}
				if scheme == "sts" {
					cfg.AuthScheme = "bearer"
					cfg.APIKey = ""
					cfg.TokenProvider = func(context.Context) (string, error) { return "fixture.sts", nil }
				}
				invoker, err := gateway.NewClient(cfg)
				require.NoError(t, err)
				client, err := NewClient(Config{Invoker: invoker})
				require.NoError(t, err)
				out, err := client.TestConnection(context.Background(), ConnectionTestInput{Modality: "llm", Provider: "ollama"}, "fixture.request")
				require.NoError(t, err)
				require.Equal(t, "dev", out.Environment)
				require.False(t, out.Verification.GenerationSubmitted)
				require.False(t, out.Verification.ModelAccessVerified)
				require.False(t, out.Verification.APIKeyVerified)
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

func TestConnectionRejectsMalformedOrFallbackResponses(t *testing.T) {
	for _, bad := range []string{"generation", "model_access", "missing_generation", "missing_api_key", "missing_model_access", "wrong_provider", "authentication", "protocol", "fallback", "missing_fallback", "incomplete", "unknown_field"} {
		t.Run(bad, func(t *testing.T) {
			result := connectionFixture(t)["expected_result"].(map[string]any)
			verification := result["verification"].(map[string]any)
			data := connectionData(result)
			switch bad {
			case "generation":
				verification["generation_submitted"] = true
			case "model_access":
				verification["model_access_verified"] = true
			case "missing_generation":
				delete(verification, "generation_submitted")
			case "missing_api_key":
				delete(verification, "api_key_verified")
			case "missing_model_access":
				delete(verification, "model_access_verified")
			case "wrong_provider":
				verification["provider"] = "another"
			case "authentication":
				verification["authentication"] = "api_key"
			case "protocol":
				data["protocol_used"] = "rest"
			case "fallback":
				data["fallback_used"] = true
			case "missing_fallback":
				delete(data, "fallback_used")
			case "incomplete":
				data["status"] = "running"
			case "unknown_field":
				result["prompt"] = "fixture.prompt"
			}
			i := &connectionInvoker{data: data}
			client, err := NewClient(Config{Invoker: i})
			require.NoError(t, err)
			_, err = client.TestConnection(context.Background(), ConnectionTestInput{Modality: "llm", Provider: "ollama"}, "")
			var typed *powerxai.HTTPError
			require.ErrorAs(t, err, &typed)
			require.Equal(t, 502, typed.StatusCode)
			require.Equal(t, "AI_CONNECTION_CONTRACT_INVALID", typed.ReasonCode)
			require.Equal(t, 1, i.calls)
		})
	}
}

func TestConnectionPreservesErrorsWithoutRetryOrCredentialFallback(t *testing.T) {
	for _, scheme := range []string{"apikey", "sts"} {
		for _, status := range []int{401, 403, 409, 422, 429, 503} {
			t.Run(scheme+http.StatusText(status), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
					w.Header().Set("X-Trace-ID", "fixture.trace")
					w.Header().Set("X-Request-ID", "fixture.request")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"error_code":"fixture.error","details":{"reason_code":"AI_SETTINGS_PROVIDER_NOT_CONFIGURED","generation_submitted":false}}`))
				}))
				defer server.Close()
				cfg := gateway.Config{BaseURL: server.URL, APIPrefix: "/api/v1", AuthScheme: "apikey", APIKey: "fixture.key", HTTPClient: server.Client()}
				if scheme == "sts" {
					cfg.AuthScheme = "bearer"
					cfg.APIKey = ""
					cfg.TokenProvider = func(context.Context) (string, error) { return "fixture.sts", nil }
				}
				i, err := gateway.NewClient(cfg)
				require.NoError(t, err)
				c, err := NewClient(Config{Invoker: i})
				require.NoError(t, err)
				_, err = c.TestConnection(context.Background(), ConnectionTestInput{Modality: "llm", Provider: "ollama"}, "")
				var typed *powerxai.HTTPError
				require.ErrorAs(t, err, &typed)
				require.Equal(t, status, typed.StatusCode)
				require.Equal(t, "fixture.error", typed.ErrorCode)
				require.Equal(t, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED", typed.ReasonCode)
				require.Equal(t, "fixture.trace", typed.TraceID)
				require.Equal(t, "fixture.request", typed.RequestID)
				require.Equal(t, false, typed.Details["generation_submitted"])
				require.Contains(t, typed.Body, "fixture.error")
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

func TestConnectionValidatesInputBeforeCallingGateway(t *testing.T) {
	i := &connectionInvoker{}
	c, err := NewClient(Config{Invoker: i})
	require.NoError(t, err)
	for _, in := range []ConnectionTestInput{{Modality: "llm"}, {Modality: "unknown", Provider: "ollama"}, {Modality: "llm", Provider: " ollama"}} {
		_, err = c.TestConnection(context.Background(), in, "")
		require.Error(t, err)
	}
	require.Zero(t, i.calls)
	result := connectionFixture(t)["expected_result"].(map[string]any)
	result["verification"] = map[string]any{"provider": "openai", "probe": "models", "endpoint_reachable": true, "api_key_verified": true, "authentication": "api_key", "model_access_verified": false, "generation_submitted": false}
	i.data = connectionData(result)
	_, err = c.TestConnection(context.Background(), ConnectionTestInput{Modality: "llm", Provider: "openai", Model: "native:model"}, "")
	require.NoError(t, err)
	raw, _ := json.Marshal(i.request.Payload)
	require.True(t, strings.Contains(string(raw), `"model":"native:model"`))
	require.NotContains(t, string(raw), "model_key")
}
