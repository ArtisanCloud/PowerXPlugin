package iam

type TenantListQuery struct {
	Status   string `form:"status"`
	Query    string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type CreateTenantRequest struct {
	Key    string `json:"key" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
}

type UpdateTenantRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
}

type DepartmentListQuery struct {
	TenantUUID string `form:"tenant_uuid"`
}

type CreateDepartmentRequest struct {
	TenantUUID           string  `json:"tenant_uuid"`
	Name                 string  `json:"name" binding:"required"`
	Code                 string  `json:"code"`
	ParentDepartmentUUID *string `json:"parent_department_uuid"`
	Description          string  `json:"description"`
	SortOrder            int     `json:"sort_order"`
}

type UpdateDepartmentRequest struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	SortOrder            *int    `json:"sort_order"`
	ParentDepartmentUUID *string `json:"parent_department_uuid"`
}

type MemberListQuery struct {
	TenantUUID string `form:"tenant_uuid"`
	Status     string `form:"status"`
	Query      string `form:"q"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type CreateMemberRequest struct {
	TenantUUID      string   `json:"tenant_uuid"`
	Email           string   `json:"email" binding:"required"`
	DisplayName     string   `json:"display_name"`
	Username        string   `json:"username"`
	Phone           string   `json:"phone"`
	DepartmentUUIDs []string `json:"department_uuids"`
	Status          string   `json:"status"`
	RoleUUIDs       []string `json:"role_uuids"`
}

type UpdateMemberRequest struct {
	DisplayName     string   `json:"display_name"`
	Status          string   `json:"status"`
	DepartmentUUIDs []string `json:"department_uuids"`
	RoleUUIDs       []string `json:"role_uuids"`
	ReplaceRoles    bool     `json:"replace_roles"`
}

type BulkImportMemberEntry struct {
	Email           string   `json:"email" binding:"required,email"`
	DisplayName     string   `json:"display_name"`
	Username        string   `json:"username"`
	Phone           string   `json:"phone"`
	DepartmentUUIDs []string `json:"department_uuids"`
	Status          string   `json:"status"`
	RoleUUIDs       []string `json:"role_uuids"`
}

type BulkImportMembersRequest struct {
	TenantUUID string                  `json:"tenant_uuid"`
	Users      []BulkImportMemberEntry `json:"users" binding:"required,min=1"`
}

type RoleListQuery struct {
	TenantUUID string `form:"tenant_uuid"`
	Query      string `form:"q"`
	ScopeType  string `form:"scope_type"`
}

type CreateRoleRequest struct {
	TenantUUID      string   `json:"tenant_uuid"`
	Code            string   `json:"code" binding:"required"`
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	ScopeType       string   `json:"scope_type"`
	CloneRoleUUID   *string  `json:"clone_role_uuid"`
	PermissionUUIDs []string `json:"permission_uuids"`
	MemberUUIDs     []string `json:"member_uuids"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ScopeType   string `json:"scope_type"`
}

type ReplaceRolePermissionsRequest struct {
	TenantUUID      string   `json:"tenant_uuid"`
	PermissionUUIDs []string `json:"permission_uuids"`
}

type RoleMembersRequest struct {
	TenantUUID  string   `json:"tenant_uuid"`
	MemberUUIDs []string `json:"member_uuids"`
}
