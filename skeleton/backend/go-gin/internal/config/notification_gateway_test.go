package config

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
)

type notificationGatewayTransport func(*http.Request) (*http.Response, error)

func (f notificationGatewayTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoadedGatewayPrefixReachesNotificationClient(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("POWERX_PROVIDER_MODE", "local")
	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("PX_GATEWAY_BASE_URL", "https://core.example")
	for _, tc := range []struct{ input, prefix string }{
		{"", "/api/v1"}, {"/api/v2", "/api/v2"}, {"/core/custom", "/core/custom"}, {"/", "/"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			t.Setenv("PX_GATEWAY_API_PREFIX", tc.input)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Gateway.APIPrefix != tc.prefix {
				t.Fatalf("gateway prefix=%q, expected=%q", cfg.Gateway.APIPrefix, tc.prefix)
			}
			called := false
			client, err := notifications.NewClient(notifications.Config{BaseURL: cfg.Gateway.BaseURL, APIPrefix: cfg.Gateway.APIPrefix, BearerToken: "fixture"}, &http.Client{Transport: notificationGatewayTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				if r.URL.Path != strings.TrimRight(tc.prefix, "/")+"/notifications" {
					t.Fatalf("notification path=%q", r.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":{"id":"11111111-1111-4111-8111-111111111111"}}`))}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Create(context.Background(), notifications.CreateInput{Title: "fixture_title", Content: "fixture_content"}); err != nil || !called {
				t.Fatalf("notification call=%t, error=%v", called, err)
			}
		})
	}
}
