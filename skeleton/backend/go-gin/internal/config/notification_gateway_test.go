package config

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
)

type notificationGatewayTransport func(*http.Request) (*http.Response, error)

func (f notificationGatewayTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNotificationChannelRequiresExplicitCoreHostConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "")
	t.Setenv("POWERX_PROVIDER_MODE", "local")
	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("PX_GATEWAY_BASE_URL", "https://core.example")
	t.Setenv("PX_GATEWAY_AUTH_SCHEME", "apikey")
	t.Setenv("PX_GATEWAY_API_KEY", "controlled-api-key")
	t.Setenv("PX_NOTIFICATION_CHANNEL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NotificationChannel() != "" {
		t.Fatal("API_KEY_SILENTLY_REPLACED_LOCAL")
	}
	t.Setenv("PX_NOTIFICATION_CHANNEL", "core_host")
	cfg, err = Load()
	if err != nil || cfg.NotificationChannel() != "core_host" {
		t.Fatal("EXPLICIT_CORE_HOST_NOT_SELECTED")
	}
	t.Setenv("PX_NOTIFICATION_CHANNEL", "unknown")
	if _, err := Load(); err == nil {
		t.Fatal("UNKNOWN_NOTIFICATION_CHANNEL_ACCEPTED")
	}
	t.Setenv("PX_NOTIFICATION_CHANNEL", "core_host")
	t.Setenv("PX_GATEWAY_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("CORE_HOST_WITHOUT_CREDENTIAL_ACCEPTED")
	}
}

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

func TestVersionedReceiptConflictKeepsBatchAndCredentialChannel(t *testing.T) {
	for _, channel := range []string{"apikey", "sts"} {
		t.Run(channel, func(t *testing.T) {
			ids := []string{"33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"}
			input := notifications.ReadReceiptInput{MemberUUID: "22222222-2222-4222-8222-222222222222", NotificationUUIDs: ids, Revisions: map[string]int64{ids[0]: 2, ids[1]: 1}}
			calls := 0
			httpClient := &http.Client{Transport: notificationGatewayTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				var sent notifications.ReadReceiptInput
				if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
					t.Fatal(err)
				}
				want := "Bearer controlled-sts"
				if channel == "apikey" {
					want = "ApiKey controlled-key"
				}
				if req.Header.Get("Authorization") != want || len(sent.NotificationUUIDs) != 2 || sent.Revisions[ids[0]] != 2 || sent.Revisions[ids[1]] != 1 {
					t.Fatal("BATCH_OR_CREDENTIAL_CHANGED")
				}
				return &http.Response{StatusCode: 409, Body: io.NopCloser(strings.NewReader(`{"error_code":"NOTIFICATION_REVISION_CONFLICT","trace_id":"batch-trace"}`))}, nil
			})}
			var client *notifications.Client
			var err error
			if channel == "apikey" {
				client, err = notifications.NewClient(notifications.Config{BaseURL: "https://core.test", AuthScheme: "apikey", APIKey: "controlled-key"}, httpClient)
			} else {
				client, err = notifications.NewClientWithTokenProvider(notifications.Config{BaseURL: "https://core.test"}, notifications.TokenProviderFunc(func(context.Context) (string, error) { return "controlled-sts", nil }), httpClient)
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Acknowledge(context.Background(), input)
			var upstream *notifications.HTTPError
			if !errors.As(err, &upstream) || upstream.StatusCode != 409 || upstream.ReasonCode != "NOTIFICATION_REVISION_CONFLICT" || upstream.TraceID != "batch-trace" || calls != 1 {
				t.Fatal("BATCH_CONFLICT_NOT_PROPAGATED_ATOMICALLY")
			}
		})
	}
}
