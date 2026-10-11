using System.Globalization;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Text.RegularExpressions;

namespace PowerXPlugin.Framework.IAM.Models;

public sealed record DelegatedIdentityQuery
{
    public required string ContextTicket { get; init; }
    public override string ToString() => "DelegatedIdentityQuery{redacted}";
    public void Validate()
    {
        if (string.IsNullOrEmpty(ContextTicket)) throw new IAMAdapterException("IAM_CONTEXT_PROOF_REQUIRED");
        if (!Regex.IsMatch(ContextTicket, "^pxctx_v1_[A-Za-z0-9_-]{43}$")) throw new IAMAdapterException("IAM_CONTEXT_PROOF_INVALID");
    }
}

public sealed record DelegatedIdentity
{
    [JsonIgnore] public string? WSRevalidationHandle {get;init;}
    [JsonIgnore] public string? WSRevalidationExpiresAt {get;init;}
    [JsonIgnore] public DateTimeOffset WSRevalidationExpiry => ParseExpiry(WSRevalidationExpiresAt ?? "");
    public override string ToString() => "DelegatedIdentity{redacted}";
    internal static DelegatedIdentity FromCore(JsonElement element,JsonSerializerOptions options)
    {
        try {
        var identity=element.Deserialize<DelegatedIdentity>(options) ?? throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        string? Read(string name) => element.TryGetProperty(name,out var value) ? value.ValueKind==JsonValueKind.String ? value.GetString() : throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID") : null;
        return identity with { WSRevalidationHandle=Read("ws_revalidation_handle"),WSRevalidationExpiresAt=Read("ws_revalidation_expires_at") };
        } catch(JsonException) { throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID"); }
    }
    [JsonRequired, JsonPropertyName("schema")] public string Schema { get; init; } = "";
    [JsonRequired, JsonPropertyName("mode")] public string Mode { get; init; } = "";
    [JsonRequired, JsonPropertyName("plugin_id")] public string PluginId { get; init; } = "";
    [JsonRequired, JsonPropertyName("tenant_uuid")] public string TenantUuid { get; init; } = "";
    [JsonRequired, JsonPropertyName("user_uuid")] public string UserUuid { get; init; } = "";
    [JsonRequired, JsonPropertyName("member_uuid")] public string? MemberUuid { get; init; }
    [JsonRequired, JsonPropertyName("is_root")] public bool IsRoot { get; init; }
    [JsonRequired, JsonPropertyName("roles")] public string[] Roles { get; init; } = [];
    [JsonRequired, JsonPropertyName("permissions")] public string[] Permissions { get; init; } = [];
    [JsonRequired, JsonPropertyName("policy_version")] public string PolicyVersion { get; init; } = "";
    [JsonRequired, JsonPropertyName("perms_hash")] public string PermsHash { get; init; } = "";
    [JsonRequired, JsonPropertyName("source_method")] public string SourceMethod { get; init; } = "";
    [JsonRequired, JsonPropertyName("source_path")] public string SourcePath { get; init; } = "";
    [JsonRequired, JsonPropertyName("source_required_permission")] public string SourceRequiredPermission { get; init; } = "";
    [JsonRequired, JsonPropertyName("transport")] public string Transport { get; init; } = "";
    [JsonRequired, JsonPropertyName("source_request_id")] public string SourceRequestId { get; init; } = "";
    [JsonRequired, JsonPropertyName("source_trace_id")] public string SourceTraceId { get; init; } = "";
    [JsonRequired, JsonPropertyName("trace_id")] public string TraceId { get; init; } = "";
    // Preserve RFC3339Nano exactly; DateTimeOffset only has 100ns precision.
    [JsonRequired, JsonPropertyName("proof_expires_at")] public string ProofExpiresAt { get; init; } = "";
    [JsonRequired, JsonPropertyName("user_session_expires_at")] public string UserSessionExpiresAt { get; init; } = "";
    [JsonIgnore] public DateTimeOffset ProofExpiry => ParseExpiry(ProofExpiresAt);
    [JsonIgnore] public DateTimeOffset UserSessionExpiry => ParseExpiry(UserSessionExpiresAt);

    public void Validate(string pluginId, DateTimeOffset now)
        => ValidateSchema(pluginId,now,"powerx.iam.plugin-request-identity/v1");
    internal void ValidateSchema(string pluginId,DateTimeOffset now,string schema)
    {
        var invalid = false;
        invalid |= Schema != schema || Mode != "delegated" || string.IsNullOrEmpty(pluginId) || PluginId != pluginId;
        invalid |= !Uuid(TenantUuid) || !Uuid(UserUuid) || (MemberUuid == null ? !IsRoot : !Uuid(MemberUuid));
        invalid |= Roles == null || Permissions == null || string.IsNullOrEmpty(PolicyVersion) || string.IsNullOrEmpty(PermsHash)
            || string.IsNullOrEmpty(SourceMethod) || string.IsNullOrEmpty(SourcePath) || !SourcePath.StartsWith('/')
            || string.IsNullOrEmpty(SourceRequestId) || string.IsNullOrEmpty(SourceTraceId) || string.IsNullOrEmpty(TraceId)
            || SourceRequiredPermission == null || Transport is not ("http" or "ws_grant" or "ws_upgrade");
        invalid |= Roles?.Any(string.IsNullOrWhiteSpace) == true || Permissions?.Any(string.IsNullOrWhiteSpace) == true;
        if (invalid || ProofExpiry > UserSessionExpiry) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        if (now >= UserSessionExpiry) throw new IAMAdapterException("IAM_CONTEXT_USER_SESSION_EXPIRED");
        if(Transport=="http") {
            if(schema=="powerx.iam.plugin-ws-identity/v1" || WSRevalidationHandle!=null || WSRevalidationExpiresAt!=null) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        } else {
            if(WSRevalidationHandle==null || !Regex.IsMatch(WSRevalidationHandle,"^pxwsi_v1_[A-Za-z0-9_-]{43}$") || WSRevalidationExpiry!=UserSessionExpiry)
                throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
            if(now>=WSRevalidationExpiry) throw new IAMAdapterException("IAM_CONTEXT_WS_HANDLE_EXPIRED");
            if(schema=="powerx.iam.plugin-ws-identity/v1" && ProofExpiry!=WSRevalidationExpiry) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        }
        if (now >= ProofExpiry) throw new IAMAdapterException("IAM_CONTEXT_PROOF_EXPIRED");
    }
    public bool HasPermission(string required)
    {
        if (string.IsNullOrWhiteSpace(required)) return false;
        if (IsRoot) return true;
        var at = required.LastIndexOf(':');
        return at > 0 && Permissions.Any(p => p == required || p is "*" or "*:*" || p == required[..at] + ":*" || p == "*" + required[at..]);
    }
    private static bool Uuid(string value) => Guid.TryParseExact(value, "D", out var parsed) && parsed != Guid.Empty && parsed.ToString("D") == value;
    private static DateTimeOffset ParseExpiry(string value)
    {
        if (value == null || !Regex.IsMatch(value, @"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?Z$"))
            throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        // Floor sub-tick precision when calculating bounds, never round a
        // credential lifetime past the Core session deadline.
        var bounded = Regex.Replace(value, @"\.(\d{7})\d{1,2}Z$", ".$1Z");
        if (!DateTimeOffset.TryParse(bounded, CultureInfo.InvariantCulture, DateTimeStyles.RoundtripKind, out var parsed))
            throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        return parsed;
    }
}
