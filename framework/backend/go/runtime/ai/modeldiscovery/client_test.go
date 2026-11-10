package modeldiscovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/stretchr/testify/require"
)

func TestProviderCatalogProtocolsPathsAuthenticationAndPagination(t *testing.T) {
	for _, tc := range []struct{ provider, base, path, protocol string }{
		{"openai", "/v1", "/v1/models", "openai"}, {"openrouter", "/api/v1", "/api/v1/models", "openai"}, {"deepseek", "", "/models", "openai"}, {"moonshot", "/v1", "/v1/models", "openai"}, {"huggingface", "/v1", "/v1/models", "openai"}, {"openai_compatible", "/proxy/v1", "/proxy/v1/models", "openai"},
		{"anthropic", "/proxy/v1", "/proxy/v1/models", "anthropic"}, {"google", "/v1beta", "/v1beta/models", "gemini"}, {"qwen-cn", "/compatible-mode/v1", "/api/v1/models", "dashscope"}, {"qwen-intl", "/gateway/compatible-mode/v1", "/gateway/api/v1/models", "dashscope"}, {"ollama", "/proxy/v1", "/proxy/api/tags", "ollama"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Empty(t, r.URL.Query().Get("key"))
				require.NotContains(t, r.URL.String(), "fixture-key")
				switch tc.protocol {
				case "anthropic":
					require.Equal(t, "fixture-key", r.Header.Get("x-api-key"))
					require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
					require.Empty(t, r.Header.Get("Authorization"))
				case "gemini":
					require.Equal(t, "fixture-key", r.Header.Get("x-goog-api-key"))
				case "ollama":
					require.Empty(t, r.Header.Get("Authorization"))
				default:
					require.Equal(t, "Bearer fixture-key", r.Header.Get("Authorization"))
				}
				if tc.protocol == "ollama" && r.Method == http.MethodPost {
					require.Equal(t, "/proxy/api/show", r.URL.Path)
					var body map[string]string
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "custom:14b", body["model"])
					_, _ = w.Write([]byte(`{"capabilities":["completion","vision"]}`))
					return
				}
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, tc.path, r.URL.Path)
				response := `{"data":[{"id":"new/model:tag","architecture":{"output_modalities":["text"]}}]}`
				switch tc.protocol {
				case "anthropic":
					if calls == 1 {
						response = `{"data":[{"id":"claude-new-1"}],"has_more":true,"last_id":"cursor-1"}`
					} else {
						require.Equal(t, "cursor-1", r.URL.Query().Get("after_id"))
						response = `{"data":[{"id":"claude-new-2"}],"has_more":false}`
					}
				case "gemini":
					if calls == 1 {
						response = `{"models":[{"name":"models/gemini-new","displayName":"New Gemini","supportedGenerationMethods":["generateContent"]}],"nextPageToken":"page-2"}`
					} else {
						require.Equal(t, "page-2", r.URL.Query().Get("pageToken"))
						response = `{"models":[{"name":"models/embed-new","supportedGenerationMethods":["embedContent"]}]}`
					}
				case "dashscope":
					if calls == 1 {
						require.Equal(t, "1", r.URL.Query().Get("page_no"))
						response = `{"success":true,"output":{"total":2,"models":[{"model":"qwen-new","capabilities":["TG"]}]}}`
					} else {
						require.Equal(t, "2", r.URL.Query().Get("page_no"))
						response = `{"success":true,"output":{"total":2,"models":[{"model":"qwen-image-new","capabilities":["IG"]}]}}`
					}
				case "ollama":
					response = `{"models":[{"name":"custom:14b"}]}`
				}
				if tc.provider == "openrouter" {
					require.Equal(t, "all", r.URL.Query().Get("output_modalities"))
				}
				_, _ = w.Write([]byte(response))
			}))
			defer srv.Close()
			key := "fixture-key"
			if tc.protocol == "ollama" {
				key = ""
			}
			result, err := Fetch(context.Background(), Input{Provider: tc.provider, Endpoint: srv.URL + tc.base, APIKey: key})
			require.NoError(t, err)
			require.Equal(t, tc.protocol, result.Protocol)
			require.Equal(t, srv.URL+tc.path, result.Source)
			require.Contains(t, result.Models[0].Modalities, "llm")
			if tc.protocol == "gemini" || tc.protocol == "dashscope" || tc.protocol == "anthropic" {
				require.Len(t, result.Models, 2)
				require.Equal(t, 2, calls)
			}
			if tc.protocol == "dashscope" {
				require.Empty(t, result.Models[1].Modalities, "image contract must not be invented from an online ID")
			}
			if tc.protocol == "ollama" {
				require.Equal(t, "custom:14b", result.Models[0].ID)
				require.Equal(t, 2, calls)
			}
		})
	}
}

func TestDiscoveryFailsExplicitlyWithoutLeakingUpstreamBodiesOrFollowingRedirects(t *testing.T) {
	for _, tc := range []struct {
		status     int
		body, code string
	}{
		{401, "leaked-fixture-key", "AI_MODEL_SYNC_AUTH_FAILED"}, {500, "leaked-fixture-key", "AI_MODEL_DISCOVERY_UPSTREAM_FAILED"}, {200, `{"error":"leaked-fixture-key"}`, "AI_MODEL_DISCOVERY_RESPONSE_INVALID"}, {200, `{"data":[{}]}`, "AI_MODEL_DISCOVERY_RESPONSE_INVALID"}, {200, `{"data":[],"has_more":true}`, "AI_MODEL_DISCOVERY_RESPONSE_INVALID"},
	} {
		t.Run(tc.code+tc.body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			result, err := Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key"})
			require.Nil(t, result)
			requireDiscoveryError(t, err, tc.code)
			require.NotContains(t, err.Error(), "leaked-fixture-key")
		})
	}
	redirects := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirects++ }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer srv.Close()
	_, err := Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key"})
	requireDiscoveryError(t, err, "AI_MODEL_DISCOVERY_UPSTREAM_FAILED")
	require.Zero(t, redirects)
	for _, endpoint := range []string{"file:///tmp/models", "https://user:secret@example.invalid", "https://example.invalid?key=secret", "https://example.invalid#fragment"} {
		_, err := Fetch(context.Background(), Input{Provider: "openai", Endpoint: endpoint, APIKey: "fixture-key"})
		requireDiscoveryError(t, err, "AI_MODEL_DISCOVERY_CONFIG_INVALID")
	}
	for _, provider := range []string{"coze", "volcengine", "hunyuan", "hash", "agnes", "baidu", "comfyui", "sentence-transformers"} {
		_, err := Fetch(context.Background(), Input{Provider: provider})
		requireDiscoveryError(t, err, "AI_MODEL_SYNC_UNSUPPORTED")
	}
	_, err = Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL})
	requireDiscoveryError(t, err, "AI_SETTINGS_PROVIDER_NOT_CONFIGURED")
}

func TestDiscoveryEmptyAndUnknownModelsDoNotAcquireInventedCapabilities(t *testing.T) {
	response := `{"data":[]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
	defer srv.Close()
	result, err := Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key"})
	require.NoError(t, err)
	require.Empty(t, result.Models)
	response = `{"data":[{"id":"gpt-brand-new"},{"id":"gpt-brand-new"}]}`
	result, err = Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key"})
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	require.Empty(t, result.Models[0].Modalities)
	response = `{"data":[{"id":"` + strings.Repeat("x", 513) + `"}]}`
	_, err = Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key"})
	requireDiscoveryError(t, err, "AI_MODEL_DISCOVERY_RESPONSE_INVALID")
}

func TestDiscoveryRespectsConfiguredMetadataBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	_, err := Fetch(context.Background(), Input{Provider: "openai", Endpoint: srv.URL, APIKey: "fixture-key", Timeout: 20 * time.Millisecond})
	requireDiscoveryError(t, err, "AI_MODEL_DISCOVERY_TIMEOUT")
}

func requireDiscoveryError(t *testing.T, err error, code string) {
	t.Helper()
	var typed *powerxai.HTTPError
	require.ErrorAs(t, err, &typed)
	require.Equal(t, code, typed.ReasonCode)
	require.Equal(t, code, typed.ErrorCode)
}

func TestOllamaMetadataFailurePreservesExactLocationModelAndTrace(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodGet && r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[{"name":"retired-private:tag"}]}`))
			return
		}
		require.Equal(t, "POST /api/show", r.Method+" "+r.URL.Path)
		w.Header().Set("X-Trace-ID", "fixture.trace")
		w.Header().Set("X-Request-ID", "fixture.request")
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte("fixture.private_upstream_body"))
	}))
	defer server.Close()
	_, err := Fetch(context.Background(), Input{Provider: "ollama", Endpoint: server.URL})
	var typed *powerxai.HTTPError
	require.ErrorAs(t, err, &typed)
	require.Equal(t, 502, typed.StatusCode)
	require.Equal(t, "fixture.trace", typed.TraceID)
	require.Equal(t, "fixture.request", typed.RequestID)
	require.Empty(t, typed.Body)
	require.Equal(t, server.URL+"/api/show", typed.Details["source"])
	require.Equal(t, "retired-private:tag", typed.Details["model"])
	require.Equal(t, http.StatusGone, typed.Details["upstream_status"])
	require.Equal(t, false, typed.Details["generation_submitted"])
	require.Equal(t, 2, calls)
}
