package middleware

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	common "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/middleware"
)

const tenantHeader = "tenant_uuid"

// TenantAdminGuard enforces tenant admin role + optional permissions and ensures tenant header present.
func TenantAdminGuard(requiredPerms ...string) bootstrap.Middleware {
	base := common.RBACGuard(common.GuardOptions{
		AllowedRoles:        []common.Role{common.RoleTenantAdmin},
		RequiredPermissions: requiredPerms,
	})
	return func(next bootstrap.Handler) bootstrap.Handler {
		wrapped := base(func(ctx bootstrap.Context) {
			if ctx.Header(tenantHeader) == "" {
				router.RespondError(ctx, http.StatusBadRequest, "TENANT_ID_REQUIRED", "tenant header is required", nil)
				return
			}
			next(ctx)
		})
		return wrapped
	}
}

// ReviewerGuard reuses common guard for Marketplace reviewers handling online/ offline approvals.
func ReviewerGuard(requiredPerms ...string) bootstrap.Middleware {
	return common.RBACGuard(common.GuardOptions{
		AllowedRoles:        []common.Role{common.RoleReviewer, common.RoleOps},
		RequiredPermissions: requiredPerms,
	})
}
