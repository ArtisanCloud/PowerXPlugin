package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type IAMAdapterMode string

const (
	IAMAdapterModeDelegated IAMAdapterMode = "delegated"
	IAMAdapterModeLocal     IAMAdapterMode = "local"
)

func (m IAMAdapterMode) String() string { return string(m) }

var (
	ErrUnsupportedMode  = errors.New("iam: unsupported mode")
	ErrAuthUnavailable  = errors.New("iam: auth service unavailable")
	ErrUnauthorized     = errors.New("iam: unauthorized")
	ErrInvalidArguments = errors.New("iam: invalid arguments")
	ErrMemberNotFound   = errors.New("iam: member not found")
)

type LoginRequest struct {
	Tenant     string
	Identifier string
	Password   string
	Remember   bool
}

type FederatedLoginRequest struct {
	TenantUUID     string
	MemberID       uint64
	Provider       string
	ExternalUserID string
}

type AuthTokens struct {
	TokenType     string
	AccessToken   string
	RefreshToken  string
	ExpiresIn     int64
	Scope         string
	ExpiresAt     time.Time
	TenantUUID    string
	PluginID      string
	PolicyVersion string
	PermsHash     string
	AuthzSource   string
}

type UserContext struct {
	TenantUUID    string
	TenantUuid    string
	TenantKey     string
	TenantName    string
	TenantID      uint64
	IsRoot        bool
	MemberID      uint64
	MemberUUID    string
	UserID        uint64
	UserUUID      string
	Username      string
	Email         string
	DisplayName   string
	AvatarURL     string
	Roles         []string
	Permissions   []string
	DepartmentIDs []uint64
	PolicyVersion string
	PermsHash     string
	AuthzSource   string
	PluginID      string
	IssuedAt      time.Time
}

type RoleInfo struct {
	ID          uint64
	UUID        string
	TenantUUID  string
	TenantUuid  string
	Code        string
	Name        string
	Description string
}

type DepartmentInfo struct {
	ID          uint64
	UUID        string
	TenantUUID  string
	TenantUuid  string
	Name        string
	Code        string
	ParentID    *uint64
	Description string
}

type TenantContext struct {
	TenantUUID  string
	TenantUuid  string
	UserUUID    string
	MemberUUID  string
	UserID      uint64
	Roles       []string
	Permissions []string
	PolicyVer   string
}

type IAMDirectory interface {
	Mode() IAMAdapterMode
	Login(ctx context.Context, req LoginRequest) (*AuthTokens, *UserContext, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	CurrentUser(ctx context.Context) (*UserContext, error)
	ListRoles(ctx context.Context, tenantUUID string) ([]RoleInfo, error)
	ListDepartments(ctx context.Context, tenantUUID string) ([]DepartmentInfo, error)
	ListMembers(ctx context.Context, tenantUUID string) ([]MemberInfo, error)
	GetMember(ctx context.Context, tenantUUID, memberUUID string) (*MemberInfo, error)
	CheckPermission(ctx context.Context, tc TenantContext, resource, action string) error
}

type MemberInfo struct {
	MemberUUID  string
	TenantUUID  string
	UserUUID    string
	DisplayName string
	Status      string
}

type MemberDisplayNameResolutionStatus string

const (
	MemberDisplayNameFound     MemberDisplayNameResolutionStatus = "found"
	MemberDisplayNameNotFound  MemberDisplayNameResolutionStatus = "not_found"
	MemberDisplayNameAmbiguous MemberDisplayNameResolutionStatus = "ambiguous"
)

// MemberDisplayNameResolutionItem is the local IAM projection used by the
// framework adapter. Member is populated only when the name has one match.
type MemberDisplayNameResolutionItem struct {
	DisplayName string
	Status      MemberDisplayNameResolutionStatus
	Member      *MemberInfo
}

// PermissionInfo is the UUID-only permission catalog projection for framework consumers.
type PermissionInfo struct {
	PermissionUUID string
	Resource       string
	Action         string
	Scope          string
}

func NormalizeMemberUUIDs(memberUUIDs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(memberUUIDs))
	result := make([]string, 0, len(memberUUIDs))
	for _, raw := range memberUUIDs {
		memberUUID := strings.TrimSpace(raw)
		if memberUUID == "" {
			return nil, fmt.Errorf("%w: member_uuid is required", ErrInvalidArguments)
		}
		if _, exists := seen[memberUUID]; exists {
			return nil, fmt.Errorf("%w: duplicate member_uuid", ErrInvalidArguments)
		}
		seen[memberUUID] = struct{}{}
		result = append(result, memberUUID)
	}
	return result, nil
}

// NormalizeMemberDisplayNames trims import values while preserving caller
// order and duplicates. Unlike member UUIDs, duplicate names are valid: each
// spreadsheet row needs its own found/not_found/ambiguous result.
func NormalizeMemberDisplayNames(displayNames []string) ([]string, error) {
	if len(displayNames) == 0 {
		return nil, fmt.Errorf("%w: display_names are required", ErrInvalidArguments)
	}
	result := make([]string, 0, len(displayNames))
	for _, raw := range displayNames {
		displayName := strings.TrimSpace(raw)
		if displayName == "" {
			return nil, fmt.Errorf("%w: display_name is required", ErrInvalidArguments)
		}
		result = append(result, displayName)
	}
	return result, nil
}
