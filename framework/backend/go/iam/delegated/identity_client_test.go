package delegated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

const identityPlugin = "com.example.identity"
const identityTenant = "11111111-1111-4111-8111-111111111111"
const identityUser = "22222222-2222-4222-8222-222222222222"
const identityTicket = "pxctx_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func validIdentity() contracts.DelegatedIdentity {
	return contracts.DelegatedIdentity{Schema: contracts.IdentityResultSchema, Mode: "delegated", PluginID: identityPlugin, TenantUUID: identityTenant, UserUUID: identityUser, IsRoot: true, Roles: []string{}, Permissions: []string{}, PolicyVersion: "policy-1", PermsHash: "perms-1", SourceMethod: "GET", SourcePath: "/admin/records", SourceRequiredPermission: "records:read", Transport: "http", SourceRequestID: "original-request", SourceTraceID: "original-trace", TraceID: "resolve-trace", ProofExpiresAt: time.Now().UTC().Add(30 * time.Second), UserSessionExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
}
func writeIdentity(w http.ResponseWriter, identity any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"result": map[string]any{"result": identity}}})
}

func TestIdentityClientTypedAPIKeyAndSTS(t *testing.T) {
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		t.Run(scheme, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/api/v1/tenant/invocations" || r.Method != "POST" || r.Header.Get("Authorization") != scheme+" server-owned" || r.URL.RawQuery != "" || r.Header.Get("tenant_uuid") != "" {
					t.Errorf("incorrect identity transport")
				}
				if r.Header.Get("X-Request-ID") != "original-request" || r.Header.Get("X-Trace-ID") != "original-trace" {
					t.Error("correlation metadata lost")
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				expected := `{"capability_id":"com.corex.iam.identity_context.service_read","preferred_protocol":"core_internal","payload":{"body":{"operation":"resolve_request_identity","request":{"schema":"powerx.iam.plugin-request-identity-query/v1","context_ticket":"` + identityTicket + `"}}}}`
				var wanted map[string]any
				_ = json.Unmarshal([]byte(expected), &wanted)
				actual, _ := json.Marshal(body)
				want, _ := json.Marshal(wanted)
				if string(actual) != string(want) {
					t.Error("typed request contract mismatch")
				}
				writeIdentity(w, validIdentity())
			}))
			defer server.Close()
			cfg := CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "server-owned", nil })}
			client, err := NewCoreClient(cfg)
			if scheme == "ApiKey" {
				client, err = NewCoreClientWithAPIKey(cfg, "server-owned")
			}
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithRequestMetadata(context.Background(), RequestMetadata{RequestID: "original-request", TraceID: "original-trace"})
			result, err := client.ResolveDelegatedIdentity(ctx, contracts.DelegatedIdentityQuery{ContextTicket: identityTicket})
			if err != nil || result.MemberUUID != nil || !result.IsRoot || result.SourceTraceID != "original-trace" || result.TraceID != "resolve-trace" || !result.ProofExpiresAt.Before(result.UserSessionExpiresAt) {
				t.Fatalf("identity mapping failed: %v", err)
			}
			if _, err = client.ResolveIdentity(ctx, "inbound-plugin-bearer"); iamerrors.CodeOf(err) != iamerrors.CodeIdentityMigrationRequired || calls != 1 {
				t.Error("old delegated identity must fail without a user endpoint request")
			}
		})
	}
}

func TestIdentityClientPreservesErrorPlaneStatusAndTrace(t *testing.T) {
	for _, row := range []struct {
		status int
		code   string
	}{{400, "IAM_CONTEXT_INVALID_ARGUMENT"}, {401, "IAM_CONTEXT_SERVICE_CREDENTIAL_EXPIRED"}, {401, "IAM_CONTEXT_PROOF_EXPIRED"}, {401, "IAM_CONTEXT_USER_SESSION_INVALID"}, {403, "IAM_CONTEXT_PLUGIN_FORBIDDEN"}, {409, "IAM_CONTEXT_POLICY_CHANGED"}, {424, "CAPABILITY_MANUAL_UPGRADE_REQUIRED"}, {503, "IAM_CONTEXT_UNAVAILABLE"}} {
		t.Run(row.code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Trace-ID", "resolve-trace")
				w.Header().Set("X-Request-ID", "upstream-request")
				w.WriteHeader(row.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": row.status, "error_code": row.code, "reason_code": row.code, "message": identityTicket, "source_trace_id": "original-trace", "source_request_id": "original-request"})
			}))
			defer server.Close()
			client, _ := NewCoreClient(CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "service-token", nil })})
			_, err := client.ResolveDelegatedIdentity(context.Background(), contracts.DelegatedIdentityQuery{ContextTicket: identityTicket})
			var out *IdentityRequestError
			if !errors.As(err, &out) || iamerrors.StatusCode(err) != row.status || iamerrors.CodeOf(err) != row.code || out.TraceID != "resolve-trace" || out.RequestID != "upstream-request" || out.SourceTraceID != "original-trace" || strings.Contains(err.Error(), identityTicket) {
				t.Fatalf("identity error not retained safely: %v", err)
			}
		})
	}
}

func TestIdentityClientRejectsIncompleteAndCrossScopeResults(t *testing.T) {
	cases := map[string]func(map[string]any){
		"missing-root":        func(v map[string]any) { delete(v, "is_root") },
		"missing-null-member": func(v map[string]any) { delete(v, "member_uuid") },
		"member-required":     func(v map[string]any) { v["is_root"] = false },
		"invalid-tenant":      func(v map[string]any) { v["tenant_uuid"] = "42" },
		"cross-plugin":        func(v map[string]any) { v["plugin_id"] = "another-plugin" },
		"old-expiry": func(v map[string]any) {
			delete(v, "proof_expires_at")
			v["expires_at"] = time.Now().UTC().Add(time.Minute)
		},
		"null-permissions":    func(v map[string]any) { v["permissions"] = nil },
		"proof-after-session": func(v map[string]any) { v["user_session_expires_at"] = time.Now().UTC().Add(time.Second) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(validIdentity())
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			mutate(body)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeIdentity(w, body) }))
			defer server.Close()
			client, _ := NewCoreClient(CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "service-token", nil })})
			_, err := client.ResolveDelegatedIdentity(context.Background(), contracts.DelegatedIdentityQuery{ContextTicket: identityTicket})
			if err == nil {
				t.Fatal("invalid identity was accepted")
			}
		})
	}
}
func TestIdentityInputFailsBeforeHTTPAndRedirectsAreRejected(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/other", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, _ := NewCoreClient(CoreClientConfig{BaseURL: server.URL, PluginID: identityPlugin, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "service-token", nil })})
	for _, ticket := range []string{"", "inbound-plugin-bearer"} {
		if _, err := client.ResolveDelegatedIdentity(context.Background(), contracts.DelegatedIdentityQuery{ContextTicket: ticket}); err == nil {
			t.Fatal("bad proof accepted")
		}
	}
	if calls != 0 {
		t.Fatal("invalid proof made an HTTP request")
	}
	_, err := client.ResolveDelegatedIdentity(context.Background(), contracts.DelegatedIdentityQuery{ContextTicket: identityTicket})
	if err == nil || calls != 1 {
		t.Fatalf("redirect followed: %v calls=%d", err, calls)
	}
	if strings.Contains(fmt.Sprint(contracts.DelegatedIdentityQuery{ContextTicket: identityTicket}), identityTicket) {
		t.Fatal("proof exposed by formatting")
	}
}
