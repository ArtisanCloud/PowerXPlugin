package contracts

import "time"

// IAMMode 描述运行时 IAM 模式。
type IAMMode string

const (
	IAMModeLocal     IAMMode = "local"
	IAMModeDelegated IAMMode = "delegated"
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
	ID         string `json:"id"`
	TenantUUID string `json:"tenant_uuid"`
	Name       string `json:"name"`
	Code       string `json:"code,omitempty"`
	ParentID   string `json:"parent_id,omitempty"`
}

// Member 为统一成员模型。
type Member struct {
	ID          string `json:"id"`
	TenantUUID  string `json:"tenant_uuid"`
	UserID      string `json:"user_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Status      string `json:"status,omitempty"`
}

// Role 为统一角色模型。
type Role struct {
	ID          string `json:"id"`
	TenantUUID  string `json:"tenant_uuid"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Permission 为统一权限模型。
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope,omitempty"`
}

// AuthorizationRequest 描述授权判定输入。
type AuthorizationRequest struct {
	TenantUUID  string `json:"tenant_uuid"`
	UserID      string `json:"user_id,omitempty"`
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
	UserID     string `json:"user_id,omitempty"`
	Mode       string `json:"mode,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

// IdentityContext 为统一身份上下文载体。
type IdentityContext struct {
	TenantUUID  string   `json:"tenant_uuid"`
	UserID      string   `json:"user_id,omitempty"`
	MemberID    string   `json:"member_id,omitempty"`
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
