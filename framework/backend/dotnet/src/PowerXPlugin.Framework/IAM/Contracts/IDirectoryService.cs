using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM.Contracts;

public interface IDirectoryService
{
    Task<Tenant?> GetTenant(string tenantUuid, CancellationToken ct = default);
    Task<IReadOnlyList<Department>> ListDepartments(string tenantUuid, CancellationToken ct = default);
    Task<IReadOnlyList<Member>> ListMembers(string tenantUuid, CancellationToken ct = default);
    Task<IReadOnlyList<Role>> ListRoles(string tenantUuid, CancellationToken ct = default);
    Task<IReadOnlyList<Permission>> ListPermissions(string tenantUuid, CancellationToken ct = default);
}

public interface IAuthzService
{
    Task<AuthorizationDecision?> AuthorizeAsync(AuthorizationRequest request, CancellationToken ct = default);
}

public interface IIdentityContextService
{
    Task<IdentityContext?> ResolveIdentity(string? bearerToken, CancellationToken ct = default);
}
