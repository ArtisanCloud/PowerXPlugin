package delegated

import (
	"context"
	"errors"
	"testing"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
)

type directoryProxyStub struct {
	member         *authproxy.DirectoryMember
	members        []authproxy.DirectoryMember
	resolution     *authproxy.DirectoryMemberResolution
	nameResolution *authproxy.DirectoryMemberDisplayNameResolution
	departments    []authproxy.DirectoryDepartment
	roles          []authproxy.DirectoryRole
	permissions    []authproxy.DirectoryPermission
	decision       *authproxy.AuthorizationDecision
	err            error
}

func (s directoryProxyStub) MeContext(context.Context, string) (*authproxy.MeContext, error) {
	return nil, errors.New("not used")
}

func (s directoryProxyStub) GetDirectoryMember(context.Context, string) (*authproxy.DirectoryMember, error) {
	return s.member, s.err
}

func (s directoryProxyStub) BatchGetDirectoryMembers(context.Context, []string) ([]authproxy.DirectoryMember, error) {
	return s.members, s.err
}
func (s directoryProxyStub) BatchResolveDirectoryMembers(context.Context, []string) (*authproxy.DirectoryMemberResolution, error) {
	return s.resolution, s.err
}
func (s directoryProxyStub) BatchResolveDirectoryMembersByDisplayNames(context.Context, []string) (*authproxy.DirectoryMemberDisplayNameResolution, error) {
	return s.nameResolution, s.err
}
func (s directoryProxyStub) ListDirectoryDepartments(context.Context) ([]authproxy.DirectoryDepartment, error) {
	return s.departments, s.err
}
func (s directoryProxyStub) ListDirectoryRoles(context.Context) ([]authproxy.DirectoryRole, error) {
	return s.roles, s.err
}
func (s directoryProxyStub) ListDirectoryPermissions(context.Context) ([]authproxy.DirectoryPermission, error) {
	return s.permissions, s.err
}
func (s directoryProxyStub) CheckDirectoryAuthorization(context.Context, authproxy.DirectoryAuthorizationRequest) (*authproxy.AuthorizationDecision, error) {
	return s.decision, s.err
}

func TestAdapterBatchGetMembersRejectsOmittedMember(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{members: []authproxy.DirectoryMember{{
		MemberUUID: "member-a", TenantUUID: "tenant-a", DisplayName: "Operator",
	}}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	_, err = bundle.Directory.BatchGetMembers(context.Background(), "tenant-a", []string{"member-a", "member-b"})
	if fwiamerrors.CodeOf(err) != fwiamerrors.CodeMemberNotFound {
		t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), fwiamerrors.CodeMemberNotFound)
	}
}

func TestAdapterGetMemberMapsHostNotFound(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{err: &authproxy.ProxyError{Status: 404}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	_, err = bundle.Directory.GetMember(context.Background(), "tenant-a", "member-a")
	if fwiamerrors.CodeOf(err) != fwiamerrors.CodeMemberNotFound {
		t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), fwiamerrors.CodeMemberNotFound)
	}
}

func TestAdapterGetMemberMapsHostCapabilityAndDependencyFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		code   string
	}{
		{name: "forbidden", status: 403, code: fwiamerrors.CodeForbidden},
		{name: "upstream", status: 503, code: fwiamerrors.CodeUpstreamDependency},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle, err := NewBundle(directoryProxyStub{err: &authproxy.ProxyError{Status: tc.status, ReasonCode: tc.code}})
			if err != nil {
				t.Fatalf("NewBundle() error = %v", err)
			}
			_, err = bundle.Directory.GetMember(context.Background(), "tenant-a", "member-a")
			if fwiamerrors.CodeOf(err) != tc.code {
				t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), tc.code)
			}
		})
	}
}

func TestAdapterBatchGetMembersPreservesRequestOrderAndUUIDDTO(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{members: []authproxy.DirectoryMember{
		{MemberUUID: "member-b", TenantUUID: "tenant-a", UserUUID: "user-b", DisplayName: "Beta"},
		{MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha"},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	members, err := bundle.Directory.BatchGetMembers(context.Background(), "tenant-a", []string{"member-a", "member-b"})
	if err != nil {
		t.Fatalf("BatchGetMembers() error = %v", err)
	}
	if len(members) != 2 || members[0].MemberUUID != "member-a" || members[1].MemberUUID != "member-b" {
		t.Fatalf("members = %#v", members)
	}
	if members[0].DisplayName == members[0].MemberUUID || members[0].UserUUID == "" {
		t.Fatalf("member UUID DTO was not preserved: %#v", members[0])
	}
}

func TestAdapterBatchGetMembersRejectsDuplicateUUID(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	_, err = bundle.Directory.BatchGetMembers(context.Background(), "tenant-a", []string{"member-a", "member-a"})
	if fwiamerrors.CodeOf(err) != fwiamerrors.CodeInvalidArgument {
		t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), fwiamerrors.CodeInvalidArgument)
	}
}

func TestAdapterBatchResolveMembersPreservesResolvedAndMissingUUIDs(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{resolution: &authproxy.DirectoryMemberResolution{
		Items:              []authproxy.DirectoryMember{{MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha"}},
		MissingMemberUUIDs: []string{"member-b"},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	result, err := bundle.Directory.BatchResolveMembers(context.Background(), "tenant-a", []string{"member-a", "member-b"})
	if err != nil {
		t.Fatalf("BatchResolveMembers() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].DisplayName != "Alpha" || len(result.MissingMemberUUIDs) != 1 || result.MissingMemberUUIDs[0] != "member-b" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterBatchResolveMembersByDisplayNamesPreservesFoundMissingAndAmbiguous(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{nameResolution: &authproxy.DirectoryMemberDisplayNameResolution{Items: []authproxy.DirectoryMemberDisplayNameResolutionItem{
		{DisplayName: "Alpha", Status: "found", Member: &authproxy.DirectoryMemberDisplayNameMatch{MemberUUID: "member-a", UserUUID: "user-a", DisplayName: "Alpha"}},
		{DisplayName: "Unknown", Status: "not_found"},
		{DisplayName: "Beta", Status: "ambiguous"},
		{DisplayName: "Alpha", Status: "found", Member: &authproxy.DirectoryMemberDisplayNameMatch{MemberUUID: "member-a", UserUUID: "user-a", DisplayName: "Alpha"}},
	}}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	result, err := bundle.Directory.BatchResolveMembersByDisplayNames(context.Background(), "tenant-a", []string{"Alpha", "Unknown", "Beta", "Alpha"})
	if err != nil {
		t.Fatalf("BatchResolveMembersByDisplayNames() error = %v", err)
	}
	if len(result.Items) != 4 || result.Items[0].Status != fwiamcontracts.MemberDisplayNameResolutionFound || result.Items[0].Member == nil || result.Items[0].Member.MemberUUID != "member-a" || result.Items[0].Member.TenantUUID != "tenant-a" || result.Items[0].Member.Status != "" || result.Items[1].Status != fwiamcontracts.MemberDisplayNameResolutionNotFound || result.Items[2].Status != fwiamcontracts.MemberDisplayNameResolutionAmbiguous || result.Items[2].Member != nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterBatchResolveMembersRejectsMalformedHostResponse(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{resolution: &authproxy.DirectoryMemberResolution{
		Items: []authproxy.DirectoryMember{{MemberUUID: "member-a", TenantUUID: "other-tenant"}},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	_, err = bundle.Directory.BatchResolveMembers(context.Background(), "tenant-a", []string{"member-a"})
	if fwiamerrors.CodeOf(err) != fwiamerrors.CodeUpstreamDependency {
		t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), fwiamerrors.CodeUpstreamDependency)
	}
}

func TestAdapterListDirectoryRecordsMapsUUIDDTOs(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{
		departments: []authproxy.DirectoryDepartment{{DepartmentUUID: "department-a", TenantUUID: "tenant-a", Name: "Operations", Code: "ops", ParentDepartmentUUID: "department-root"}},
		roles:       []authproxy.DirectoryRole{{RoleUUID: "role-a", TenantUUID: "tenant-a", Code: "operator", Name: "Operator", Description: "handles operations"}},
		permissions: []authproxy.DirectoryPermission{{PermissionUUID: "permission-a", Resource: "iam.member", Action: "read", Scope: "tenant"}},
	})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	departments, err := bundle.Directory.ListDepartments(context.Background(), "tenant-a")
	if err != nil || len(departments) != 1 || departments[0].DepartmentUUID != "department-a" || departments[0].ParentDepartmentUUID != "department-root" {
		t.Fatalf("ListDepartments() = %#v, %v", departments, err)
	}
	roles, err := bundle.Directory.ListRoles(context.Background(), "tenant-a")
	if err != nil || len(roles) != 1 || roles[0].RoleUUID != "role-a" {
		t.Fatalf("ListRoles() = %#v, %v", roles, err)
	}
	permissions, err := bundle.Directory.ListPermissions(context.Background(), "tenant-a")
	if err != nil || len(permissions) != 1 || permissions[0].PermissionUUID != "permission-a" || permissions[0].Resource != "iam.member" {
		t.Fatalf("ListPermissions() = %#v, %v", permissions, err)
	}
}

func TestAdapterAuthorizePreservesDeniedDecision(t *testing.T) {
	bundle, err := NewBundle(directoryProxyStub{decision: &authproxy.AuthorizationDecision{Allowed: false, ReasonCode: "IAM_PERMISSION_DENIED"}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	decision, err := bundle.Authz.Authorize(context.Background(), fwiamcontracts.AuthorizationRequest{TenantUUID: "tenant-a", MemberUUID: "member-a", UserUUID: "user-a", Resource: "iam.member", Action: "read"})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}
	if decision.Allowed || decision.ReasonCode != "IAM_PERMISSION_DENIED" || decision.Mode != string(fwiamcontracts.IAMAdapterModeDelegated) {
		t.Fatalf("decision = %#v", decision)
	}
}

var _ fwiamcontracts.DirectoryService = (*Adapter)(nil)
