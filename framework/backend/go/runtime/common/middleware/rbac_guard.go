package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// Role enumerates Publish Hub actor types.
type Role string

const (
	RoleDeveloper     Role = "plugin_developer"
	RoleReviewer      Role = "marketplace_reviewer"
	RoleTenantAdmin   Role = "tenant_admin"
	RoleOps           Role = "platform_ops"
	roleHeader             = "X-Powerx-Role"
	permissionsHeader      = "X-Powerx-Permissions"
)

// GuardOptions drive RBACGuard behaviour.
type GuardOptions struct {
	AllowedRoles        []Role
	RequiredPermissions []string
	AuditLogger         *slog.Logger
}

// RBACGuard enforces role + permission requirements using request headers.
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
			roleValue := strings.TrimSpace(ctx.Header(roleHeader))
			role := Role(strings.ToLower(roleValue))
			if len(allowed) > 0 {
				if _, ok := allowed[role]; !ok {
					recordDeny(opts.AuditLogger, roleValue, "role_not_allowed")
					router.RespondError(ctx, http.StatusForbidden, "RBAC_DENY", "role not allowed", map[string]any{"expected": opts.AllowedRoles})
					return
				}
			}

			if len(requiredPerms) > 0 {
				provided := parseCSV(ctx.Header(permissionsHeader))
				if !containsAll(provided, requiredPerms) {
					recordDeny(opts.AuditLogger, roleValue, "missing_permission")
					router.RespondError(ctx, http.StatusForbidden, "RBAC_PERMISSION_DENY", "missing required permission", map[string]any{"required": requiredPerms})
					return
				}
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
