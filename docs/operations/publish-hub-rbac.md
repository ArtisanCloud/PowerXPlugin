# RBAC Integration Guide

## Overview

The Publish Hub uses Role-Based Access Control (RBAC) to secure API endpoints and ensure proper authorization. This guide describes how to use the RBAC system.

## Components

### 1. AuthService

The `AuthService` provides user authentication and permission checking:

```go
authService := services.NewAuthService(logger)
```

#### Demo Users

The service includes demo users for testing:

| UserID | Role | Permissions |
|--------|------|-------------|
| dev1 | plugin_developer | publish:submit, publish:view, plugin:view, system:view_logs |
| reviewer1 | marketplace_reviewer | publish:approve, publish:reject, publish:view, marketplace:review, system:view_logs |
| tenant_admin1 | tenant_admin | plugin:install, plugin:rollback, plugin:view, plugin:manage, marketplace:offline, system:view_logs, system:metrics |
| ops1 | platform_ops | admin:configure, admin:view, plugin:view, publish:view, marketplace:review, system:view_logs, system:metrics |

### 2. RBACGuard Middleware

The `RBACGuard` middleware enforces authorization requirements:

```go
middleware.RBACGuard(middleware.GuardOptions{
    AllowedRoles: []middleware.Role{middleware.RoleTenantAdmin},
    RequiredPermissions: []string{services.PermPluginInstall},
    AuthService: authService,
    Resource: "plugin",
    Action: "install",
})
```

## API Headers

Clients must include these headers:

- `X-Powerx-User-Id`: The authenticated user ID
- `X-Powerx-Role`: The user's role
- `X-Powerx-Permissions`: Comma-separated list of permissions (optional if using AuthService)

Example:
```
X-Powerx-User-Id: tenant_admin1
X-Powerx-Role: tenant_admin
X-Powerx-Permissions: plugin:install,plugin:rollback
```

## Endpoint Protection Examples

### Plugin Management API

```go
// Plugin install endpoint - requires tenant_admin role
router.Handle("POST", "/admin/plugins/install",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermPluginInstall},
        AuthService: authService,
        Resource: "plugin",
        Action: "install",
    }),
    handlers.InstallPlugin,
)

// Plugin rollback endpoint
router.Handle("POST", "/admin/plugins/:id/rollback",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermPluginRollback},
        AuthService: authService,
        Resource: "plugin",
        Action: "rollback",
    }),
    handlers.RollbackPlugin,
)

// View plugins - allows developers and tenant admins
router.Handle("GET", "/plugins",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleDeveloper, middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermPluginView},
        AuthService: authService,
        Resource: "plugin",
        Action: "view",
    }),
    handlers.ListPlugins,
)
```

### Publish API

```go
// Submit plugin for review - developers only
router.Handle("POST", "/publish/submit",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleDeveloper},
        RequiredPermissions: []string{services.PermPublishSubmit},
        AuthService: authService,
        Resource: "publish",
        Action: "submit",
    }),
    handlers.SubmitPublish,
)

// Approve/reject publish - reviewers only
router.Handle("POST", "/publish/:id/approve",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleReviewer},
        RequiredPermissions: []string{services.PermPublishApprove},
        AuthService: authService,
        Resource: "publish",
        Action: "approve",
    }),
    handlers.ApprovePublish,
)

router.Handle("POST", "/publish/:id/reject",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleReviewer},
        RequiredPermissions: []string{services.PermPublishReject},
        AuthService: authService,
        Resource: "publish",
        Action: "reject",
    }),
    handlers.RejectPublish,
)
```

### Marketplace API

```go
// Review marketplace submission - reviewers only
router.Handle("POST", "/marketplace/review",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleReviewer},
        RequiredPermissions: []string{services.PermMarketplaceReview},
        AuthService: authService,
        Resource: "marketplace",
        Action: "review",
    }),
    handlers.ReviewMarketplace,
)

// Offline package review
router.Handle("POST", "/marketplace/offline/review",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleReviewer, middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermMarketplaceOffline},
        AuthService: authService,
        Resource: "marketplace",
        Action: "offline_review",
    }),
    handlers.OfflineReview,
)
```

### Admin API

```go
// Admin configuration - ops only
router.Handle("POST", "/admin/config",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleOps},
        RequiredPermissions: []string{services.PermAdminConfigure},
        AuthService: authService,
        Resource: "admin",
        Action: "configure",
    }),
    handlers.ConfigureAdmin,
)

// View system metrics - ops and tenant admins
router.Handle("GET", "/admin/metrics",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleOps, middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermSystemMetrics},
        AuthService: authService,
        Resource: "admin",
        Action: "view_metrics",
    }),
    handlers.ViewMetrics,
)
```

### Dev API

```go
// Register dev session - developers only
router.Handle("POST", "/internal/dev/plugins/register",
    middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleDeveloper},
        RequiredPermissions: []string{services.PermPluginView},
        AuthService: authService,
        Resource: "dev_plugin",
        Action: "register",
    }),
    handlers.RegisterDevPlugin,
)
```

## Permission Constants

Available permissions:

**Plugin Permissions:**
- `plugin:install` - Install plugins
- `plugin:rollback` - Rollback plugin versions
- `plugin:view` - View plugin information
- `plugin:manage` - Manage plugin versions

**Publish Permissions:**
- `publish:submit` - Submit plugins for review
- `publish:approve` - Approve published plugins
- `publish:reject` - Reject published plugins
- `publish:view` - View publish status

**Marketplace Permissions:**
- `marketplace:review` - Review marketplace submissions
- `marketplace:offline` - Handle offline packages

**Admin Permissions:**
- `admin:configure` - Configure system settings
- `admin:view` - View admin panel

**System Permissions:**
- `system:view_logs` - View system logs
- `system:metrics` - View system metrics

## Audit Logging

All access attempts are logged for security auditing:

```go
// Access granted
2024-01-15T10:30:00Z INFO rbac access granted userId=tenant_admin1 role=tenant_admin resource=plugin action=install

// Access denied
2024-01-15T10:30:00Z WARN rbac denied userId=dev1 reason=missing_permission_plugin:install
```

## Production Considerations

1. **Database Integration**: Replace the in-memory user store with a real database
2. **Password Validation**: Implement proper password hashing and validation
3. **Token-Based Auth**: Integrate with JWT or OAuth2 for stateless authentication
4. **HTTPS Only**: Ensure all API endpoints use HTTPS in production
5. **Rate Limiting**: Add rate limiting to prevent brute force attacks
6. **Session Management**: Implement proper session management
7. **Audit Trail**: Store all access logs in a persistent audit trail
8. **Password Policies**: Enforce strong password requirements
9. **Account Lockout**: Implement account lockout after failed attempts
10. **Regular Reviews**: Regularly review and audit user permissions

## Testing

### Unit Tests

```go
// Test permission checking
func TestCheckPermission(t *testing.T) {
    authService := services.NewAuthService(nil)

    if !authService.CheckPermission("tenant_admin1", services.PermPluginInstall) {
        t.Error("tenant_admin1 should have plugin:install permission")
    }

    if authService.CheckPermission("dev1", services.PermPluginRollback) {
        t.Error("dev1 should not have plugin:rollback permission")
    }
}
```

### Integration Tests

```go
// Test RBAC middleware
func TestRBACMiddleware(t *testing.T) {
    authService := services.NewAuthService(nil)

    handler := middleware.RBACGuard(middleware.GuardOptions{
        AllowedRoles: []middleware.Role{middleware.RoleTenantAdmin},
        RequiredPermissions: []string{services.PermPluginInstall},
        AuthService: authService,
        Resource: "plugin",
        Action: "install",
    })

    // Test with valid credentials
    // Test with invalid role
    // Test with missing permissions
}
```

## Troubleshooting

### Common Issues

1. **401 Unauthorized**: Missing or invalid `X-Powerx-User-Id` header
2. **403 Forbidden**: Role or permission requirements not met
3. **User Not Found**: User ID exists in request but not in auth service
4. **Permission Denied**: User lacks required permission for the action

### Debugging

Enable debug logging to see detailed authorization decisions:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

This will show:
- User authentication attempts
- Role checks
- Permission verification
- Access grant/deny decisions
