package contracts

import (
	"context"
	"errors"
	"testing"
)

type stubDirectoryService struct {
	getTenantFn      func(ctx context.Context, tenantUUID string) (*Tenant, error)
	listDepartments  []Department
	listMembers      []Member
	listRoles        []Role
	listPermissions  []Permission
	listFailureError error
}

func (s *stubDirectoryService) GetTenant(ctx context.Context, tenantUUID string) (*Tenant, error) {
	if s.getTenantFn != nil {
		return s.getTenantFn(ctx, tenantUUID)
	}
	return &Tenant{TenantUUID: tenantUUID, Name: "stub-tenant"}, nil
}

func (s *stubDirectoryService) ListDepartments(context.Context, string) ([]Department, error) {
	if s.listFailureError != nil {
		return nil, s.listFailureError
	}
	return s.listDepartments, nil
}

func (s *stubDirectoryService) ListMembers(context.Context, string) ([]Member, error) {
	if s.listFailureError != nil {
		return nil, s.listFailureError
	}
	return s.listMembers, nil
}

func (s *stubDirectoryService) GetMember(_ context.Context, _ string, memberUUID string) (*Member, error) {
	for _, member := range s.listMembers {
		if member.MemberUUID == memberUUID {
			copy := member
			return &copy, nil
		}
	}
	return nil, errors.New("member not found")
}

func (s *stubDirectoryService) BatchGetMembers(_ context.Context, _ string, memberUUIDs []string) ([]Member, error) {
	result := make([]Member, 0, len(memberUUIDs))
	for _, memberUUID := range memberUUIDs {
		member, err := s.GetMember(context.Background(), "", memberUUID)
		if err != nil {
			return nil, err
		}
		result = append(result, *member)
	}
	return result, nil
}

func (s *stubDirectoryService) BatchResolveMembers(_ context.Context, _ string, memberUUIDs []string) (*MemberResolution, error) {
	result := &MemberResolution{}
	for _, memberUUID := range memberUUIDs {
		member, err := s.GetMember(context.Background(), "", memberUUID)
		if err != nil {
			result.MissingMemberUUIDs = append(result.MissingMemberUUIDs, memberUUID)
			continue
		}
		result.Items = append(result.Items, *member)
	}
	return result, nil
}

func (s *stubDirectoryService) ListRoles(context.Context, string) ([]Role, error) {
	if s.listFailureError != nil {
		return nil, s.listFailureError
	}
	return s.listRoles, nil
}

func (s *stubDirectoryService) ListPermissions(context.Context, string) ([]Permission, error) {
	if s.listFailureError != nil {
		return nil, s.listFailureError
	}
	return s.listPermissions, nil
}

func TestDirectoryContractServiceSnapshot(t *testing.T) {
	service := NewDirectoryContractService(&stubDirectoryService{
		listDepartments: []Department{{DepartmentUUID: "dep-1", TenantUUID: "tenant-1", Name: "dep"}},
		listMembers:     []Member{{MemberUUID: "member-1", TenantUUID: "tenant-1", UserUUID: "user-1"}},
		listRoles:       []Role{{RoleUUID: "role-1", TenantUUID: "tenant-1", Code: "admin", Name: "Admin"}},
		listPermissions: []Permission{{PermissionUUID: "permission-1", Resource: "iam.role", Action: "read"}},
	})

	snapshot, err := service.Snapshot(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil || snapshot.Tenant == nil {
		t.Fatalf("expected snapshot tenant")
	}
	if snapshot.Tenant.TenantUUID != "tenant-1" {
		t.Fatalf("tenant uuid mismatch, got=%s", snapshot.Tenant.TenantUUID)
	}
	if len(snapshot.Departments) != 1 || len(snapshot.Members) != 1 || len(snapshot.Roles) != 1 || len(snapshot.Permissions) != 1 {
		t.Fatalf("snapshot list mismatch: %#v", snapshot)
	}
	if snapshot.Permissions[0].PermissionUUID != "permission-1" {
		t.Fatalf("permission uuid mismatch: %#v", snapshot.Permissions[0])
	}
}

func TestDirectoryContractServiceSnapshotError(t *testing.T) {
	expectErr := errors.New("mock list failed")
	service := NewDirectoryContractService(&stubDirectoryService{
		listFailureError: expectErr,
	})
	_, err := service.Snapshot(context.Background(), "tenant-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, expectErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}
