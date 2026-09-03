package local

import (
	"context"
	"errors"
	"testing"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

type directoryStub struct {
	members              map[string]iamservice.MemberInfo
	permissions          []iamservice.PermissionInfo
	displayNameResult    []iamservice.MemberDisplayNameResolutionItem
	displayNameResultErr error
}

func (directoryStub) Mode() iamservice.IAMAdapterMode { return iamservice.IAMAdapterModeLocal }
func (directoryStub) Login(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
	return nil, nil, errors.New("not used")
}
func (directoryStub) Refresh(context.Context, string) (*iamservice.AuthTokens, error) {
	return nil, errors.New("not used")
}
func (directoryStub) Logout(context.Context, string) error { return errors.New("not used") }
func (directoryStub) CurrentUser(context.Context) (*iamservice.UserContext, error) {
	return nil, errors.New("not used")
}
func (directoryStub) ListRoles(context.Context, string) ([]iamservice.RoleInfo, error) {
	return nil, nil
}
func (directoryStub) ListDepartments(context.Context, string) ([]iamservice.DepartmentInfo, error) {
	return nil, nil
}
func (s directoryStub) ListMembers(context.Context, string) ([]iamservice.MemberInfo, error) {
	result := make([]iamservice.MemberInfo, 0, len(s.members))
	for _, member := range s.members {
		result = append(result, member)
	}
	return result, nil
}
func (s directoryStub) GetMember(_ context.Context, _ string, memberUUID string) (*iamservice.MemberInfo, error) {
	member, ok := s.members[memberUUID]
	if !ok {
		return nil, iamservice.ErrMemberNotFound
	}
	return &member, nil
}
func (directoryStub) CheckPermission(context.Context, iamservice.TenantContext, string, string) error {
	return nil
}
func (s directoryStub) ListPermissions(context.Context, string) ([]iamservice.PermissionInfo, error) {
	return s.permissions, nil
}
func (s directoryStub) BatchResolveMembersByDisplayNames(context.Context, string, []string) ([]iamservice.MemberDisplayNameResolutionItem, error) {
	return s.displayNameResult, s.displayNameResultErr
}

func TestAdapterGetMemberUsesUUIDAndNeverUsesItAsDisplayName(t *testing.T) {
	bundle, err := NewBundle(directoryStub{members: map[string]iamservice.MemberInfo{
		"member-a": {MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha", Status: "active"},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	member, err := bundle.Directory.GetMember(context.Background(), "tenant-a", "member-a")
	if err != nil {
		t.Fatalf("GetMember() error = %v", err)
	}
	if member.MemberUUID != "member-a" || member.UserUUID != "user-a" || member.DisplayName != "Alpha" {
		t.Fatalf("member = %#v", member)
	}
	if member.DisplayName == member.MemberUUID {
		t.Fatalf("display_name must not use member_uuid")
	}
}

func TestAdapterGetMemberMapsNotFound(t *testing.T) {
	bundle, err := NewBundle(directoryStub{})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	_, err = bundle.Directory.GetMember(context.Background(), "tenant-a", "missing")
	if fwiamerrors.CodeOf(err) != fwiamerrors.CodeMemberNotFound {
		t.Fatalf("error code = %q, want %q", fwiamerrors.CodeOf(err), fwiamerrors.CodeMemberNotFound)
	}
}

func TestAdapterListMembersPageUsesBoundedDirectoryPage(t *testing.T) {
	bundle, err := NewBundle(directoryStub{members: map[string]iamservice.MemberInfo{
		"member-a": {MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha"},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	page, err := bundle.Directory.ListMembersPage(context.Background(), "tenant-a", fwiamcontracts.MemberPageRequest{Page: 1, PageSize: 1})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].MemberUUID != "member-a" {
		t.Fatalf("ListMembersPage() = %#v, %v", page, err)
	}
}

func TestAdapterBatchResolveMembersReturnsMissingWithoutUUIDDisplayFallback(t *testing.T) {
	bundle, err := NewBundle(directoryStub{members: map[string]iamservice.MemberInfo{
		"member-a": {MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha", Status: "active"},
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

func TestAdapterBatchResolveMembersByDisplayNamesPreservesPerInputOutcomes(t *testing.T) {
	bundle, err := NewBundle(directoryStub{displayNameResult: []iamservice.MemberDisplayNameResolutionItem{
		{DisplayName: "Alpha", Status: iamservice.MemberDisplayNameFound, Member: &iamservice.MemberInfo{MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha"}},
		{DisplayName: "Unknown", Status: iamservice.MemberDisplayNameNotFound},
		{DisplayName: "Beta", Status: iamservice.MemberDisplayNameAmbiguous},
		{DisplayName: "Alpha", Status: iamservice.MemberDisplayNameFound, Member: &iamservice.MemberInfo{MemberUUID: "member-a", TenantUUID: "tenant-a", UserUUID: "user-a", DisplayName: "Alpha"}},
	}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	result, err := bundle.Directory.BatchResolveMembersByDisplayNames(context.Background(), "tenant-a", []string{"Alpha", "Unknown", "Beta", "Alpha"})
	if err != nil {
		t.Fatalf("BatchResolveMembersByDisplayNames() error = %v", err)
	}
	if len(result.Items) != 4 || result.Items[0].Status != fwiamcontracts.MemberDisplayNameResolutionFound || result.Items[0].Member == nil || result.Items[0].Member.MemberUUID != "member-a" || result.Items[1].Status != fwiamcontracts.MemberDisplayNameResolutionNotFound || result.Items[2].Status != fwiamcontracts.MemberDisplayNameResolutionAmbiguous || result.Items[2].Member != nil || result.Items[3].DisplayName != "Alpha" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterListPermissionsUsesPermissionUUID(t *testing.T) {
	bundle, err := NewBundle(directoryStub{permissions: []iamservice.PermissionInfo{{PermissionUUID: "permission-a", Resource: "iam.member", Action: "read", Scope: "tenant"}}})
	if err != nil {
		t.Fatalf("NewBundle() error = %v", err)
	}
	permissions, err := bundle.Directory.ListPermissions(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("ListPermissions() error = %v", err)
	}
	if len(permissions) != 1 || permissions[0].PermissionUUID != "permission-a" || permissions[0].Resource != "iam.member" {
		t.Fatalf("permissions = %#v", permissions)
	}
}

var _ fwiamcontracts.DirectoryService = (*Adapter)(nil)
