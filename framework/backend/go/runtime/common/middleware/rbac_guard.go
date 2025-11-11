package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/services"
)

// Role enumerates Publish Hub actor types.
type Role string

const (
	RoleDeveloper     Role = "plugin_developer"
	RoleReviewer      Role = "marketplace_reviewer"
	RoleTenantAdmin   Role = "tenant_admin"
	RoleOps           Role = "platform_ops"

	roleHeader       = "X-Powerx-Role"
	permissionsHeader = "X-Powerx-Permissions"
	userIdHeader      = "X-Powerx-User-Id"
)

// GuardOptions drive RBACGuard behaviour.
type GuardOptions struct {
	AllowedRoles        []Role
	RequiredPermissions []string
	AuditLogger         *slog.Logger
	AuthService         *services.AuthService
	Resource            string
	Action              string
}

// RBACGuard enforces role + permission requirements with real auth service integration.
func RBACGuard(opts GuardOptions) bootstrap.Middleware {
	allowed := make(map[Role]struct{}, len(opts.AllowedRoles))
	for _, r := range opts.AllowedRoles {
		allowed[r] = struct{}{}
	}
	requiredPerms := make([]string, 0, len(opts.RequiredPermissions))
	for _, p := range opts.RequiredPermissions {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			requiredPerms = append(requiredPerms, trimmed)
		}
	}

	return func(next bootstrap.Handler) bootstrap.Handler {
		return func(ctx bootstrap.Context) {
			// Get user identity from headers
			userID := strings.TrimSpace(ctx.Header(userIdHeader))
			roleValue := strings.TrimSpace(ctx.Header(roleHeader))
			role := Role(strings.ToLower(roleValue))

			// Validate authentication (userId must be present)
			if userID == "" {
				recordDeny(opts.AuditLogger, "unknown", "missing_user_id")
				router.RespondError(ctx, http.StatusUnauthorized, "RBAC_DENY", "authentication required", nil)
				return
			}

			// Verify user exists in auth service
			if opts.AuthService != nil {
				_, err := opts.AuthService.GetUserByID(userID)
				if err != nil {
					recordDeny(opts.AuditLogger, userID, "user_not_found")
					router.RespondError(ctx, http.StatusUnauthorized, "RBAC_DENY", "invalid user", nil)
					return
				}
			}

			// Check role requirements
			if len(allowed) > 0 {
				if _, ok := allowed[role]; !ok {
					recordDeny(opts.AuditLogger, userID, "role_not_allowed")
					router.RespondError(ctx, http.StatusForbidden, "RBAC_DENY", "role not allowed", map[string]any{"expected": opts.AllowedRoles})
					return
				}
			}

			// Check permission requirements
			if len(requiredPerms) > 0 {
				// Check with auth service if available
				if opts.AuthService != nil {
					for _, perm := range requiredPerms {
						if !opts.AuthService.CheckPermission(userID, perm) {
							recordDeny(opts.AuditLogger, userID, "missing_permission_"+perm)
							router.RespondError(ctx, http.StatusForbidden, "RBAC_PERMISSION_DENY", "missing required permission", map[string]any{"required": perm})
							return
						}
					}
				} else {
					// Fallback to header-based permissions
					provided := parseCSV(ctx.Header(permissionsHeader))
					if !containsAll(provided, requiredPerms) {
						recordDeny(opts.AuditLogger, userID, "missing_permission")
						router.RespondError(ctx, http.StatusForbidden, "RBAC_PERMISSION_DENY", "missing required permission", map[string]any{"required": requiredPerms})
						return
					}
				}
			}

			// Record successful access
			if opts.AuthService != nil {
				opts.AuthService.RecordAccess(userID, opts.Resource, opts.Action, true)
			}

			// Deny logging for successful access (for audit trail)
			if opts.AuditLogger != nil {
				opts.AuditLogger.Info("rbac access granted",
					slog.String("userId", userID),
					slog.String("role", string(role)),
					slog.String("resource", opts.Resource),
					slog.String("action", opts.Action),
				)
			}

			next(ctx)
		}
	}
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, strings.ToLower(trimmed))
		}
	}
	return result
}

func containsAll(have []string, need []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, v := range have {
		set[v] = struct{}{}
	}
	for _, v := range need {
		if _, ok := set[strings.ToLower(v)]; !ok {
			return false
		}
	}
	return true
}

func recordDeny(logger *slog.Logger, role string, reason string) {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Warn("rbac denied", slog.String("role", role), slog.String("reason", reason))
}
