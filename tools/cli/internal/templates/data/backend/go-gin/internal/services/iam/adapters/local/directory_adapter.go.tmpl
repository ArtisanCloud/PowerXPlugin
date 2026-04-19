package local

import (
	"context"
	"strings"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

func (a *Adapter) GetTenant(_ context.Context, tenantUUID string) (*fwiamcontracts.Tenant, error) {
	tenantUUID = strings.TrimSpace(tenantUUID)
	if tenantUUID == "" {
		return nil, fwiamerrors.New(fwiamerrors.CodeModeInvalid, "tenant uuid is required")
	}
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID}, nil
}

func (a *Adapter) ListDepartments(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Department, error) {
	items, err := a.directory.ListDepartments(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	result := make([]fwiamcontracts.Department, 0, len(items))
	for _, item := range items {
		parentID := ""
		if item.ParentID != nil {
			parentID = strings.TrimSpace(toStringUint(*item.ParentID))
		}
		result = append(result, fwiamcontracts.Department{
			ID:         toStringUint(item.ID),
			TenantUUID: firstNonEmpty(item.TenantUUID, item.TenantUuid),
			Name:       item.Name,
			Code:       item.Code,
			ParentID:   parentID,
		})
	}
	return result, nil
}

func (a *Adapter) ListMembers(_ context.Context, _ string) ([]fwiamcontracts.Member, error) {
	// Local IAMDirectory 目前没有批量成员列表接口，US2 先保留统一读契约返回空集合。
	return []fwiamcontracts.Member{}, nil
}

func (a *Adapter) ListRoles(ctx context.Context, tenantUUID string) ([]fwiamcontracts.Role, error) {
	items, err := a.directory.ListRoles(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	result := make([]fwiamcontracts.Role, 0, len(items))
	for _, item := range items {
		result = append(result, fwiamcontracts.Role{
			ID:          toStringUint(item.ID),
			TenantUUID:  firstNonEmpty(item.TenantUUID, item.TenantUuid),
			Code:        item.Code,
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return result, nil
}

func (a *Adapter) ListPermissions(_ context.Context, _ string) ([]fwiamcontracts.Permission, error) {
	// Local IAMDirectory 目前没有全量权限查询接口，US2 先保留统一读契约返回空集合。
	return []fwiamcontracts.Permission{}, nil
}
