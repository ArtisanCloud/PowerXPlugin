package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamdelegated "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	sharedapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestPluginAuthorizationProductionProxyUsesStartupSelectedClient(t *testing.T) {
	query := fwiamcontracts.AuthorizationRequest{TenantUUID: "11111111-1111-4111-8111-111111111111", MemberUUID: "22222222-2222-4222-8222-222222222222", UserUUID: "33333333-3333-4333-8333-333333333333", PluginID: "com.example.messaging", PermissionCode: "messages.conversation:reply"}
	for _, scheme := range []string{"ApiKey", "Bearer"} {
		t.Run(scheme, func(t *testing.T) {
			calls := 0
			failure := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/v1/tenant/iam/authorization:check" || r.Header.Get("Authorization") != scheme+" selected-service-credential" {
					t.Error("SELECTED_AUTHORIZATION_CHANNEL_CHANGED")
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["plugin_id"] != query.PluginID || body["permission_code"] != query.PermissionCode || body["tenant_uuid"] != nil {
					t.Error("PLUGIN_PERMISSION_QUERY_LOST", body)
				}
				if failure {
					w.Header().Set("X-Trace-ID", "core-trace")
					w.Header().Set("X-Request-ID", "core-request")
					w.WriteHeader(404)
					_ = json.NewEncoder(w).Encode(map[string]any{"error_code": fwiamerrors.CodePluginPermissionNotRegistered, "reason_code": fwiamerrors.CodePluginPermissionNotRegistered})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"allowed": false, "reason_code": fwiamerrors.CodePermissionDenied, "tenant_uuid": query.TenantUUID, "member_uuid": query.MemberUUID, "user_uuid": query.UserUUID, "plugin_id": query.PluginID, "permission_code": query.PermissionCode, "resource": "conversation", "action": "reply", "trace_id": "core-trace"}})
			}))
			defer server.Close()
			cfg := fwiamdelegated.CoreClientConfig{BaseURL: server.URL, PluginID: sharedapp.PluginID, Tokens: fwiamdelegated.TokenProviderFunc(func(context.Context) (string, error) { return "selected-service-credential", nil })}
			client, err := fwiamdelegated.NewCoreClient(cfg)
			if scheme == "ApiKey" {
				client, err = fwiamdelegated.NewCoreClientWithAPIKey(cfg, "selected-service-credential")
			}
			if err != nil {
				t.Fatal(err)
			}
			// The old proxy has deliberately different credentials: authorization
			// must use the selected typed service client, never this legacy path.
			proxy, err := authproxy.NewDelegatedClient(server.URL, "legacy-not-selected")
			if err != nil {
				t.Fatal(err)
			}
			proxy.SetIdentityResolver(client)
			deps := &sharedapp.Deps{IAMAdapterMode: iamservice.IAMAdapterModeDelegated, AuthProxy: proxy}
			if err := BindFrameworkIAM(deps, fwiamadapters.NewRegistry()); err != nil {
				t.Fatal(err)
			}
			defer deps.WSIdentityLifecycle.Close()
			result, err := deps.IAMAuthzService.Authorize(context.Background(), query)
			if err != nil || result == nil || result.Allowed || result.PluginID != query.PluginID || result.PermissionCode != query.PermissionCode || result.Resource != "conversation" || result.ReasonCode != fwiamerrors.CodePermissionDenied || result.TraceID != "core-trace" {
				t.Fatal("PRODUCTION_AUTHORIZATION_MAPPING_INVALID", result, err)
			}
			failure = true
			result, err = deps.IAMAuthzService.Authorize(context.Background(), query)
			var typed *fwiamdelegated.AuthorizationRequestError
			if result != nil || !errors.As(err, &typed) || typed.TraceID != "core-trace" || typed.RequestID != "core-request" || fwiamerrors.CodeOf(err) != fwiamerrors.CodePluginPermissionNotRegistered || fwiamerrors.StatusCode(err) != 404 || calls != 2 {
				t.Fatal("PRODUCTION_AUTHORIZATION_ERROR_LOST", result, err)
			}
		})
	}
}
