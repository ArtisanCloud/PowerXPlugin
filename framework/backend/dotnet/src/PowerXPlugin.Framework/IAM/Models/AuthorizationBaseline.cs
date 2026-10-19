using System.Globalization;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Text.RegularExpressions;
using PowerXPlugin.Framework.IAM.Contracts;

namespace PowerXPlugin.Framework.IAM.Models;

public static class AuthorizationBaselineContract
{
    public const string CapabilityId = "com.corex.iam.authorization_baseline.service_read";
    public const string Endpoint = "core://iam/authorization-baseline";
    public const string Operation = "read_member_baselines";
    public const string QuerySchema = "powerx.iam.authorization-baseline-query/v1";
    public const string ResultSchema = "powerx.iam.authorization-baseline-result/v1";
    public const string TimeFormat = "yyyy-MM-dd'T'HH:mm:ss.ffffff'Z'";
    internal static readonly Regex PermissionPattern = new(@"\A[a-zA-Z0-9_.-]+:[a-zA-Z0-9_.-]+\z", RegexOptions.CultureInvariant);
    private static readonly Regex ScopePattern = new(@"\Asha256:[a-f0-9]{64}\z", RegexOptions.CultureInvariant);
    internal static IAMAdapterException Invalid() => new("IAM_BASELINE_RESPONSE_INVALID");

    internal static void Exact(JsonElement value, params string[] fields)
    {
        if (value.ValueKind != JsonValueKind.Object) throw Invalid();
        var names = value.EnumerateObject().Select(property => property.Name).ToArray();
        if (names.Length != fields.Length || names.Distinct(StringComparer.Ordinal).Count() != names.Length
            || names.Except(fields, StringComparer.Ordinal).Any()) throw Invalid();
    }

    internal static string Text(JsonElement element, string name) => element.GetProperty(name).GetString() ?? throw Invalid();
    internal static DateTimeOffset? Time(JsonElement element, string name)
    {
        var value = element.GetProperty(name);
        if (value.ValueKind == JsonValueKind.Null) return null;
        var text = value.GetString();
        if (text is null || text.Length != 27 || !DateTimeOffset.TryParseExact(text, TimeFormat, CultureInfo.InvariantCulture,
            DateTimeStyles.AssumeUniversal | DateTimeStyles.AdjustToUniversal, out var time) || time == default
            || time.ToString(TimeFormat, CultureInfo.InvariantCulture) != text) throw Invalid();
        return time;
    }

    internal static AuthorizationBaselineDecision ReadDecision(JsonElement item)
    {
        var status = Text(item, "status");
        var member = Text(item, "member_uuid");
        var permission = Text(item, "permission_code");
        if (status == "error")
        {
            Exact(item, "member_uuid", "permission_code", "status", "error_code");
            return new() { MemberUuid = member, PermissionCode = permission, Status = status, ErrorCode = Text(item, "error_code") };
        }
        if (status != "decided" || item.GetProperty("allowed").ValueKind is not (JsonValueKind.True or JsonValueKind.False)) throw Invalid();
        var allowed = item.GetProperty("allowed").GetBoolean();
        string[] fields = ["member_uuid", "permission_code", "status", "allowed", "granted_at", "visible_since", "authorization_revision", "data_scope", "scope_fingerprint", "valid_until", "recheck_at"];
        Exact(item, allowed ? [..fields, "baseline_kind", "known_since"] : fields);
        return new() {
            MemberUuid = member, PermissionCode = permission, Status = status, Allowed = allowed,
            BaselineKind = allowed ? Text(item, "baseline_kind") : null,
            GrantedAt = Time(item, "granted_at"), KnownSince = allowed ? Time(item, "known_since") : null,
            VisibleSince = Time(item, "visible_since"), AuthorizationRevision = Text(item, "authorization_revision"),
            DataScope = Text(item, "data_scope"), ScopeFingerprint = Text(item, "scope_fingerprint"),
            ValidUntil = Time(item, "valid_until"), RecheckAt = Time(item, "recheck_at")
        };
    }

    public static AuthorizationBaselineResult Decode(JsonElement raw, AuthorizationBaselineQuery query, string pluginId)
    {
        try
        {
            Exact(raw, "schema", "tenant_uuid", "plugin_id", "evaluated_at", "items");
            var result = new AuthorizationBaselineResult {
                Schema = Text(raw, "schema"), TenantUuid = Text(raw, "tenant_uuid"), PluginId = Text(raw, "plugin_id"),
                EvaluatedAt = Time(raw, "evaluated_at") ?? throw Invalid(),
                Items = raw.GetProperty("items").EnumerateArray().Select(ReadDecision).ToArray()
            };
            result.Validate(query, pluginId);
            return result;
        }
        catch (Exception error) when (error is JsonException or KeyNotFoundException or InvalidOperationException or FormatException)
        { throw Invalid(); }
    }

    internal static void Validate(AuthorizationBaselineResult result, AuthorizationBaselineQuery query, string pluginId)
    {
        var normalized = query.Normalize();
        if (string.IsNullOrWhiteSpace(pluginId) || result.Schema != ResultSchema || result.PluginId != pluginId
            || result.TenantUuid != normalized.TenantUuid || result.EvaluatedAt == default || result.EvaluatedAt.Ticks % 10 != 0 || result.EvaluatedAt.Offset != TimeSpan.Zero
            || result.Items is null || result.Items.Count != normalized.MemberUuids.Count * normalized.PermissionCodes.Count) throw Invalid();
        var expected = normalized.MemberUuids.SelectMany(member => normalized.PermissionCodes.Select(code => member + "/" + code)).ToHashSet(StringComparer.Ordinal);
        foreach (var item in result.Items)
        {
            if (item is null || !expected.Remove(item.MemberUuid + "/" + item.PermissionCode)) throw Invalid();
            if (item.Status == "error")
            {
                if (item.ErrorCode is not ("IAM_BASELINE_MEMBER_UNAVAILABLE" or "IAM_BASELINE_UNAVAILABLE" or "IAM_BASELINE_UNSUPPORTED_SCOPE" or "IAM_BASELINE_UNSUPPORTED_POLICY" or "IAM_BASELINE_ROOT_BYPASS_UNSUPPORTED")
                    || item.Allowed.HasValue || item.BaselineKind is not null || item.GrantedAt.HasValue || item.KnownSince.HasValue
                    || item.VisibleSince.HasValue || item.AuthorizationRevision is not null || item.DataScope is not null
                    || item.ScopeFingerprint is not null || item.ValidUntil.HasValue || item.RecheckAt.HasValue) throw Invalid();
                continue;
            }
            if (item.Status != "decided" || !item.Allowed.HasValue || item.ErrorCode is not null
                || string.IsNullOrWhiteSpace(item.AuthorizationRevision) || item.DataScope != "TENANT"
                || item.ScopeFingerprint is null || !ScopePattern.IsMatch(item.ScopeFingerprint)) throw Invalid();
            foreach (var time in new[] { item.GrantedAt, item.KnownSince, item.VisibleSince, item.ValidUntil, item.RecheckAt })
                if (time.HasValue && (time.Value == default || time.Value.Ticks % 10 != 0 || time.Value.Offset != TimeSpan.Zero)) throw Invalid();
            if (item.RecheckAt.HasValue && item.RecheckAt <= result.EvaluatedAt) throw Invalid();
            if (!item.Allowed.Value)
            {
                if (item.BaselineKind is not null || item.GrantedAt.HasValue || item.KnownSince.HasValue || item.VisibleSince.HasValue || item.ValidUntil.HasValue) throw Invalid();
                continue;
            }
            if (!item.KnownSince.HasValue || !item.VisibleSince.HasValue || item.VisibleSince < item.KnownSince || item.VisibleSince > result.EvaluatedAt) throw Invalid();
            if (item.BaselineKind == "initialized") { if (item.GrantedAt.HasValue) throw Invalid(); }
            else if (item.BaselineKind is "native" or "reconstructed") { if (!item.GrantedAt.HasValue || item.GrantedAt > item.VisibleSince) throw Invalid(); }
            else throw Invalid();
            if (item.ValidUntil.HasValue && (item.ValidUntil <= result.EvaluatedAt || (item.RecheckAt.HasValue && item.RecheckAt > item.ValidUntil))) throw Invalid();
        }
    }
}

public sealed record AuthorizationBaselineResult
{
    [JsonPropertyName("schema")] public string Schema { get; init; } = "";
    [JsonPropertyName("tenant_uuid")] public string TenantUuid { get; init; } = "";
    [JsonPropertyName("plugin_id")] public string PluginId { get; init; } = "";
    [JsonPropertyName("evaluated_at"), JsonConverter(typeof(BaselineTimeConverter))] public DateTimeOffset EvaluatedAt { get; init; }
    [JsonPropertyName("items")] public IReadOnlyList<AuthorizationBaselineDecision> Items { get; init; } = [];
    public void Validate(AuthorizationBaselineQuery query, string pluginId) => AuthorizationBaselineContract.Validate(this, query, pluginId);
}

[JsonConverter(typeof(BaselineDecisionConverter))]
public sealed record AuthorizationBaselineDecision
{
    public string MemberUuid { get; init; } = "";
    public string PermissionCode { get; init; } = "";
    public string Status { get; init; } = "";
    public string? ErrorCode { get; init; }
    public bool? Allowed { get; init; }
    public string? BaselineKind { get; init; }
    public DateTimeOffset? GrantedAt { get; init; }
    public DateTimeOffset? KnownSince { get; init; }
    public DateTimeOffset? VisibleSince { get; init; }
    public string? AuthorizationRevision { get; init; }
    public string? DataScope { get; init; }
    public string? ScopeFingerprint { get; init; }
    public DateTimeOffset? ValidUntil { get; init; }
    public DateTimeOffset? RecheckAt { get; init; }
}

public sealed class BaselineTimeConverter : JsonConverter<DateTimeOffset>
{
    public override DateTimeOffset Read(ref Utf8JsonReader reader, Type type, JsonSerializerOptions options)
    {
        using var document = JsonDocument.ParseValue(ref reader);
        using var wrapped = JsonDocument.Parse("{\"time\":" + document.RootElement.GetRawText() + "}");
        return AuthorizationBaselineContract.Time(wrapped.RootElement, "time") ?? throw AuthorizationBaselineContract.Invalid();
    }
    public override void Write(Utf8JsonWriter writer, DateTimeOffset value, JsonSerializerOptions options)
    {
        if (value == default || value.Ticks % 10 != 0) throw AuthorizationBaselineContract.Invalid();
        writer.WriteStringValue(value.UtcDateTime.ToString(AuthorizationBaselineContract.TimeFormat, CultureInfo.InvariantCulture));
    }
}

public sealed class BaselineDecisionConverter : JsonConverter<AuthorizationBaselineDecision>
{
    public override AuthorizationBaselineDecision Read(ref Utf8JsonReader reader, Type type, JsonSerializerOptions options)
    {
        using var document = JsonDocument.ParseValue(ref reader);
        return AuthorizationBaselineContract.ReadDecision(document.RootElement);
    }
    public override void Write(Utf8JsonWriter writer, AuthorizationBaselineDecision value, JsonSerializerOptions options)
    {
        static string? Format(DateTimeOffset? time) {
            if (time.HasValue && (time.Value == default || time.Value.Ticks % 10 != 0)) throw AuthorizationBaselineContract.Invalid();
            return time?.UtcDateTime.ToString(AuthorizationBaselineContract.TimeFormat, CultureInfo.InvariantCulture);
        }
        var fields = new Dictionary<string, object?> { ["member_uuid"] = value.MemberUuid, ["permission_code"] = value.PermissionCode, ["status"] = value.Status };
        if (value.Status == "error") fields["error_code"] = value.ErrorCode;
        else {
            fields["allowed"] = value.Allowed; fields["authorization_revision"] = value.AuthorizationRevision;
            fields["data_scope"] = value.DataScope; fields["scope_fingerprint"] = value.ScopeFingerprint;
            fields["granted_at"] = Format(value.GrantedAt); fields["visible_since"] = Format(value.VisibleSince);
            fields["valid_until"] = Format(value.ValidUntil); fields["recheck_at"] = Format(value.RecheckAt);
            if (value.Allowed == true) { fields["baseline_kind"] = value.BaselineKind; fields["known_since"] = Format(value.KnownSince); }
        }
        JsonSerializer.Serialize(writer, fields, options);
    }
}
