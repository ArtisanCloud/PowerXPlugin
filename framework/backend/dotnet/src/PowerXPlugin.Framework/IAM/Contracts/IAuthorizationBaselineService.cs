using System.Text.Json.Serialization;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.IAM.Contracts;

public sealed record AuthorizationBaselineQuery(
    [property: JsonIgnore] string TenantUuid,
    [property: JsonPropertyName("member_uuids")] IReadOnlyList<string> MemberUuids,
    [property: JsonPropertyName("permission_codes")] IReadOnlyList<string> PermissionCodes)
{
    public AuthorizationBaselineQuery Normalize()
    {
        if (!Guid.TryParse(TenantUuid, out var tenant) || tenant == Guid.Empty
            || MemberUuids is null || MemberUuids.Count is < 1 or > 100
            || PermissionCodes is null || PermissionCodes.Count is < 1 or > 10)
            throw new IAMAdapterException("IAM_BASELINE_INVALID_ARGUMENT");
        var members = MemberUuids.Select(member => Guid.TryParse(member, out var id) && id != Guid.Empty
            ? id.ToString("D") : throw new IAMAdapterException("IAM_BASELINE_INVALID_ARGUMENT")).ToArray();
        if (members.Distinct(StringComparer.Ordinal).Count() != members.Length
            || PermissionCodes.Any(code => code is null || !AuthorizationBaselineContract.PermissionPattern.IsMatch(code))
            || PermissionCodes.Distinct(StringComparer.Ordinal).Count() != PermissionCodes.Count)
            throw new IAMAdapterException("IAM_BASELINE_INVALID_ARGUMENT");
        return new(tenant.ToString("D"), members, PermissionCodes.ToArray());
    }
}

public interface IAuthorizationBaselineService
{
    Task<AuthorizationBaselineResult> ReadMemberBaselines(AuthorizationBaselineQuery query, CancellationToken ct = default);
}

public sealed class ValidatedAuthorizationBaselineService : IAuthorizationBaselineService
{
    private readonly string _pluginId;
    private readonly IAuthorizationBaselineService _provider;
    public ValidatedAuthorizationBaselineService(string pluginId, IAuthorizationBaselineService provider)
    {
        if (string.IsNullOrWhiteSpace(pluginId) || pluginId.Trim() != pluginId || provider is null)
            throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
        _pluginId = pluginId;
        _provider = provider;
    }
    public async Task<AuthorizationBaselineResult> ReadMemberBaselines(AuthorizationBaselineQuery query, CancellationToken ct = default)
    {
        var normalized = query.Normalize();
        ct.ThrowIfCancellationRequested();
        var result = await _provider.ReadMemberBaselines(normalized, ct);
        ct.ThrowIfCancellationRequested();
        if (result is null) throw new IAMAdapterException("IAM_BASELINE_RESPONSE_INVALID");
        result.Validate(normalized, _pluginId);
        return result;
    }
}
