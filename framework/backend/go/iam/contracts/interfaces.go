package contracts

import "context"

// DirectoryService 定义组织目录统一读能力。
type DirectoryService interface {
	GetTenant(ctx context.Context, tenantUUID string) (*Tenant, error)
	ListDepartments(ctx context.Context, tenantUUID string) ([]Department, error)
	ListMembers(ctx context.Context, tenantUUID string) ([]Member, error)
	GetMember(ctx context.Context, tenantUUID, memberUUID string) (*Member, error)
	BatchGetMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) ([]Member, error)
	// BatchResolveMembers resolves historical member references without turning
	// missing or cross-tenant references into a transport failure.
	BatchResolveMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) (*MemberResolution, error)
	ListRoles(ctx context.Context, tenantUUID string) ([]Role, error)
	ListPermissions(ctx context.Context, tenantUUID string) ([]Permission, error)
}

// AuthzService 定义统一授权判定能力。
type AuthzService interface {
	Authorize(ctx context.Context, req AuthorizationRequest) (*AuthorizationDecision, error)
}

// IdentityContextService 定义统一身份上下文解析能力。
type IdentityContextService interface {
	ResolveIdentity(ctx context.Context, bearerToken string) (*IdentityContext, error)
}
