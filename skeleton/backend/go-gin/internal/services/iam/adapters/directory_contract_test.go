package adapters_test

import (
	"context"
	"errors"
	"testing"

	fwcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	delegated "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/delegated"
	local "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/local"
)

const (
	contractTenantUUID = "tenant-a"
	contractMemberA    = "member-a"
	contractMemberB    = "member-b"
)

func TestProductionDirectoryAdaptersHaveEquivalentUUIDReadSemantics(t *testing.T) {
	localBundle, err := local.NewBundle(contractLocalDirectory{members: map[string]iamservice.MemberInfo{
		contractMemberA: {MemberUUID: contractMemberA, TenantUUID: contractTenantUUID, UserUUID: "user-a", DisplayName: "Alpha", Status: "active"},
		contractMemberB: {MemberUUID: contractMemberB, TenantUUID: contractTenantUUID, UserUUID: "user-b", DisplayName: "Beta", Status: "active"},
	}})
	if err != nil {
		t.Fatalf("build local bundle: %v", err)
	}
	delegatedBundle, err := delegated.NewBundle(contractDelegatedProxy{
		departments: []authproxy.DirectoryDepartment{{DepartmentUUID: "department-a", TenantUUID: contractTenantUUID, Name: "Operations", Code: "ops"}},
		member:      &authproxy.DirectoryMember{MemberUUID: contractMemberA, TenantUUID: contractTenantUUID, UserUUID: "user-a", DisplayName: "Alpha", Status: 1},
		members: []authproxy.DirectoryMember{
			{MemberUUID: contractMemberB, TenantUUID: contractTenantUUID, UserUUID: "user-b", DisplayName: "Beta", Status: 1},
			{MemberUUID: contractMemberA, TenantUUID: contractTenantUUID, UserUUID: "user-a", DisplayName: "Alpha", Status: 1},
		},
	})
	if err != nil {
		t.Fatalf("build delegated bundle: %v", err)
	}

	for _, tc := range []struct {
		name      string
		directory fwcontracts.DirectoryService
	}{
		{name: "local", directory: localBundle.Directory},
		{name: "delegated", directory: delegatedBundle.Directory},
	} {
		t.Run(tc.name, func(t *testing.T) {
			departments, err := tc.directory.ListDepartments(context.Background(), contractTenantUUID)
			if err != nil || len(departments) != 1 {
				t.Fatalf("ListDepartments() = %#v, %v", departments, err)
			}
			if departments[0].DepartmentUUID == "" || departments[0].TenantUUID != contractTenantUUID {
				t.Fatalf("department is not UUID-only tenant scoped: %#v", departments[0])
			}

			member, err := tc.directory.GetMember(context.Background(), contractTenantUUID, contractMemberA)
			if err != nil {
				t.Fatalf("GetMember() error = %v", err)
			}
			if member.MemberUUID != contractMemberA || member.UserUUID != "user-a" || member.DisplayName != "Alpha" || member.DisplayName == member.MemberUUID {
				t.Fatalf("member DTO is invalid: %#v", member)
			}

			members, err := tc.directory.BatchGetMembers(context.Background(), contractTenantUUID, []string{contractMemberA, contractMemberB})
			if err != nil {
				t.Fatalf("BatchGetMembers() error = %v", err)
			}
			if len(members) != 2 || members[0].MemberUUID != contractMemberA || members[1].MemberUUID != contractMemberB {
				t.Fatalf("batch must preserve UUID request order: %#v", members)
			}
		})
	}
}

type contractLocalDirectory struct {
	members map[string]iamservice.MemberInfo
}

func (contractLocalDirectory) Mode() iamservice.IAMAdapterMode { return iamservice.IAMAdapterModeLocal }
func (contractLocalDirectory) Login(context.Context, iamservice.LoginRequest) (*iamservice.AuthTokens, *iamservice.UserContext, error) {
	return nil, nil, errors.New("not implemented")
}
func (contractLocalDirectory) Refresh(context.Context, string) (*iamservice.AuthTokens, error) {
	return nil, errors.New("not implemented")
}
func (contractLocalDirectory) Logout(context.Context, string) error {
	return errors.New("not implemented")
}
func (contractLocalDirectory) CurrentUser(context.Context) (*iamservice.UserContext, error) {
	return nil, errors.New("not implemented")
}
func (contractLocalDirectory) ListRoles(context.Context, string) ([]iamservice.RoleInfo, error) {
	return nil, nil
}
func (contractLocalDirectory) ListDepartments(context.Context, string) ([]iamservice.DepartmentInfo, error) {
	return []iamservice.DepartmentInfo{{UUID: "department-a", TenantUUID: contractTenantUUID, Name: "Operations", Code: "ops"}}, nil
}
func (d contractLocalDirectory) ListMembers(context.Context, string) ([]iamservice.MemberInfo, error) {
	return []iamservice.MemberInfo{d.members[contractMemberA], d.members[contractMemberB]}, nil
}
func (d contractLocalDirectory) GetMember(_ context.Context, _ string, memberUUID string) (*iamservice.MemberInfo, error) {
	member, ok := d.members[memberUUID]
	if !ok {
		return nil, iamservice.ErrMemberNotFound
	}
	return &member, nil
}
func (contractLocalDirectory) CheckPermission(context.Context, iamservice.TenantContext, string, string) error {
	return nil
}

type contractDelegatedProxy struct {
	departments []authproxy.DirectoryDepartment
	member      *authproxy.DirectoryMember
	members     []authproxy.DirectoryMember
}

func (contractDelegatedProxy) MeContext(context.Context, string) (*authproxy.MeContext, error) {
	return nil, errors.New("not implemented")
}
func (contractDelegatedProxy) GetDirectoryTenant(context.Context) (*authproxy.DirectoryTenant, error) {
	return &authproxy.DirectoryTenant{TenantUUID: contractTenantUUID}, nil
}
func (contractDelegatedProxy) ListDirectoryMembers(context.Context, int, int) (*authproxy.DirectoryMemberPage, error) {
	return &authproxy.DirectoryMemberPage{}, nil
}
func (p contractDelegatedProxy) GetDirectoryMember(context.Context, string) (*authproxy.DirectoryMember, error) {
	return p.member, nil
}
func (p contractDelegatedProxy) BatchGetDirectoryMembers(context.Context, []string) ([]authproxy.DirectoryMember, error) {
	return p.members, nil
}
func (contractDelegatedProxy) BatchResolveDirectoryMembers(context.Context, []string) (*authproxy.DirectoryMemberResolution, error) {
	return nil, errors.New("not implemented")
}
func (contractDelegatedProxy) BatchResolveDirectoryMembersByDisplayNames(context.Context, []string) (*authproxy.DirectoryMemberDisplayNameResolution, error) {
	return nil, errors.New("not implemented")
}
func (p contractDelegatedProxy) ListDirectoryDepartments(context.Context) ([]authproxy.DirectoryDepartment, error) {
	return p.departments, nil
}
func (contractDelegatedProxy) ListDirectoryRoles(context.Context) ([]authproxy.DirectoryRole, error) {
	return nil, nil
}
func (contractDelegatedProxy) ListDirectoryPermissions(context.Context) ([]authproxy.DirectoryPermission, error) {
	return nil, nil
}
func (contractDelegatedProxy) CheckDirectoryAuthorization(context.Context, authproxy.DirectoryAuthorizationRequest) (*authproxy.AuthorizationDecision, error) {
	return nil, errors.New("not implemented")
}
