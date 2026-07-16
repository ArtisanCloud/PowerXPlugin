namespace PowerXPlugin.Framework.IAM.Models;

public record Tenant
{
    public string UUID { get; init; } = "";
    public string Name { get; init; } = "";
    public string Key { get; init; } = "";
    public int Status { get; init; } = 1;
    public string Plan { get; init; } = "free";
}

public record Department
{
    public string UUID { get; init; } = "";
    public string Name { get; init; } = "";
    public string? ParentUUID { get; init; }
    public string TenantUUID { get; init; } = "";
    public DateTime CreatedAt { get; init; }
    public DateTime UpdatedAt { get; init; }
}

public record Member
{
    public string UUID { get; init; } = "";
    public string TenantUUID { get; init; } = "";
    public string UserID { get; init; } = "";
    public string Username { get; init; } = "";
    public string? DisplayName { get; init; }
    public int Status { get; init; } = 1;
    public DateTime CreatedAt { get; init; }
    public DateTime UpdatedAt { get; init; }
}

public record Role
{
    public string UUID { get; init; } = "";
    public string Name { get; init; } = "";
    public string Code { get; init; } = "";
    public string? Description { get; init; }
    public List<Permission> Permissions { get; init; } = new();
}

public record Permission
{
    public string UUID { get; init; } = "";
    public string Name { get; init; } = "";
    public string Code { get; init; } = "";
    public string Resource { get; init; } = "";
    public string Action { get; init; } = "";
    public string? Description { get; init; }
}

public record AuthorizationRequest
{
    public string TenantUUID { get; init; } = "";
    public string UserID { get; init; } = "";
    public string Resource { get; init; } = "";
    public string Action { get; init; } = "";
    public string? TraceID { get; init; }
    public string? PolicyToken { get; init; }
}

public record AuthorizationDecision
{
    public bool Allowed { get; init; }
    public string? ReasonCode { get; init; }
    public string Resource { get; init; } = "";
    public string Action { get; init; } = "";
    public string TenantUUID { get; init; } = "";
    public string UserID { get; init; } = "";
    public string Mode { get; init; } = "";
    public string? TraceID { get; init; }
}

public record IdentityContext
{
    public string? TenantUUID { get; init; }
    public string? UserID { get; init; }
    public string? MemberID { get; init; }
    public List<string> Roles { get; init; } = new();
    public List<string> Permissions { get; init; } = new();
    public string? PolicyVer { get; init; }
    public string? TraceID { get; init; }
}

public record DirectorySnapshot
{
    public Tenant? Tenant { get; init; }
    public IReadOnlyList<Department> Departments { get; init; } = Array.Empty<Department>();
    public IReadOnlyList<Member> Members { get; init; } = Array.Empty<Member>();
    public IReadOnlyList<Role> Roles { get; init; } = Array.Empty<Role>();
    public IReadOnlyList<Permission> Permissions { get; init; } = Array.Empty<Permission>();
}

public enum IAMMode
{
    Local,
    Delegated
}
