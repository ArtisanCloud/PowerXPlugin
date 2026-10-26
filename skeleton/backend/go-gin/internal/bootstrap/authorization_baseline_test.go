package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamdelegated "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	sharedapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

type bootstrapBaseline struct{}

func (*bootstrapBaseline) ReadMemberBaselines(context.Context, fwiamcontracts.AuthorizationBaselineQuery) (*fwiamcontracts.AuthorizationBaselineResult, error) {
	return nil, nil
}

func TestAuthorizationBaselineProductionProxyBinding(t *testing.T) {
	tenant, member := "20000000-0000-4000-8000-000000000001", "10000000-0000-4000-8000-000000000001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/invocations" || r.Header.Get("Authorization") != "ApiKey server-owned" {
			t.Error(r.URL, r.Header)
		}
		result := map[string]any{"schema": fwiamcontracts.AuthorizationBaselineResultSchema, "tenant_uuid": tenant, "plugin_id": sharedapp.PluginID, "evaluated_at": "2026-10-15T02:15:00.000001Z", "items": []any{map[string]any{
			"member_uuid": member, "permission_code": "orders:read", "status": "decided", "allowed": true, "baseline_kind": "initialized", "granted_at": nil,
			"known_since": "2026-10-15T02:00:00.000001Z", "visible_since": "2026-10-15T02:00:00.000001Z", "authorization_revision": "revision-1", "data_scope": "TENANT", "scope_fingerprint": "sha256:" + strings.Repeat("0", 64), "valid_until": nil, "recheck_at": nil,
		}}}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": map[string]any{"result": map[string]any{"result": result}}})
	}))
	defer server.Close()
	client, err := fwiamdelegated.NewCoreClientWithAPIKey(fwiamdelegated.CoreClientConfig{BaseURL: server.URL, PluginID: sharedapp.PluginID}, "server-owned")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := authproxy.NewDelegatedClient(server.URL, "server-owned")
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetIdentityResolver(client)
	deps := &sharedapp.Deps{IAMAdapterMode: iamservice.IAMAdapterModeDelegated, AuthProxy: proxy}
	if err := BindFrameworkIAM(deps, fwiamadapters.NewRegistry()); err != nil {
		t.Fatal(err)
	}
	defer deps.WSIdentityLifecycle.Close()
	if err := ValidateAuthorizationBaselineFeature(context.Background(), deps, true, baselineRequirements(), &baselineGrants{status: "granted"}); err != nil {
		t.Fatal(err)
	}
	provider, err := deps.IAMRegistry.AuthorizationBaselines()
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.ReadMemberBaselines(context.Background(), fwiamcontracts.AuthorizationBaselineQuery{TenantUUID: tenant, MemberUUIDs: []string{member}, PermissionCodes: []string{"orders:read"}})
	if err != nil || len(result.Items) != 1 || result.Items[0].Allowed == nil || !*result.Items[0].Allowed || result.Items[0].GrantedAt != nil {
		t.Fatal(result, err)
	}
}

type bootstrapBaselineProxy struct {
	resolverProxyStub
	provider fwiamcontracts.AuthorizationBaselineService
}

func (p bootstrapBaselineProxy) AuthorizationBaselineProvider() fwiamcontracts.AuthorizationBaselineService {
	return p.provider
}
func (bootstrapBaselineProxy) RevalidateWSIdentity(context.Context, fwiamcontracts.WSIdentityQuery) (*fwiamcontracts.WSIdentity, error) {
	return nil, nil
}

type baselineGrants struct {
	status string
	calls  int
}

func (g *baselineGrants) GrantStatus(_ context.Context, query powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	g.calls++
	items := make([]powerxcapability.GrantStatusItem, len(query.CapabilityIDs))
	for i, id := range query.CapabilityIDs {
		items[i] = powerxcapability.GrantStatusItem{CapabilityID: id, Status: g.status, ReasonCode: "CAPABILITY_GRANT_STATUS"}
	}
	return items, nil
}

func baselineRequirements() []string {
	return []string{fwiamcontracts.AuthorizationBaselineCapability, "com.corex.iam.directory.read", fwiamcontracts.IdentityContextCapability, "com.corex.capabilities.grant_status.read"}
}

func TestAuthorizationBaselineBootstrapOptionalLocalAndDelegated(t *testing.T) {
	if err := ValidateAuthorizationBaselineFeature(context.Background(), nil, false, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []iamservice.IAMAdapterMode{iamservice.IAMAdapterModeLocal, iamservice.IAMAdapterModeDelegated} {
		for _, enabledProvider := range []bool{false, true} {
			deps := &sharedapp.Deps{IAMAdapterMode: mode, IAMDirectory: resolverDirectoryStub{}}
			var provider fwiamcontracts.AuthorizationBaselineService
			if enabledProvider {
				provider = &bootstrapBaseline{}
			}
			deps.LocalAuthorizationBaselines = provider
			deps.AuthProxy = bootstrapBaselineProxy{provider: provider}
			if err := BindFrameworkIAM(deps, fwiamadapters.NewRegistry()); err != nil {
				t.Fatal(mode, err)
			}
			t.Cleanup(deps.WSIdentityLifecycle.Close)
			if err := ValidateAuthorizationBaselineFeature(context.Background(), deps, false, nil, nil); err != nil {
				t.Fatal(err)
			}
			grants := &baselineGrants{status: "granted"}
			if err := ValidateAuthorizationBaselineFeature(context.Background(), deps, true, nil, grants); !fwiamerrors.IsCode(err, "IAM_BASELINE_REQUIREMENTS_INVALID") {
				t.Fatal(err)
			}
			err := ValidateAuthorizationBaselineFeature(context.Background(), deps, true, baselineRequirements(), grants)
			if !enabledProvider {
				if !fwiamerrors.IsCode(err, fwiamerrors.CodeAdapterNotBound) {
					t.Fatal(err)
				}
				continue
			}
			if err != nil || deps.IAMAuthorizationBaselines == nil {
				t.Fatal(err)
			}
			if mode == iamservice.IAMAdapterModeLocal && grants.calls != 0 {
				t.Fatal(grants.calls)
			}
			if mode == iamservice.IAMAdapterModeDelegated {
				if grants.calls != 1 {
					t.Fatal(grants.calls)
				}
				grants.status = "not_granted"
				if err := ValidateAuthorizationBaselineFeature(context.Background(), deps, true, baselineRequirements(), grants); err == nil {
					t.Fatal("FRAMEWORK_REQUIRED_CAPABILITY_UNAVAILABLE")
				}
				for i := 0; i < len(baselineRequirements()); i++ {
					required := baselineRequirements()
					required = append(required[:i], required[i+1:]...)
					if err := ValidateAuthorizationBaselineFeature(context.Background(), deps, true, required, grants); !fwiamerrors.IsCode(err, "IAM_BASELINE_REQUIREMENTS_INVALID") {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
