using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.IAM.Models;

public record Tenant
{
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("tenant_key")] public string TenantKey { get; init; } = "";
    [JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonPropertyName("status")] public string Status { get; init; } = "";
}

public record Department
{
    [JsonPropertyName("department_uuid")] public string DepartmentUUID { get; init; } = "";
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonPropertyName("code")] public string Code { get; init; } = "";
    [JsonPropertyName("parent_department_uuid")] public string? ParentDepartmentUUID { get; init; }
}

public record Member
{
    [JsonPropertyName("member_uuid")] public string MemberUUID { get; init; } = "";
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("user_uuid")] public string UserUUID { get; init; } = "";
    [JsonPropertyName("display_name")] public string? DisplayName { get; init; }
    [JsonPropertyName("status")] public string Status { get; init; } = "";
}

public record Role
{
    [JsonPropertyName("role_uuid")] public string RoleUUID { get; init; } = "";
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("code")] public string Code { get; init; } = "";
    [JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonPropertyName("description")] public string? Description { get; init; }
}

public record Permission
{
    [JsonPropertyName("permission_uuid")] public string PermissionUUID { get; init; } = "";
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("scope")] public string? Scope { get; init; }
}

public record AuthorizationRequest
{
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("user_uuid")] public string? UserUUID { get; init; }
    [JsonPropertyName("member_uuid")] public string? MemberUUID { get; init; }
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("trace_id")] public string? TraceID { get; init; }
    [JsonPropertyName("policy_token")] public string? PolicyToken { get; init; }
}

public record AuthorizationDecision
{
    [JsonPropertyName("allowed")] public bool Allowed { get; init; }
    [JsonPropertyName("reason_code")] public string? ReasonCode { get; init; }
    [JsonPropertyName("resource")] public string Resource { get; init; } = "";
    [JsonPropertyName("action")] public string Action { get; init; } = "";
    [JsonPropertyName("tenant_uuid")] public string TenantUUID { get; init; } = "";
    [JsonPropertyName("user_uuid")] public string? UserUUID { get; init; }
    [JsonPropertyName("member_uuid")] public string? MemberUUID { get; init; }
    [JsonPropertyName("mode")] public string Mode { get; init; } = "";
    [JsonPropertyName("trace_id")] public string? TraceID { get; init; }
}

public record IdentityContext
{
    [JsonPropertyName("tenant_uuid")] public string? TenantUUID { get; init; }
    [JsonPropertyName("user_uuid")] public string? UserUUID { get; init; }
    [JsonPropertyName("member_uuid")] public string? MemberUUID { get; init; }
    [JsonPropertyName("roles")] public List<string> Roles { get; init; } = [];
    [JsonPropertyName("permissions")] public List<string> Permissions { get; init; } = [];
    [JsonPropertyName("policy_ver")] public string? PolicyVer { get; init; }
    [JsonPropertyName("trace_id")] public string? TraceID { get; init; }
}

public enum IAMAdapterMode
{
    Local,
    Delegated
}
