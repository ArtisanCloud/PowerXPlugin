package customerfw

import "context"

type BootstrapInput struct {
	Scene      string            `json:"scene,omitempty"`
	InviteCode string            `json:"invite_code,omitempty"`
	OrgCode    string            `json:"org_code,omitempty"`
	Channel    string            `json:"channel,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type BootstrapContext struct {
	TenantUUID string         `json:"tenant_uuid"`
	OrgUUID    string         `json:"org_uuid,omitempty"`
	EntryType  string         `json:"entry_type,omitempty"`
	Campaign   string         `json:"campaign,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type MiniAppBootstrapClient interface {
	ResolveEntry(ctx context.Context, input BootstrapInput) (*BootstrapContext, error)
}

type BootstrapResolver func(ctx context.Context, input BootstrapInput) (*BootstrapContext, error)

func (f BootstrapResolver) ResolveEntry(ctx context.Context, input BootstrapInput) (*BootstrapContext, error) {
	return f(ctx, input)
}

func NormalizeBootstrapContext(bc *BootstrapContext) *BootstrapContext {
	if bc == nil {
		return nil
	}
	copy := *bc
	copy.TenantUUID = normalizeID(copy.TenantUUID)
	copy.OrgUUID = normalizeID(copy.OrgUUID)
	return &copy
}
