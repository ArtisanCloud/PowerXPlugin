// Package delegated contains the Framework-owned IAM delegated adapter.
//
// Host is deliberately a transport boundary: a generated plugin supplies the
// STS-authenticated Core HTTP client, while this package owns the stable IAM
// contracts and the one-mode adapter bundle. It never reads a plugin IAM DB.
package delegated

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
)

// Host is the complete tenant-scoped Core IAM Host Contract after transport
// authentication and stable error mapping. Implementations must derive tenant
// scope from their credential and must not add numeric-ID compatibility.
type Host interface {
	contracts.DirectoryService
	contracts.AuthzService
	contracts.IdentityContextService
}

// Adapter publishes a Host as the Framework IAM adapter bundle.
type Adapter struct{ host Host }

func NewBundle(host Host) (adapters.Bundle, error) {
	if host == nil {
		return adapters.Bundle{}, iamerrors.New(iamerrors.CodeAdapterNotBound, "delegated IAM Host client is nil")
	}
	adapter := &Adapter{host: host}
	return adapters.Bundle{Directory: adapter, Authz: adapter, Context: adapter}, nil
}

func (a *Adapter) GetTenant(ctx context.Context, tenantUUID string) (*contracts.Tenant, error) {
	return a.host.GetTenant(ctx, tenantUUID)
}
func (a *Adapter) ListDepartments(ctx context.Context, tenantUUID string) ([]contracts.Department, error) {
	return a.host.ListDepartments(ctx, tenantUUID)
}
func (a *Adapter) ListMembers(ctx context.Context, tenantUUID string) ([]contracts.Member, error) {
	return a.host.ListMembers(ctx, tenantUUID)
}
func (a *Adapter) ListMembersPage(ctx context.Context, tenantUUID string, req contracts.MemberPageRequest) (*contracts.MemberPage, error) {
	return a.host.ListMembersPage(ctx, tenantUUID, req)
}
func (a *Adapter) GetMember(ctx context.Context, tenantUUID, memberUUID string) (*contracts.Member, error) {
	return a.host.GetMember(ctx, tenantUUID, memberUUID)
}
func (a *Adapter) BatchGetMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) ([]contracts.Member, error) {
	return a.host.BatchGetMembers(ctx, tenantUUID, memberUUIDs)
}
func (a *Adapter) BatchResolveMembers(ctx context.Context, tenantUUID string, memberUUIDs []string) (*contracts.MemberResolution, error) {
	return a.host.BatchResolveMembers(ctx, tenantUUID, memberUUIDs)
}
func (a *Adapter) BatchResolveMembersByDisplayNames(ctx context.Context, tenantUUID string, displayNames []string) (*contracts.MemberDisplayNameResolution, error) {
	return a.host.BatchResolveMembersByDisplayNames(ctx, tenantUUID, displayNames)
}
func (a *Adapter) ListRoles(ctx context.Context, tenantUUID string) ([]contracts.Role, error) {
	return a.host.ListRoles(ctx, tenantUUID)
}
func (a *Adapter) ListPermissions(ctx context.Context, tenantUUID string) ([]contracts.Permission, error) {
	return a.host.ListPermissions(ctx, tenantUUID)
}
func (a *Adapter) Authorize(ctx context.Context, req contracts.AuthorizationRequest) (*contracts.AuthorizationDecision, error) {
	return a.host.Authorize(ctx, req)
}
func (a *Adapter) ResolveIdentity(ctx context.Context, bearerToken string) (*contracts.IdentityContext, error) {
	return a.host.ResolveIdentity(ctx, bearerToken)
}

var _ contracts.DirectoryService = (*Adapter)(nil)
var _ contracts.AuthzService = (*Adapter)(nil)
var _ contracts.IdentityContextService = (*Adapter)(nil)
