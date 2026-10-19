package delegated

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func baselineClientFixture(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../contracts/testdata/authorization_baseline_result.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func baselineClientQuery() contracts.AuthorizationBaselineQuery {
	return contracts.AuthorizationBaselineQuery{TenantUUID: "20000000-0000-4000-8000-000000000001", MemberUUIDs: []string{"10000000-0000-4000-8000-000000000001"}, PermissionCodes: []string{"orders:read", "cost:read", "scope:read"}}
}

func TestAuthorizationBaselineServiceCredentialsAndBinding(t *testing.T) {
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodPost || request.URL.Path != "/api/v1/tenant/invocations" || request.Header.Get("Authorization") != scheme+" server-owned" || request.Header.Get("X-Trace-ID") != "request-trace" {
					t.Error(request.Method, request.URL, request.Header)
				}
				raw, _ := io.ReadAll(request.Body)
				var invocation map[string]any
				if err := json.Unmarshal(raw, &invocation); err != nil {
					t.Error(err)
					return
				}
				body := invocation["payload"].(map[string]any)["body"].(map[string]any)
				query := body["request"].(map[string]any)
				if invocation["capability_id"] != contracts.AuthorizationBaselineCapability || invocation["preferred_protocol"] != "core_internal" || body["operation"] != contracts.AuthorizationBaselineOperation || query["schema"] != contracts.AuthorizationBaselineQuerySchema || len(query) != 3 || strings.Contains(string(raw), "tenant_uuid") || strings.Contains(string(raw), "plugin_id") {
					t.Error(string(raw))
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"result": map[string]any{"result": baselineClientFixture(t)}}})
			}))
			defer server.Close()
			config := CoreClientConfig{BaseURL: server.URL, PluginID: "com.powerx.fixture", Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "server-owned", nil })}
			var client *CoreClient
			var err error
			if scheme == "ApiKey" {
				client, err = NewCoreClientWithAPIKey(config, "server-owned")
			} else {
				client, err = NewCoreClient(config)
			}
			if err != nil {
				t.Fatal(err)
			}
			bundle, err := NewBundle(client)
			if err != nil || bundle.AuthorizationBaselines == nil {
				t.Fatal(err)
			}
			result, err := bundle.AuthorizationBaselines.ReadMemberBaselines(WithRequestMetadata(context.Background(), RequestMetadata{TraceID: "request-trace"}), baselineClientQuery())
			if err != nil || len(result.Items) != 3 {
				t.Fatal(result, err)
			}
		})
	}
}

func TestAuthorizationBaselineErrorsPreserveStatusCodeTraceAndFailClosed(t *testing.T) {
	for _, status := range []int{400, 401, 403, 409, 503, 200} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Trace-ID", "core-trace")
				w.Header().Set("X-Request-ID", "core-request")
				w.WriteHeader(status)
				if status == 200 {
					malformed := strings.Replace(string(baselineClientFixture(t)), "com.powerx.fixture", "com.powerx.other", 1)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"result": map[string]any{"result": json.RawMessage(malformed)}}})
				} else {
					_ = json.NewEncoder(w).Encode(map[string]any{"code": status, "error_code": "IAM_BASELINE_SERVICE_FORBIDDEN"})
				}
			}))
			defer server.Close()
			client, _ := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL, PluginID: "com.powerx.fixture"}, "server-owned")
			result, err := client.ReadMemberBaselines(context.Background(), baselineClientQuery())
			var upstream *AuthorizationBaselineRequestError
			if result != nil || !errors.As(err, &upstream) || upstream.Status != status || upstream.TraceID != "core-trace" || upstream.RequestID != "core-request" {
				t.Fatal(result, err)
			}
			code := "IAM_BASELINE_SERVICE_FORBIDDEN"
			mapped := status
			if status == 200 {
				code, mapped = iamerrors.CodeBaselineResponseInvalid, 502
			}
			if iamerrors.CodeOf(err) != code || iamerrors.StatusCode(err) != mapped {
				t.Fatal(err, iamerrors.CodeOf(err), iamerrors.StatusCode(err))
			}
		})
	}
}

func TestAuthorizationBaselineCancellationTimeoutAndInvalidQuery(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	client, _ := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL, PluginID: "com.powerx.fixture", Timeout: 40 * time.Millisecond}, "server-owned")
	if _, err := client.ReadMemberBaselines(context.Background(), baselineClientQuery()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ReadMemberBaselines(ctx, baselineClientQuery()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := client.ReadMemberBaselines(context.Background(), contracts.AuthorizationBaselineQuery{}); !iamerrors.IsCode(err, iamerrors.CodeBaselineInvalidArgument) {
		t.Fatal(err)
	}
}
