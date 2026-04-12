package contracts

import (
	"context"
	"strings"

	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// DirectorySnapshot 聚合目录读模型，用于跨模式统一读取。
type DirectorySnapshot struct {
	Tenant      *Tenant      `json:"tenant,omitempty"`
	Departments []Department `json:"departments,omitempty"`
	Members     []Member     `json:"members,omitempty"`
	Roles       []Role       `json:"roles,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// DirectoryContractService 提供基于 DirectoryService 的统一聚合读取入口。
type DirectoryContractService struct {
	directory DirectoryService
}

// NewDirectoryContractService 构造统一目录读取服务。
func NewDirectoryContractService(directory DirectoryService) *DirectoryContractService {
	return &DirectoryContractService{directory: directory}
}

// Snapshot 读取某租户下的目录快照。
func (s *DirectoryContractService) Snapshot(ctx context.Context, tenantUUID string) (*DirectorySnapshot, error) {
	if s == nil || s.directory == nil {
		return nil, iamerrors.New(iamerrors.CodeAdapterNotBound, "directory service is not available")
	}
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, iamerrors.New(iamerrors.CodeModeInvalid, "tenant uuid is required")
	}

	tenant, err := s.directory.GetTenant(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	departments, err := s.directory.ListDepartments(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	members, err := s.directory.ListMembers(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	roles, err := s.directory.ListRoles(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	permissions, err := s.directory.ListPermissions(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}

	return &DirectorySnapshot{
		Tenant:      tenant,
		Departments: departments,
		Members:     members,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}
