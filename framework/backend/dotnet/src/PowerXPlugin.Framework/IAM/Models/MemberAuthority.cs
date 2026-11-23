using System.Text.Json.Serialization;
using System.Diagnostics.CodeAnalysis;

namespace PowerXPlugin.Framework.IAM.Models;

public sealed record MemberAuthorityQuery(IReadOnlyList<string> MemberUuids)
{
    public void Validate()
    {
        if (MemberUuids is null || MemberUuids.Count is < 1 or > 200 || MemberUuids.Any(id => !MemberAuthorityBatch.Canonical(id)) || MemberUuids.Distinct(StringComparer.Ordinal).Count() != MemberUuids.Count)
            throw new IAMAdapterException("IAM_ORGANIZATION_INVALID");
    }
}
public sealed record MemberAuthority
{
    [JsonRequired, JsonPropertyName("member_uuid")] public string MemberUuid { get; init; } = "";
    [JsonPropertyName("user_uuid")] public string? UserUuid { get; init; }
    [JsonRequired, JsonPropertyName("status")] public string Status { get; init; } = "";
    [JsonRequired, JsonPropertyName("role_codes")] public List<string> RoleCodes { get; init; } = [];
    [JsonRequired, JsonPropertyName("department_uuids")] public List<string> DepartmentUuids { get; init; } = [];
    [JsonRequired, JsonPropertyName("direct_manager_member_uuid")] public string? DirectManagerMemberUuid { get; init; }
    [JsonRequired, JsonPropertyName("manager_status")] public string ManagerStatus { get; init; } = "";
    [JsonPropertyName("updated_at")] public DateTimeOffset? UpdatedAt { get; init; }
}
public sealed record AuthorityDepartment
{
    [JsonRequired, JsonPropertyName("department_uuid")] public string DepartmentUuid { get; init; } = "";
    [JsonRequired, JsonPropertyName("name")] public string Name { get; init; } = "";
    [JsonRequired, JsonPropertyName("parent_department_uuid")] public string? ParentDepartmentUuid { get; init; }
    [JsonRequired, JsonPropertyName("leader_member_uuid")] public string? LeaderMemberUuid { get; init; }
    [JsonRequired, JsonPropertyName("leader_status")] public string LeaderStatus { get; init; } = "";
}
public sealed record MemberAuthorityBatch
{
    public const string CapabilityId = "com.corex.iam.organization.service_read";
    [JsonRequired, JsonPropertyName("schema")] public string Schema { get; init; } = "powerx.iam.member-authority.v1";
    [JsonRequired, JsonPropertyName("tenant_uuid")] public string TenantUuid { get; init; } = "";
    [JsonRequired, JsonPropertyName("evaluated_at")] public DateTimeOffset EvaluatedAt { get; init; }
    [JsonRequired, JsonPropertyName("members")] public List<MemberAuthority> Members { get; init; } = [];
    [JsonRequired, JsonPropertyName("departments")] public List<AuthorityDepartment> Departments { get; init; } = [];
    public static bool Canonical(string? value) => Guid.TryParse(value, out var id) && id != Guid.Empty && id.ToString() == value;

    public void Validate(MemberAuthorityQuery query, string tenant)
    {
        [DoesNotReturn] void Invalid() => throw new IAMAdapterException("IAM_ORGANIZATION_RESPONSE_INVALID");
        query.Validate();
        if (Schema != "powerx.iam.member-authority.v1" || TenantUuid != tenant || !Canonical(tenant) || EvaluatedAt == default || Members is null || Departments is null || Members.Any(m => m is null) || Departments.Any(d => d is null) || Members.Count != query.MemberUuids.Count || Departments.Count > 2000
            || Members.Select(m => m.MemberUuid).Distinct().Count() != Members.Count || !Members.Select(m => m.MemberUuid).ToHashSet().SetEquals(query.MemberUuids)
            || Departments.Any(d => !Canonical(d.DepartmentUuid)) || Departments.Select(d => d.DepartmentUuid).Distinct().Count() != Departments.Count) Invalid();
        var departments = Departments!.ToDictionary(d => d.DepartmentUuid);
        foreach (var member in Members!)
        {
            if (member.Status is not ("active" or "not_found") || member.RoleCodes is null || member.DepartmentUuids is null || member.RoleCodes.Count > 4000 || member.RoleCodes.Any(string.IsNullOrWhiteSpace)
                || member.RoleCodes.Distinct().Count() != member.RoleCodes.Count || member.DepartmentUuids.Distinct().Count() != member.DepartmentUuids.Count || member.DepartmentUuids.Any(id => !departments.ContainsKey(id))) Invalid();
            if (member.Status == "active" && (!Canonical(member.UserUuid) || member.UpdatedAt is null || member.UpdatedAt == default(DateTimeOffset))) Invalid();
            if (member.Status == "not_found" && (member.UserUuid is not null || member.RoleCodes!.Count != 0 || member.DepartmentUuids!.Count != 0 || member.UpdatedAt is not null || member.ManagerStatus != "unavailable")) Invalid();
            if (member.ManagerStatus is not ("active" or "unset" or "unavailable") || (member.ManagerStatus == "active" ? !Canonical(member.DirectManagerMemberUuid) || member.DirectManagerMemberUuid == member.MemberUuid : member.DirectManagerMemberUuid is not null)) Invalid();
        }
        foreach (var department in Departments)
        {
            if (department.LeaderStatus is not ("active" or "unset" or "unavailable") || (department.LeaderStatus == "active" ? !Canonical(department.LeaderMemberUuid) : department.LeaderMemberUuid is not null)) Invalid();
            var seen = new HashSet<string>(); var current = department;
            for (var depth = 0; ; depth++)
            {
                if (depth >= 64 || !seen.Add(current.DepartmentUuid)) Invalid();
                if (current.ParentDepartmentUuid is null) break;
                if (!departments.TryGetValue(current.ParentDepartmentUuid, out var parent)) Invalid();
                current = parent!;
            }
        }
    }
}
