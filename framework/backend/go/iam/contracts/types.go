package contracts

import "time"

// IAMAdapterMode 描述运行时 IAM 模式。
type IAMAdapterMode string

const (
	IAMAdapterModeLocal     IAMAdapterMode = "local"
	IAMAdapterModeDelegated IAMAdapterMode = "delegated"
)

// Tenant 为统一租户模型。
type Tenant struct {
	TenantUUID string `json:"tenant_uuid"`
	TenantKey  string `json:"tenant_key,omitempty"`
	Name       string `json:"name,omitempty"`
	Status     string `json:"status,omitempty"`
}

// Department 为统一部门模型。
type Department struct {
	DepartmentUUID       string `json:"department_uuid"`
	TenantUUID           string `json:"tenant_uuid"`
	Name                 string `json:"name"`
	Code                 string `json:"code,omitempty"`
	ParentDepartmentUUID string `json:"parent_department_uuid,omitempty"`
}

// Member 为统一成员模型。
type Member struct {
	MemberUUID  string `json:"member_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
	UserUUID    string `json:"user_uuid,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Status      string `json:"status,omitempty"`
}

// MemberPageRequest is the explicit bounded member-directory query shared by
// local and delegated IAM modes. Tenant identity remains a separate argument
// and is never accepted from a Host request body.
type MemberPageRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// MemberPage is a stable UUID-only member-directory page.
type MemberPage struct {
	Items    []Member `json:"items"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Total    int64    `json:"total"`
}

// MemberResolution is the tolerant batch directory result used by historical
// audit readers. Missing UUIDs are explicit; authentication, authorization,
// validation, and upstream failures remain errors.
type MemberResolution struct {
	Items              []Member `json:"items"`
	MissingMemberUUIDs []string `json:"missing_member_uuids"`
}

// MemberDisplayNameResolutionStatus describes the only per-item outcomes of
// a tenant-scoped exact display-name lookup.
type MemberDisplayNameResolutionStatus string

const (
	MemberDisplayNameResolutionFound     MemberDisplayNameResolutionStatus = "found"
	MemberDisplayNameResolutionNotFound  MemberDisplayNameResolutionStatus = "not_found"
	MemberDisplayNameResolutionAmbiguous MemberDisplayNameResolutionStatus = "ambiguous"
)

// MemberDisplayNameResolution keeps one result for each requested display
// name. A Member is present only for a unique match; ambiguous responses do
// not reveal candidate identities.
type MemberDisplayNameResolution struct {
	Items []MemberDisplayNameResolutionItem `json:"items"`
}

type MemberDisplayNameResolutionItem struct {
	DisplayName string                            `json:"display_name"`
	Status      MemberDisplayNameResolutionStatus `json:"status"`
	Member      *Member                           `json:"member,omitempty"`
}

// Role 为统一角色模型。
type Role struct {
	RoleUUID    string `json:"role_uuid"`
	TenantUUID  string `json:"tenant_uuid"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Permission 为统一权限模型。
type Permission struct {
	PermissionUUID string `json:"permission_uuid"`
	Resource       string `json:"resource"`
	Action         string `json:"action"`
	Scope          string `json:"scope,omitempty"`
}

// AuthorizationRequest 描述授权判定输入。
type AuthorizationRequest struct {
	TenantUUID  string `json:"tenant_uuid"`
	UserUUID    string `json:"user_uuid,omitempty"`
	MemberUUID  string `json:"member_uuid,omitempty"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	TraceID     string `json:"trace_id,omitempty"`
	PolicyToken string `json:"policy_token,omitempty"`
}

// AuthorizationDecision 描述授权判定输出。
type AuthorizationDecision struct {
	Allowed    bool   `json:"allowed"`
	ReasonCode string `json:"reason_code,omitempty"`
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	TenantUUID string `json:"tenant_uuid,omitempty"`
	UserUUID   string `json:"user_uuid,omitempty"`
	MemberUUID string `json:"member_uuid,omitempty"`
	Mode       string `json:"mode,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

// IdentityContext 为统一身份上下文载体。
type IdentityContext struct {
	TenantUUID  string   `json:"tenant_uuid"`
	UserUUID    string   `json:"user_uuid,omitempty"`
	MemberUUID  string   `json:"member_uuid,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	PolicyVer   string   `json:"policy_version,omitempty"`
	TraceID     string   `json:"trace_id,omitempty"`
}

// ModeResolutionAudit 描述模式解析审计字段。
type ModeResolutionAudit struct {
	Source      string    `json:"source"`
	ResolvedAt  time.Time `json:"resolved_at"`
	Environment string    `json:"environment,omitempty"`
}
