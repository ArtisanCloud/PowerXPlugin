package customer

import (
	"context"
	"strings"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
)

type BootstrapAdapter struct{}

func NewBootstrapAdapter() *BootstrapAdapter {
	return &BootstrapAdapter{}
}

func (a *BootstrapAdapter) ResolveEntry(_ context.Context, input customerfw.BootstrapInput) (*customerfw.BootstrapContext, error) {
	if tenant := strings.TrimSpace(input.Metadata["tenant_uuid"]); tenant != "" {
		return &customerfw.BootstrapContext{TenantUUID: tenant, EntryType: "tenant_hint", Channel: input.Channel}, nil
	}
	if tenant := strings.TrimSpace(input.OrgCode); tenant != "" {
		return &customerfw.BootstrapContext{TenantUUID: tenant, EntryType: "org_code", Channel: input.Channel}, nil
	}
	return nil, customerfw.NewError(customerfw.CodeCustomerBootstrapFailed, "customer bootstrap failed")
}
