package taskbus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
)

func TestHostProviderExplicitAPIKeyAndNoCredentialFallback(t *testing.T) {
	for _, status := range []int{200, 401, 403} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls, tokenCalls := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/v1/admin/runtime/ws-bus/publish" || r.Header.Get("Authorization") != "ApiKey fixture-key" {
					t.Error("TASKBUS_EXPLICIT_API_KEY_NOT_USED")
				}
				var body struct {
					Topic  string `json:"topic"`
					Tenant string `json:"tenant_uuid"`
					Trace  string `json:"trace_id"`
				}
				if json.NewDecoder(r.Body).Decode(&body) != nil || body.Topic != "fixture.event.v1" || body.Tenant != "00000000-0000-0000-0000-000000000001" || body.Trace != "fixture-trace" {
					t.Error("TASKBUS_EVENT_IDENTITY_LOST")
				}
				w.WriteHeader(status)
				if status == 200 {
					_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
				}
			}))
			defer server.Close()
			p := NewHostProvider(HostProviderConfig{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture-key", TokenProvider: func(context.Context) (string, error) { tokenCalls++; return "fixture-sts", nil }})
			emitter, err := p.NewEmitter()
			if err != nil {
				t.Fatal(err)
			}
			err = emitter.Emit(middleware.WithBearerToken(context.Background(), "fixture-user-token"), event.Event{Topic: "fixture.event.v1", Meta: event.Meta{TenantUUID: "00000000-0000-0000-0000-000000000001", RequestID: "fixture-request", TraceID: "fixture-trace"}, Payload: json.RawMessage(`{}`)})
			if (err == nil) != (status == 200) || calls != 1 || tokenCalls != 0 {
				t.Fatal("TASKBUS_AUTH_FAILED_OR_FELL_BACK", err, calls, tokenCalls)
			}
		})
	}
	p := NewHostProvider(HostProviderConfig{BaseURL: "http://core.invalid", AuthScheme: "apikey", TokenProvider: func(context.Context) (string, error) { t.Fatal("TASKBUS_MISSING_KEY_USED_STS"); return "", nil }})
	if _, err := p.NewEmitter(); err == nil {
		t.Fatal("TASKBUS_MISSING_API_KEY_ACCEPTED")
	}
}

func TestHostProviderFromAppPreservesAPIKeyConfiguration(t *testing.T) {
	p := NewHostProviderFromApp(&bootstrap.App{Config: &bootstrap.Config{Gateway: bootstrap.GatewayConfig{BaseURL: "http://core.invalid", AuthScheme: "apikey", APIKey: "fixture-key"}}}, "fixture-plugin", "v1")
	if p.cfg.AuthScheme != "apikey" || p.cfg.APIKey != "fixture-key" {
		t.Fatal("APP_TASKBUS_CREDENTIAL_DROPPED")
	}
	if _, err := p.NewEmitter(); err != nil {
		t.Fatal(err)
	}
}
