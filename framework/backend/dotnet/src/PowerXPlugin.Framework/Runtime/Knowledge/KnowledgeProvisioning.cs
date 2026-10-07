using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Knowledge;

// Service DTOs deliberately contain no tenant, actor, endpoint or raw proxy fields.
public sealed record KnowledgeSpaceQuotas(
    [property: JsonPropertyName("cpu_cores")] int CpuCores,
    [property: JsonPropertyName("storage_gb")] int StorageGb,
    [property: JsonPropertyName("ingestion_concurrency")] int IngestionConcurrency);
public sealed record KnowledgeProfileRef(string Uuid, string Key, int Version);
public sealed record KnowledgeProfileMapping(KnowledgeProfileRef? Ingestion, KnowledgeProfileRef? Index, KnowledgeProfileRef? Rag);
public sealed record KnowledgePolicyTemplateRef(string Uuid, string Name, string Version);
public sealed record KnowledgeStrategyDependencies(IReadOnlyList<string>? Index, IReadOnlyList<string>? Runtime, IReadOnlyList<string>? Assets);
public sealed record KnowledgeScene(string Key, string Label, string Description,
    [property: JsonPropertyName("default_bundle")] string DefaultBundle,
    [property: JsonPropertyName("allowed_bundles")] IReadOnlyList<string> AllowedBundles);
public sealed record KnowledgeStrategyPackage(string Key, string Label, string Summary,
    [property: JsonPropertyName("recommended_profile_key")] string RecommendedProfileKey,
    [property: JsonPropertyName("recommended_scenes")] IReadOnlyList<string> RecommendedScenes,
    KnowledgeStrategyDependencies Dependencies, KnowledgeProfileMapping Profiles, bool Available,
    [property: JsonPropertyName("unavailable_reasons")] IReadOnlyList<string> UnavailableReasons,
    [property: JsonPropertyName("activation_dependencies")] IReadOnlyList<string> ActivationDependencies);
public sealed record KnowledgeCatalog(string Version, string Source, IReadOnlyList<KnowledgeScene> Scenes,
    [property: JsonPropertyName("strategy_packages")] IReadOnlyList<KnowledgeStrategyPackage> StrategyPackages,
    [property: JsonPropertyName("policy_templates")] IReadOnlyList<KnowledgePolicyTemplateRef> PolicyTemplates,
    [property: JsonPropertyName("default_policy_template_uuid")] string? DefaultPolicyTemplateUuid,
    [property: JsonPropertyName("quota_defaults")] KnowledgeSpaceQuotas QuotaDefaults,
    [property: JsonPropertyName("quota_minimums")] KnowledgeSpaceQuotas QuotaMinimums,
    [property: JsonPropertyName("quota_override_allowed")] bool QuotaOverrideAllowed);
public sealed record CreateKnowledgeSpaceRequest(string Name,
    [property: JsonPropertyName("department_uuid")] string DepartmentUuid,
    [property: JsonPropertyName("strategy_key")] string StrategyKey,
    [property: JsonPropertyName("scene_key"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? SceneKey = null,
    [property: JsonPropertyName("policy_template_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? PolicyTemplateUuid = null,
    [property: JsonPropertyName("ingestion_profile_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? IngestionProfileUuid = null,
    [property: JsonPropertyName("index_profile_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? IndexProfileUuid = null,
    [property: JsonPropertyName("rag_profile_uuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? RagProfileUuid = null,
    [property: JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] KnowledgeSpaceQuotas? Quotas = null);
public sealed record CreatedKnowledgeSpace(
    [property: JsonPropertyName("space_uuid")] string SpaceUuid, string Name, string Status,
    [property: JsonPropertyName("department_uuid")] string DepartmentUuid,
    [property: JsonPropertyName("strategy_key")] string StrategyKey,
    [property: JsonPropertyName("scene_key")] string SceneKey,
    [property: JsonPropertyName("policy_template_uuid")] string PolicyTemplateUuid,
    KnowledgeProfileMapping Profiles, KnowledgeSpaceQuotas Quotas);

// Explicit extension: Local implementations do not gain Core provisioning implicitly.
public interface IKnowledgeProvisioningService
{
    Task<KnowledgeCatalog> GetCatalogAsync(CancellationToken ct = default);
    Task<CreatedKnowledgeSpace> CreateSpaceAsync(CreateKnowledgeSpaceRequest request, CancellationToken ct = default);
}

public sealed partial class PowerXKnowledgeClient
{
    public async Task<KnowledgeCatalog> GetCatalogAsync(CancellationToken ct = default)
    {
        LocalKnowledgeAdapter.ValidateTenant(_tenantUuid);
        var response = await SendAsync<CatalogResponse>(HttpMethod.Get, "/api/v1/tenant/knowledge/catalog", null, ct);
        var catalog = response.Catalog;
        if (catalog is null || catalog.Source != "powerx_core" || string.IsNullOrWhiteSpace(catalog.Version) || !ValidQuotas(catalog.QuotaDefaults) || !ValidQuotas(catalog.QuotaMinimums) || catalog.Scenes is null || catalog.PolicyTemplates is null || catalog.StrategyPackages is null)
            throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        foreach (var template in catalog.PolicyTemplates)
            if (!ValidUuid(template.Uuid) || string.IsNullOrWhiteSpace(template.Name) || string.IsNullOrWhiteSpace(template.Version)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        if (!string.IsNullOrEmpty(catalog.DefaultPolicyTemplateUuid) && !catalog.PolicyTemplates.Any(item => item.Uuid == catalog.DefaultPolicyTemplateUuid)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        foreach (var strategy in catalog.StrategyPackages)
            if (string.IsNullOrWhiteSpace(strategy.Key) || string.IsNullOrWhiteSpace(strategy.Label) || !ValidProfiles(strategy.Profiles, strategy.Available)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        return catalog;
    }
    public async Task<CreatedKnowledgeSpace> CreateSpaceAsync(CreateKnowledgeSpaceRequest request, CancellationToken ct = default)
    {
        LocalKnowledgeAdapter.ValidateTenant(_tenantUuid);
        if (string.IsNullOrWhiteSpace(request.Name) || Encoding.UTF8.GetByteCount(request.Name) > 128 || string.IsNullOrWhiteSpace(request.StrategyKey) || !ValidUuid(request.DepartmentUuid) || request.Quotas is not null && !ValidQuotas(request.Quotas)) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
        foreach (var id in new[] { request.PolicyTemplateUuid, request.IngestionProfileUuid, request.IndexProfileUuid, request.RagProfileUuid })
            if (id is not null && !ValidUuid(id)) throw new KnowledgeException(KnowledgeErrors.InvalidArgument);
        var response = await SendAsync<CreatedResponse>(HttpMethod.Post, "/api/v1/tenant/knowledge/spaces", request, ct);
        var item = response.Item;
        if (item is null || !ValidUuid(item.SpaceUuid) || !ValidUuid(item.DepartmentUuid) || !ValidUuid(item.PolicyTemplateUuid) || string.IsNullOrWhiteSpace(item.Name) || item.Status != "pending_iam" || item.DepartmentUuid != request.DepartmentUuid || item.StrategyKey != request.StrategyKey || string.IsNullOrWhiteSpace(item.SceneKey) || !ValidProfiles(item.Profiles, true) || !ValidQuotas(item.Quotas)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        return item;
    }
    private static bool ValidUuid(string? value) => Guid.TryParseExact(value, "D", out var id) && id != Guid.Empty && id.ToString() == value;
    private static bool ValidQuotas(KnowledgeSpaceQuotas? quotas) => quotas is { CpuCores: >= 1, StorageGb: >= 50, IngestionConcurrency: >= 1 };
    private static bool ValidProfiles(KnowledgeProfileMapping? mapping, bool required)
    {
        if (mapping is null) return false;
        return new[] { mapping.Ingestion, mapping.Index, mapping.Rag }.All(profile => profile is null ? !required : ValidUuid(profile.Uuid) && !string.IsNullOrWhiteSpace(profile.Key) && profile.Version >= 1);
    }
    private static KnowledgeException HostError(HttpResponseMessage response, string raw)
    {
        string? code = null;
        var trace = response.Headers.TryGetValues("X-Trace-Id", out var values) ? values.FirstOrDefault() ?? string.Empty : string.Empty;
        try
        {
            using var json = JsonDocument.Parse(raw);
            var root = json.RootElement;
            if (root.TryGetProperty("reason_code", out var reason) && reason.ValueKind == JsonValueKind.String) code = reason.GetString();
            if (string.IsNullOrWhiteSpace(code) && root.TryGetProperty("error_code", out var error) && error.ValueKind == JsonValueKind.String) code = error.GetString();
            if (trace.Length == 0 && root.TryGetProperty("request_id", out var id) && id.ValueKind == JsonValueKind.String) trace = id.GetString() ?? string.Empty;
        }
        catch (JsonException) { /* Preserve HTTP status even when the upstream error body is not JSON. */ }
        code = string.IsNullOrWhiteSpace(code) ? response.StatusCode switch
        {
            HttpStatusCode.BadRequest => KnowledgeErrors.InvalidDocument,
            HttpStatusCode.Unauthorized => KnowledgeErrors.Unauthorized,
            HttpStatusCode.Forbidden => KnowledgeErrors.Forbidden,
            HttpStatusCode.NotFound => KnowledgeErrors.NotFound,
            HttpStatusCode.Conflict => KnowledgeErrors.Conflict,
            _ => KnowledgeErrors.Upstream
        } : code;
        return new KnowledgeException(code, statusCode: (int)response.StatusCode, traceId: trace);
    }
    private sealed record CatalogResponse(KnowledgeCatalog? Catalog);
    private sealed record CreatedResponse(CreatedKnowledgeSpace? Item);
}
