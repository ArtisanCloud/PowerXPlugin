package delegated

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
)

type hostStub struct{}

func (hostStub) GetTenant(context.Context, string) (*contracts.Tenant, error) {
	return &contracts.Tenant{TenantUUID: "tenant"}, nil
}

func TestCoreClientUsesSTSAndMapsStableCoreError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sts-token" {
			t.Fatalf("Authorization=%q", got)
		}
		if r.URL.Path != "/api/v1/tenant/iam/departments" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"items":[{"department_uuid":"department-1","tenant_uuid":"tenant-1","name":"Operations"}]}}`))
	}))
	defer server.Close()
	client, err := NewCoreClient(CoreClientConfig{BaseURL: server.URL, Tokens: TokenProviderFunc(func(context.Context) (string, error) { return "sts-token", nil })})
	if err != nil {
		t.Fatalf("NewCoreClient: %v", err)
	}
	items, err := client.ListDepartments(context.Background(), "tenant-1")
	if err != nil || len(items) != 1 || items[0].DepartmentUUID != "department-1" {
		t.Fatalf("ListDepartments=%#v, %v", items, err)
	}
}
func (hostStub) ListDepartments(context.Context, string) ([]contracts.Department, error) {
	return nil, nil
}
func (hostStub) ListMembers(context.Context, string) ([]contracts.Member, error) { return nil, nil }
func (hostStub) ListMembersPage(context.Context, string, contracts.MemberPageRequest) (*contracts.MemberPage, error) {
	return &contracts.MemberPage{}, nil
}
func (hostStub) GetMember(context.Context, string, string) (*contracts.Member, error) {
	return nil, nil
}
func (hostStub) BatchGetMembers(context.Context, string, []string) ([]contracts.Member, error) {
	return nil, nil
}
func (hostStub) BatchResolveMembers(context.Context, string, []string) (*contracts.MemberResolution, error) {
	return &contracts.MemberResolution{}, nil
}
func (hostStub) BatchResolveMembersByDisplayNames(context.Context, string, []string) (*contracts.MemberDisplayNameResolution, error) {
	return &contracts.MemberDisplayNameResolution{}, nil
}
func (hostStub) ListRoles(context.Context, string) ([]contracts.Role, error) { return nil, nil }
func (hostStub) ListPermissions(context.Context, string) ([]contracts.Permission, error) {
	return nil, nil
}
func (hostStub) Authorize(context.Context, contracts.AuthorizationRequest) (*contracts.AuthorizationDecision, error) {
	return &contracts.AuthorizationDecision{Allowed: true}, nil
}
func (hostStub) ResolveIdentity(context.Context, string) (*contracts.IdentityContext, error) {
	return &contracts.IdentityContext{TenantUUID: "tenant"}, nil
}

func TestNewBundleForwardsToCoreHost(t *testing.T) {
	bundle, err := NewBundle(hostStub{})
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	member, err := bundle.Context.ResolveIdentity(context.Background(), "customer-token")
	if err != nil || member.TenantUUID != "tenant" {
		t.Fatalf("ResolveIdentity = %#v, %v", member, err)
	}
}

func TestNewBundleRejectsNilHost(t *testing.T) {
	if _, err := NewBundle(nil); err == nil {
		t.Fatal("expected missing host to fail")
	}
}
