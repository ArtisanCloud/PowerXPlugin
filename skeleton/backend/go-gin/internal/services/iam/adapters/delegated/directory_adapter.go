package delegated

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
	// delegated 模式下租户是宿主权威源，插件侧仅暴露只读投影。
	return &fwiamcontracts.Tenant{TenantUUID: tenantUUID}, nil
}

func (a *Adapter) ListDepartments(_ context.Context, _ string) ([]fwiamcontracts.Department, error) {
	// delegated 目录读取在宿主侧聚合，插件侧保持只读空投影并通过 handler 约束写操作。
	return []fwiamcontracts.Department{}, nil
}

func (a *Adapter) ListMembers(_ context.Context, _ string) ([]fwiamcontracts.Member, error) {
	return []fwiamcontracts.Member{}, nil
}

func (a *Adapter) ListRoles(_ context.Context, _ string) ([]fwiamcontracts.Role, error) {
	return []fwiamcontracts.Role{}, nil
}

func (a *Adapter) ListPermissions(_ context.Context, _ string) ([]fwiamcontracts.Permission, error) {
	return []fwiamcontracts.Permission{}, nil
}
