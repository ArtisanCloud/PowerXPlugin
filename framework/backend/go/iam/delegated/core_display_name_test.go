package delegated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

const (
	displayNameTenant      = "12e90d14-53e9-4598-a6c2-7ce30d080883"
	displayNameOtherTenant = "e4e05b2c-a39e-4b34-a0ca-38dbe3f40149"
	displayNameMember      = "fda3589b-3bc7-4c2f-a5b4-0b5b33a61391"
	displayNameUser        = "810b7272-7d8b-45a7-99e0-d72a37fe35ef"
)

// The fixture mirrors Core's lean response: found members have no tenant_uuid
// or status; ambiguous matches are candidates, not a selected Member.
func displayNameCoreItem(name string) map[string]any {
	return map[string]any{"display_name": name, "status": "found", "member": map[string]any{
		"member_uuid": displayNameMember, "user_uuid": displayNameUser, "display_name": name,
	}}
}

func TestCoreDisplayNameBindsCredentialTenant(t *testing.T) {
	for _, scheme := range []string{"Bearer", "ApiKey"} {
		t.Run(scheme, func(t *testing.T) {
			var tokenCalls, scopeCalls, resolveCalls atomic.Int32
			names := []string{"planner-1", "missing", "duplicate", "planner-1"}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != scheme+" service-credential" || r.URL.RawQuery != "" {
					t.Errorf("auth_scheme=%s query=%s", scheme, r.URL.RawQuery)
				}
				switch r.URL.Path {
				case "/api/v1/tenant/iam/tenant":
					scopeCalls.Add(1)
					if r.Method != http.MethodGet {
						t.Errorf("method=%s", r.Method)
					}
					writeCoreTenantTestResponse(t, w, displayNameTenant, 1)
				case "/api/v1/tenant/iam/members:batch-find-by-display-names":
					resolveCalls.Add(1)
					if r.Method != http.MethodPost {
						t.Errorf("method=%s", r.Method)
					}
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					var got []string
					if err := json.Unmarshal(body["display_names"], &got); err != nil {
						t.Error(err)
					}
					if len(body) != 1 || !slices.Equal(got, names) {
						t.Errorf("body=%v", body)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"items": []any{
						displayNameCoreItem(names[0]),
						map[string]any{"display_name": names[1], "status": "not_found"},
						map[string]any{"display_name": names[2], "status": "ambiguous", "members": []any{
							map[string]any{"member_uuid": displayNameMember, "user_uuid": displayNameUser, "display_name": names[2]},
							map[string]any{"member_uuid": "2aee02f6-b995-4014-9597-35641d0b6d5a", "user_uuid": "5ae67a76-7b61-4a9f-b787-597d1cc5d10c", "display_name": names[2]},
						}},
						displayNameCoreItem(names[3]),
					}}})
				default:
					t.Errorf("path=%s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			cfg := CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) {
				tokenCalls.Add(1)
				return "service-credential", nil
			})}
			var client *CoreClient
			var err error
			if scheme == "ApiKey" {
				client, err = NewCoreClientWithAPIKey(cfg, "service-credential")
			} else {
				client, err = NewCoreClient(cfg)
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.BatchResolveMembersByDisplayNames(context.Background(), displayNameTenant, names)
			if err != nil {
				t.Fatal(err)
			}
			if scopeCalls.Load() != 1 || resolveCalls.Load() != 1 {
				t.Fatalf("scope=%d resolve=%d", scopeCalls.Load(), resolveCalls.Load())
			}
			if scheme == "Bearer" && tokenCalls.Load() != 1 {
				t.Fatalf("credentials=%d", tokenCalls.Load())
			}
			if result == nil || len(result.Items) != len(names) {
				t.Fatalf("result=%#v", result)
			}
			for _, index := range []int{0, 3} {
				member := result.Items[index].Member
				if member == nil || member.TenantUUID != displayNameTenant || member.MemberUUID != displayNameMember || member.UserUUID != displayNameUser || member.Status != "" {
					t.Fatalf("member=%#v", member)
				}
			}
			if result.Items[1].Status != contracts.MemberDisplayNameResolutionNotFound || result.Items[2].Status != contracts.MemberDisplayNameResolutionAmbiguous || result.Items[2].Member != nil {
				t.Fatalf("items=%#v", result.Items)
			}
		})
	}
}

func TestCoreDisplayNameRejectsUnverifiedScope(t *testing.T) {
	for _, tc := range []struct{ name, asserted, scope, code string }{
		{"empty", "", displayNameTenant, iamerrors.CodeInvalidArgument},
		{"invalid", "tenant-1", displayNameTenant, iamerrors.CodeInvalidArgument},
		{"cross-tenant", displayNameOtherTenant, displayNameTenant, iamerrors.CodeForbidden},
		{"missing-scope", displayNameTenant, "", iamerrors.CodeUpstreamDependency},
		{"invalid-scope", displayNameTenant, "tenant-1", iamerrors.CodeUpstreamDependency},
		{"zero-scope", displayNameTenant, "00000000-0000-0000-0000-000000000000", iamerrors.CodeUpstreamDependency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var resolveCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/tenant/iam/tenant" {
					writeCoreTenantTestResponse(t, w, tc.scope, 1)
					return
				}
				resolveCalls.Add(1)
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"items": []any{displayNameCoreItem("planner-1")}}})
			}))
			defer server.Close()
			client, err := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL}, "service-credential")
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.BatchResolveMembersByDisplayNames(context.Background(), tc.asserted, []string{"planner-1"})
			if result != nil || iamerrors.CodeOf(err) != tc.code || resolveCalls.Load() != 0 {
				t.Fatalf("result=%#v code=%s resolve=%d", result, iamerrors.CodeOf(err), resolveCalls.Load())
			}
		})
	}
}

func TestCoreDisplayNameRejectsInvalidResolution(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any) []any
	}{
		{"missing-items", func(map[string]any) []any { return nil }},
		{"wrong-name", func(item map[string]any) []any { item["display_name"] = "planner-2"; return []any{item} }},
		{"missing-member", func(item map[string]any) []any { delete(item, "member"); return []any{item} }},
		{"invalid-member-uuid", func(item map[string]any) []any {
			item["member"].(map[string]any)["member_uuid"] = "member-1"
			return []any{item}
		}},
		{"missing-user-uuid", func(item map[string]any) []any {
			delete(item["member"].(map[string]any), "user_uuid")
			return []any{item}
		}},
		{"cross-tenant-member", func(item map[string]any) []any {
			item["member"].(map[string]any)["tenant_uuid"] = displayNameOtherTenant
			return []any{item}
		}},
		{"ambiguous-selected-member", func(item map[string]any) []any { item["status"] = "ambiguous"; return []any{item} }},
		{"unknown-status", func(item map[string]any) []any { item["status"] = "unknown"; return []any{item} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := tc.mutate(displayNameCoreItem("planner-1"))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/tenant/iam/tenant" {
					writeCoreTenantTestResponse(t, w, displayNameTenant, 1)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"items": items}})
			}))
			defer server.Close()
			client, err := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL}, "service-credential")
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.BatchResolveMembersByDisplayNames(context.Background(), displayNameTenant, []string{"planner-1"})
			if result != nil || iamerrors.CodeOf(err) != iamerrors.CodeUpstreamDependency {
				t.Fatalf("result=%#v code=%s", result, iamerrors.CodeOf(err))
			}
		})
	}
}

func TestCoreDisplayNamePreservesUpstreamFailure(t *testing.T) {
	for _, endpoint := range []string{"/tenant", "/members:batch-find-by-display-names"} {
		for _, status := range []int{401, 403, 503} {
			t.Run(fmt.Sprintf("%s/%d", endpoint, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v1/tenant/iam"+endpoint {
						w.Header().Set("X-Trace-Id", "test-trace")
						w.WriteHeader(status)
						fmt.Fprint(w, `{"reason_code":"IAM_UPSTREAM_DEPENDENCY"}`)
						return
					}
					writeCoreTenantTestResponse(t, w, displayNameTenant, 1)
				}))
				defer server.Close()
				client, err := NewCoreClientWithAPIKey(CoreClientConfig{BaseURL: server.URL}, "service-credential")
				if err != nil {
					t.Fatal(err)
				}
				result, err := client.BatchResolveMembersByDisplayNames(context.Background(), displayNameTenant, []string{"planner-1"})
				var upstream *CoreRequestError
				if result != nil || !errors.As(err, &upstream) || upstream.Status != status || upstream.TraceID != "test-trace" {
					t.Fatalf("result=%#v err=%v", result, err)
				}
			})
		}
	}
}
