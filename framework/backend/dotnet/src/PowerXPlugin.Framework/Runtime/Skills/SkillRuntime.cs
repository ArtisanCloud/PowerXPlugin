using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Skills;

public sealed record SkillScope(string TenantUuid, string UserUuid, string AgentId, string SessionId, string TraceId, string PluginId = "", string Capability = "");
/// <summary>Context published to Core. Trusted tenant/user fields are intentionally absent.</summary>
public sealed record SkillInvokeRequest(string SkillId, string Version, JsonElement Payload, IReadOnlyDictionary<string, JsonElement>? Context = null);
public sealed record SkillInvokeResult(string TraceId, string Status, string ProtocolUsed, bool FallbackUsed, JsonElement Result);
public sealed record LocalSkillManifest(string SkillId, string Version, string Capability = "");
public interface ISkillInvoker { Task<SkillInvokeResult> InvokeAsync(SkillScope scope, SkillInvokeRequest request, CancellationToken ct = default); }
/// <summary>Local registry authorizes only registered Skill ID/version/capability combinations.</summary>
public interface ILocalSkillAuthorizationRegistry { Task<LocalSkillManifest?> ResolveAuthorizedAsync(SkillScope scope, string skillId, string version, CancellationToken ct = default); }
public interface ILocalSkillExecutor { Task<SkillInvokeResult> ExecuteAsync(SkillScope scope, SkillInvokeRequest request, LocalSkillManifest manifest, CancellationToken ct = default); }
public sealed class LocalSkillInvoker(ILocalSkillAuthorizationRegistry registry, ILocalSkillExecutor executor) : ISkillInvoker
{
    private readonly ILocalSkillAuthorizationRegistry _registry = registry ?? throw new ArgumentNullException(nameof(registry)); private readonly ILocalSkillExecutor _executor = executor ?? throw new ArgumentNullException(nameof(executor));
    public async Task<SkillInvokeResult> InvokeAsync(SkillScope scope, SkillInvokeRequest request, CancellationToken ct = default)
    {
        Validate(scope, request); var manifest = await _registry.ResolveAuthorizedAsync(scope, request.SkillId, request.Version, ct) ?? throw new SkillException(SkillErrors.NotAuthorized);
        if (!string.Equals(manifest.SkillId, request.SkillId, StringComparison.Ordinal) || !string.Equals(manifest.Version, request.Version, StringComparison.Ordinal) || !string.IsNullOrEmpty(scope.Capability) && !string.Equals(scope.Capability, manifest.Capability, StringComparison.Ordinal)) throw new SkillException(SkillErrors.NotAuthorized);
        var output = await _executor.ExecuteAsync(scope, request, manifest, ct); if (output is null || string.IsNullOrWhiteSpace(output.TraceId) || output.Result.ValueKind == JsonValueKind.Undefined) throw new SkillException(SkillErrors.InvalidResponse); return output;
    }
    internal static void Validate(SkillScope scope, SkillInvokeRequest request)
    {
        if (!Guid.TryParse(scope.TenantUuid, out var tenant) || tenant == Guid.Empty || tenant.ToString() != scope.TenantUuid || !Guid.TryParse(scope.UserUuid, out var user) || user == Guid.Empty || user.ToString() != scope.UserUuid || string.IsNullOrWhiteSpace(scope.AgentId) || string.IsNullOrWhiteSpace(scope.SessionId) || string.IsNullOrWhiteSpace(scope.TraceId) || string.IsNullOrWhiteSpace(request.SkillId) || string.IsNullOrWhiteSpace(request.Version) || request.Payload.ValueKind == JsonValueKind.Undefined) throw new SkillException(SkillErrors.InvalidArgument);
    }
}

/// <summary>Typed delegated client for Core's tenant Skill invocation contract.</summary>
public sealed class PowerXSkillClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : ISkillInvoker
{
    private readonly string _base = string.IsNullOrWhiteSpace(baseUrl) ? throw new SkillException(SkillErrors.Unavailable) : baseUrl.TrimEnd('/'); private readonly string _tenant = tenantUuid; private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials)); private readonly HttpClient _http = http ?? new();
    public async Task<SkillInvokeResult> InvokeAsync(SkillScope scope, SkillInvokeRequest request, CancellationToken ct = default)
    {
        LocalSkillInvoker.Validate(scope, request); if (!string.Equals(scope.TenantUuid, _tenant, StringComparison.Ordinal)) throw new SkillException(SkillErrors.TenantMismatch);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate(); using var message = new HttpRequestMessage(HttpMethod.Post, _base + "/api/v1/tenant/skills/invoke"); message.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value); message.Content = new StringContent(JsonSerializer.Serialize(new { skill_id = request.SkillId, version = request.Version, payload = request.Payload, context = request.Context ?? new Dictionary<string, JsonElement>() }), Encoding.UTF8, "application/json");
        try { using var response = await _http.SendAsync(message, ct); var raw = await response.Content.ReadAsStringAsync(ct); if (!response.IsSuccessStatusCode) throw new SkillException(response.StatusCode switch { HttpStatusCode.Unauthorized => SkillErrors.Unauthorized, HttpStatusCode.Forbidden => SkillErrors.Forbidden, HttpStatusCode.NotFound => SkillErrors.NotFound, HttpStatusCode.BadRequest => SkillErrors.InvalidArgument, _ => SkillErrors.Upstream }); using var json = JsonDocument.Parse(raw); if (!json.RootElement.TryGetProperty("data", out var data)) throw new SkillException(SkillErrors.InvalidResponse); var output = JsonSerializer.Deserialize<HostOutput>(data.GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new SkillException(SkillErrors.InvalidResponse); if (string.IsNullOrWhiteSpace(output.TraceId) || output.Result.ValueKind == JsonValueKind.Undefined) throw new SkillException(SkillErrors.InvalidResponse); return new SkillInvokeResult(output.TraceId, output.Status ?? string.Empty, output.ProtocolUsed ?? string.Empty, output.FallbackUsed, output.Result); } catch (SkillException) { throw; } catch (HttpRequestException e) { throw new SkillException(SkillErrors.Upstream, e); } catch (JsonException e) { throw new SkillException(SkillErrors.InvalidResponse, e); }
    }
    private sealed class HostOutput { [JsonPropertyName("trace_id")] public string TraceId { get; init; } = ""; public string? Status { get; init; } [JsonPropertyName("protocol_used")] public string? ProtocolUsed { get; init; } [JsonPropertyName("fallback_used")] public bool FallbackUsed { get; init; } public JsonElement Result { get; init; } }
}
public static class SkillRuntimeExtensions { public static IServiceCollection AddPowerXSkillRuntime(this IServiceCollection s, ProviderMode m, Func<IServiceProvider, ISkillInvoker> l, Func<IServiceProvider, ISkillInvoker> d) => s.AddPowerXRuntime("skills", m, l, d); }
public static class SkillErrors { public const string InvalidArgument="FRAMEWORK_SKILL_INVALID_ARGUMENT", InvalidResponse="FRAMEWORK_SKILL_INVALID_RESPONSE", NotAuthorized="FRAMEWORK_SKILL_NOT_AUTHORIZED", TenantMismatch="FRAMEWORK_SKILL_TENANT_MISMATCH", Unavailable="FRAMEWORK_SKILL_UNAVAILABLE", Unauthorized="FRAMEWORK_SKILL_UNAUTHORIZED", Forbidden="FRAMEWORK_SKILL_FORBIDDEN", NotFound="FRAMEWORK_SKILL_NOT_FOUND", Upstream="FRAMEWORK_SKILL_UPSTREAM_DEPENDENCY"; }
public sealed class SkillException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; } = code; }
