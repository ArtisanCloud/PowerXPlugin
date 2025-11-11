package services

import (
	"log/slog"
	"time"
)

// AuthUser represents a user in the system
type AuthUser struct {
	UserID     string            `json:"userId"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Roles      []string          `json:"roles"`
	Permissions []string         `json:"permissions"`
	TenantID   string            `json:"tenantId"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// Permission constants for Publish Hub
const (
	// Plugin permissions
	PermPluginInstall    = "plugin:install"
	PermPluginRollback   = "plugin:rollback"
	PermPluginView       = "plugin:view"
	PermPluginManage     = "plugin:manage"

	// Publish permissions
	PermPublishSubmit    = "publish:submit"
	PermPublishApprove   = "publish:approve"
	PermPublishReject    = "publish:reject"
	PermPublishView      = "publish:view"

	// Marketplace permissions
	PermMarketplaceReview = "marketplace:review"
	PermMarketplaceOffline = "marketplace:offline"

	// Admin permissions
	PermAdminConfigure = "admin:configure"
	PermAdminView      = "admin:view"

	// System permissions
	PermSystemViewLogs = "system:view_logs"
	PermSystemMetrics  = "system:metrics"
)

// RolePermissions maps roles to their default permissions
var RolePermissions = map[string][]string{
	"plugin_developer": {
		PermPublishSubmit,
		PermPublishView,
		PermPluginView,
		PermSystemViewLogs,
	},
	"marketplace_reviewer": {
		PermPublishApprove,
		PermPublishReject,
		PermPublishView,
		PermMarketplaceReview,
		PermSystemViewLogs,
	},
	"tenant_admin": {
		PermPluginInstall,
		PermPluginRollback,
		PermPluginView,
		PermPluginManage,
		PermMarketplaceOffline,
		PermSystemViewLogs,
		PermSystemMetrics,
	},
	"platform_ops": {
		PermAdminConfigure,
		PermAdminView,
		PermPluginView,
		PermPublishView,
		PermMarketplaceReview,
		PermSystemViewLogs,
		PermSystemMetrics,
	},
}

// AuthService provides authentication and authorization services
type AuthService struct {
	logger *slog.Logger
	// In production, this would be a database or external auth provider
	users map[string]*AuthUser
}

// NewAuthService creates a new AuthService with default demo users
func NewAuthService(logger *slog.Logger) *AuthService {
	if logger == nil {
		logger = slog.Default()
	}

	// Initialize with demo users
	users := map[string]*AuthUser{
		"dev1": {
			UserID:      "dev1",
			Username:    "developer1",
			Email:       "dev1@example.com",
			Roles:       []string{"plugin_developer"},
			Permissions: RolePermissions["plugin_developer"],
			TenantID:    "tenant_dev",
			Metadata:    map[string]string{"department": "engineering"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		"reviewer1": {
			UserID:      "reviewer1",
			Username:    "reviewer1",
			Email:       "reviewer1@example.com",
			Roles:       []string{"marketplace_reviewer"},
			Permissions: RolePermissions["marketplace_reviewer"],
			TenantID:    "tenant_marketplace",
			Metadata:    map[string]string{"team": "security"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		"tenant_admin1": {
			UserID:      "tenant_admin1",
			Username:    "tenant_admin1",
			Email:       "admin1@tenant.com",
			Roles:       []string{"tenant_admin"},
			Permissions: RolePermissions["tenant_admin"],
			TenantID:    "tenant_001",
			Metadata:    map[string]string{"tier": "enterprise"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		"ops1": {
			UserID:      "ops1",
			Username:    "ops1",
			Email:       "ops@example.com",
			Roles:       []string{"platform_ops"},
			Permissions: RolePermissions["platform_ops"],
			TenantID:    "platform",
			Metadata:    map[string]string{"shift": "primary"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	return &AuthService{
		logger: logger,
		users:  users,
	}
}

// AuthenticateUser validates user credentials and returns user info
// In production, this would validate JWT tokens, API keys, or other auth mechanisms
func (s *AuthService) AuthenticateUser(username, password string) (*AuthUser, error) {
	// Demo: any password works for demo users
	user, exists := s.users[username]
	if !exists {
		return nil, &AuthError{Code: AuthErrUserNotFound, Message: "user not found"}
	}

	// In production, validate password hash here
	s.logger.Info("user authenticated", slog.String("userId", user.UserID), slog.String("username", user.Username))
	return user, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(userID string) (*AuthUser, error) {
	for _, user := range s.users {
		if user.UserID == userID {
			return user, nil
		}
	}
	return nil, &AuthError{Code: AuthErrUserNotFound, Message: "user not found"}
}

// CheckPermission verifies if a user has a specific permission
func (s *AuthService) CheckPermission(userID, permission string) bool {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false
	}

	for _, perm := range user.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// CheckRole verifies if a user has a specific role
func (s *AuthService) CheckRole(userID, role string) bool {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false
	}

	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// GetUserPermissions returns all permissions for a user
func (s *AuthService) GetUserPermissions(userID string) []string {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil
	}
	return user.Permissions
}

// RecordAccess logs access attempts for audit purposes
func (s *AuthService) RecordAccess(userID, resource, action string, granted bool) {
	if granted {
		s.logger.Info("access granted",
			slog.String("userId", userID),
			slog.String("resource", resource),
			slog.String("action", action),
		)
	} else {
		s.logger.Warn("access denied",
			slog.String("userId", userID),
			slog.String("resource", resource),
			slog.String("action", action),
		)
	}
}

// AuthError represents authentication errors
type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// Auth error codes
const (
	AuthErrUserNotFound   = "USER_NOT_FOUND"
	AuthErrInvalidCreds   = "INVALID_CREDENTIALS"
	AuthErrTokenExpired   = "TOKEN_EXPIRED"
	AuthErrPermissionDenied = "PERMISSION_DENIED"
)
