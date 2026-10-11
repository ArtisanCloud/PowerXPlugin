package runtimeidentity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

func TestTypedAPIKeyAndSTS(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/tenant/invocations" || r.Method != "POST" || r.Header.Get("Authorization") != scheme+" test-token" {
					t.Error("INVALID_SERVICE_BINDING")
				}
				var in capability.InvokeInput
				if json.NewDecoder(r.Body).Decode(&in) != nil {
					t.Fatal("INVALID_REQUEST")
				}
				if in.CapabilityID != CapabilityID || in.PreferredProtocol != "core_internal" || len(in.Payload) != 3 {
					t.Error("INVALID_TYPED_PAYLOAD")
				}
				body := in.Payload["body"].(map[string]any)
				if len(body) != 2 || body["plugin_id"] != "plugin.test" || body["operation"] != "get" {
					t.Error("INVALID_OPERATION")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"trace_id": "core-trace", "payload": map[string]any{"item": Identity{RuntimeMode: "powerx", DeploymentEnv: "dev", CoreVersion: "v1", PluginID: "plugin.test", RuntimePluginID: "plugin.test", PluginVersion: "2.3", PluginVersionSource: "registry", PluginState: "stopped"}}}})
			}))
			defer srv.Close()
			var registry *capability.Client
			var err error
			if scheme == "ApiKey" {
				registry, err = capability.NewClient(capability.Config{BaseURL: srv.URL, AuthScheme: "apikey", APIKey: "test-token"}, nil)
			} else {
				registry, err = capability.NewClientWithTokenProvider(capability.Config{BaseURL: srv.URL}, capability.TokenProviderFunc(func(context.Context) (string, error) { return "test-token", nil }), nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			item, err := NewClient(registry).ReadRuntimeIdentity(context.Background(), "plugin.test")
			if err != nil || item.PluginState != "stopped" || item.TraceID != "core-trace" {
				t.Fatalf("INVALID_IDENTITY_RESULT: %v", err)
			}
		})
	}
}

func TestErrorStatusAndTraceArePreserved(t *testing.T) {
	for _, status := range []int{401, 403, 404, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Trace-ID", "trace-test")
			w.Header().Set("X-Request-ID", "request-test")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"reason_code":"RUNTIME_IDENTITY_FORBIDDEN"}`))
		}))
		registry, err := capability.NewClient(capability.Config{BaseURL: srv.URL, AuthScheme: "apikey", APIKey: "test-token"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = NewClient(registry).ReadRuntimeIdentity(context.Background(), "plugin.test")
		var remote *Error
		if !errors.As(err, &remote) || remote.StatusCode != status || remote.ReasonCode != "RUNTIME_IDENTITY_FORBIDDEN" || remote.TraceID != "trace-test" || remote.RequestID != "request-test" {
			t.Fatalf("ERROR_CONTRACT_LOST: %v", err)
		}
		srv.Close()
	}
}

func TestLiveHostRuntimeIdentity(t *testing.T) {
	key := os.Getenv("RUNTIME_IDENTITY_TEST_API_KEY")
	if key == "" {
		t.Skip("LIVE_HOST_CREDENTIAL_REQUIRED")
	}
	client, err := capability.NewClient(capability.Config{BaseURL: os.Getenv("RUNTIME_IDENTITY_TEST_BASE_URL"), AuthScheme: "apikey", APIKey: key}, nil)
	if err != nil {
		t.Fatal("LIVE_HOST_CONFIG_INVALID")
	}
	item, err := NewClient(client).ReadRuntimeIdentity(context.Background(), "com.powerx.plugins.ai-craft")
	if err == nil {
		t.Logf("LIVE_RUNTIME_IDENTITY_OK source=%s state=%s", item.PluginVersionSource, item.PluginState)
		return
	}
	var remote *Error
	if !errors.As(err, &remote) {
		t.Fatal("LIVE_ERROR_CONTRACT_INVALID")
	}
	t.Logf("LIVE_RUNTIME_IDENTITY status=%d reason=%s trace_id=%s request_id=%s", remote.StatusCode, remote.ReasonCode, remote.TraceID, remote.RequestID)
	if remote.StatusCode != 403 || remote.TraceID == "" || remote.ReasonCode == "CAPABILITY_UPSTREAM_DEPENDENCY" {
		t.Fatal("LIVE_EXPECTED_PRECISE_GRANT_REJECTION")
	}
}
