package taskbus

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostProviderExplicitAPIKeyAndNoCredentialFallback(t *testing.T) {
	for _, status := range []int{200, 401, 403} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, tokens := 0, 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
				require.Equal(t, "ApiKey fixture-key", r.Header.Get("Authorization"))
				if status != 200 {
					w.WriteHeader(status)
					return
				}
				hostSuccess(w)
			}))
			defer s.Close()
			cfg := fixtureHostConfig(s.URL)
			cfg.AuthScheme = "apikey"
			cfg.APIKey = "fixture-key"
			cfg.TokenProvider = func(context.Context) (string, error) { tokens++; return "fixture-sts", nil }
			emitter, err := NewHostProvider(cfg).NewEmitter()
			require.NoError(t, err)
			err = emitter.Emit(context.Background(), hostFixtureEvent())
			require.Equal(t, status == 200, err == nil)
			require.Equal(t, 1, calls)
			require.Zero(t, tokens)
		})
	}
	cfg := fixtureHostConfig("http://core.invalid")
	cfg.AuthScheme = "apikey"
	cfg.TokenProvider = func(context.Context) (string, error) { t.Error("EVENT_FABRIC_CREDENTIAL_FALLBACK"); return "", nil }
	_, err := NewHostProvider(cfg).NewEmitter()
	require.Error(t, err)
}
func TestHostProviderFromAppPreservesAPIKeyConfiguration(t *testing.T) {
	p := NewHostProviderFromApp(&bootstrap.App{Config: &bootstrap.Config{Gateway: bootstrap.GatewayConfig{BaseURL: "http://core.invalid", AuthScheme: "apikey", APIKey: "fixture-key"}}}, "fixture-plugin", "1")
	require.Equal(t, "apikey", p.cfg.AuthScheme)
	require.Equal(t, "fixture-key", p.cfg.APIKey)
	_, err := p.NewEmitter()
	require.Error(t, err)
}
