package runtimeexample

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLocalConnectionProbesNeverGenerateAndBoundResponses(t *testing.T) {
	for _, item := range []struct {
		name   string
		status int
		body   string
		want   int
		reason string
	}{
		{"empty_models", 200, `{"models":[]}`, 0, ""},
		{"auth", 401, `{}`, 422, "AI_CONNECTION_AUTH_FAILED"},
		{"redirect", 302, `{}`, 502, "AI_CONNECTION_REDIRECT_REJECTED"},
		{"unsupported", 404, `{}`, 501, "AI_CONNECTION_PROBE_UNSUPPORTED"},
		{"rate_limit", 429, `{}`, 429, "AI_CONNECTION_RATE_LIMITED"},
		{"upstream", 500, `{}`, 502, "AI_CONNECTION_UPSTREAM_FAILED"},
		{"null_models", 200, `{"models":null}`, 502, "AI_CONNECTION_RESPONSE_INVALID"},
		{"missing_models", 200, `{}`, 502, "AI_CONNECTION_RESPONSE_INVALID"},
		{"oversized", 200, strings.Repeat("x", (2<<20)+1), 502, "AI_CONNECTION_RESPONSE_INVALID"},
	} {
		t.Run(item.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/api/tags", r.URL.Path)
				w.Header().Set("Location", "/must-not-follow")
				w.WriteHeader(item.status)
				_, _ = w.Write([]byte(item.body))
			}))
			defer server.Close()
			ai, err := NewLocalAI(nil, nil)
			require.NoError(t, err)
			out, err := ai.ProbeProviderConnection(context.Background(), "ollama", server.URL+"/v1")
			if item.want == 0 {
				require.NoError(t, err)
				require.True(t, out.EndpointReachable)
				require.False(t, out.ModelAccessVerified)
				require.False(t, out.GenerationSubmitted)
			} else {
				var typed *powerxai.HTTPError
				require.ErrorAs(t, err, &typed)
				require.Equal(t, item.want, typed.StatusCode)
				require.Equal(t, item.reason, typed.ReasonCode)
				require.Equal(t, false, typed.Details["generation_submitted"])
			}
			require.Equal(t, 1, calls)
		})
	}
}
func TestLocalProfileConnectionTestUsesOnlyReadOnlyProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/tags", r.URL.Path)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()
	ai, err := NewLocalAI(nil, nil)
	require.NoError(t, err)
	require.NoError(t, ai.TestConnectionWithProfile(context.Background(), config.LocalAIModel{Provider: "ollama", Endpoint: server.URL}))
}
