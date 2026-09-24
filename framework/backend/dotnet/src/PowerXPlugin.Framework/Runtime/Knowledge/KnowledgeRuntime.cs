using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Knowledge;

/// <summary>Published operations supported by the PowerX Knowledge host.</summary>
public static class KnowledgeOperations
{
    public const string Retrieve = "retrieve";
    public const string Search = "search";
    public const string Upsert = "upsert";
    public const string Delete = "delete";
    public const string Reindex = "reindex";
    public const string Health = "health";
}

public static class KnowledgeIndexStatuses
{
    public const string Queued = "queued";
    public const string Running = "running";
    public const string Succeeded = "succeeded";
    public const string Failed = "failed";
    public const string Cancelled = "cancelled";
}

public sealed record KnowledgeCapabilities(string Provider, ProviderMode Mode, IReadOnlySet<string> Operations)
{
    public bool Supports(string operation) => Operations.Contains(operation);
}

public sealed record KnowledgeSpace(
    string SpaceId,
    string SpaceName,
    string TenantUuid,
    string Status,
    string Visibility = "tenant",
    string PluginId = "");

public sealed record ListKnowledgeSpacesRequest(
    string TenantUuid,
    int Limit = 10,
    string PluginId = "",
    string Visibility = "",
    string Status = "",
    string TraceId = "");

public sealed record KnowledgeDocument(
    string DocumentId,
    string SpaceId,
    string TenantUuid,
    string Title,
    string Uri,
    string Content,
    string ContentType,
    string Checksum,
    string Version,
    IReadOnlyList<string>? Tags = null);

public sealed record KnowledgeQuery(
    string TenantUuid,
    string Query,
    IReadOnlyList<string>? SpaceIds = null,
    int Limit = 10,
    string TraceId = "",
    IReadOnlyList<string>? Tags = null,
    double MinScore = 0,
    string PluginId = "",
    string Visibility = "");

public sealed record KnowledgeCitation(string DocumentId, string Title, string Uri, string Provider, DateTimeOffset RetrievedAt);
public sealed record KnowledgeChunk(string DocumentId, string SpaceId, string Text, KnowledgeCitation Citation, IReadOnlyList<string>? Tags = null);
public sealed record KnowledgeSearchResult(string Provider, IReadOnlyList<KnowledgeChunk> Chunks, int Total, string TraceId = "");
public sealed record KnowledgeIndexJob(string JobId, string SpaceId, string DocumentId, string Operation, string Status, string ErrorCode = "", string Message = "");
public sealed record DeleteKnowledgeDocumentRequest(string TenantUuid, string SpaceId, string DocumentId, string TraceId = "");
public sealed record ReindexKnowledgeRequest(string TenantUuid, string SpaceId, string DocumentId = "", string TraceId = "");
public sealed record GetKnowledgeIndexJobRequest(string TenantUuid, string JobId, string TraceId = "");

public interface IKnowledgeService
{
    KnowledgeCapabilities Capabilities { get; }
    Task<IReadOnlyList<KnowledgeSpace>> ListSpacesAsync(ListKnowledgeSpacesRequest request, CancellationToken ct = default);
    Task<KnowledgeSearchResult> SearchAsync(KnowledgeQuery query, CancellationToken ct = default);
    Task<KnowledgeIndexJob> UpsertDocumentAsync(KnowledgeDocument document, CancellationToken ct = default);
    Task<KnowledgeIndexJob> DeleteDocumentAsync(DeleteKnowledgeDocumentRequest request, CancellationToken ct = default);
    Task<KnowledgeIndexJob> ReindexAsync(ReindexKnowledgeRequest request, CancellationToken ct = default);
    Task<KnowledgeIndexJob> GetIndexJobAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default);
}

/// <summary>Plugin-owned local implementation. The Framework never supplies a mock store as a fallback.</summary>
public interface ILocalKnowledgeStore : IKnowledgeService { }

public sealed class LocalKnowledgeAdapter(ILocalKnowledgeStore store) : IKnowledgeService
{
    private readonly ILocalKnowledgeStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public KnowledgeCapabilities Capabilities => _store.Capabilities with { Mode = ProviderMode.Local };
    public async Task<IReadOnlyList<KnowledgeSpace>> ListSpacesAsync(ListKnowledgeSpacesRequest request, CancellationToken ct = default)
    {
        ValidateList(request); var spaces = await _store.ListSpacesAsync(request, ct);
        if (spaces.Any(space => space is null || space.TenantUuid != request.TenantUuid || string.IsNullOrWhiteSpace(space.SpaceId) || string.IsNullOrWhiteSpace(space.SpaceName))) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        return spaces;
    }
    public async Task<KnowledgeSearchResult> SearchAsync(KnowledgeQuery query, CancellationToken ct = default)
    {
        ValidateQuery(query); var result = await _store.SearchAsync(query, ct);
        if (result is null || result.Total < 0 || result.Chunks.Any(chunk => chunk is null || string.IsNullOrWhiteSpace(chunk.DocumentId) || string.IsNullOrWhiteSpace(chunk.SpaceId) || string.IsNullOrWhiteSpace(chunk.Text) || chunk.Citation is null)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse);
        return result;
    }
    public async Task<KnowledgeIndexJob> UpsertDocumentAsync(KnowledgeDocument document, CancellationToken ct = default) { ValidateDocument(document); return ValidateJob(await _store.UpsertDocumentAsync(document, ct), document.TenantUuid, document.SpaceId); }
    public async Task<KnowledgeIndexJob> DeleteDocumentAsync(DeleteKnowledgeDocumentRequest request, CancellationToken ct = default) { ValidateDelete(request); return ValidateJob(await _store.DeleteDocumentAsync(request, ct), request.TenantUuid, request.SpaceId); }
    public async Task<KnowledgeIndexJob> ReindexAsync(ReindexKnowledgeRequest request, CancellationToken ct = default) { ValidateReindex(request, allowDocument: true); return ValidateJob(await _store.ReindexAsync(request, ct), request.TenantUuid, request.SpaceId); }
    public async Task<KnowledgeIndexJob> GetIndexJobAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default) { ValidateJobRequest(request); return ValidateJob(await _store.GetIndexJobAsync(request, ct), request.TenantUuid, null); }

    internal static void ValidateTenant(string tenant) { if (!Guid.TryParse(tenant, out var parsed) || parsed == Guid.Empty || parsed.ToString() != tenant) throw new KnowledgeException(KnowledgeErrors.InvalidArgument); }
    internal static void ValidateList(ListKnowledgeSpacesRequest request) { ValidateTenant(request.TenantUuid); if (request.Limit is < 1 or > 100 || !string.IsNullOrEmpty(request.PluginId) || !string.IsNullOrEmpty(request.Visibility) || !string.IsNullOrEmpty(request.Status)) throw new KnowledgeException(KnowledgeErrors.Unsupported); }
    internal static void ValidateQuery(KnowledgeQuery query) { ValidateTenant(query.TenantUuid); if (string.IsNullOrWhiteSpace(query.Query) || query.Limit is < 1 or > 100 || query.Tags?.Count > 0 || query.MinScore != 0 || !string.IsNullOrEmpty(query.PluginId) || !string.IsNullOrEmpty(query.Visibility) || query.SpaceIds?.Any(string.IsNullOrWhiteSpace) == true) throw new KnowledgeException(KnowledgeErrors.InvalidArgument); }
    internal static void ValidateDocument(KnowledgeDocument document) { ValidateTenant(document.TenantUuid); if (string.IsNullOrWhiteSpace(document.SpaceId) || string.IsNullOrWhiteSpace(document.Title) || (string.IsNullOrWhiteSpace(document.Content) && string.IsNullOrWhiteSpace(document.Uri))) throw new KnowledgeException(KnowledgeErrors.InvalidDocument); }
    internal static void ValidateDelegatedDocument(KnowledgeDocument document) { ValidateDocument(document); if (string.IsNullOrWhiteSpace(document.Content) || string.IsNullOrWhiteSpace(document.ContentType) || string.IsNullOrWhiteSpace(document.Checksum) || string.IsNullOrWhiteSpace(document.Version)) throw new KnowledgeException(KnowledgeErrors.InvalidDocument); }
    internal static void ValidateDelete(DeleteKnowledgeDocumentRequest request) { ValidateTenant(request.TenantUuid); if (string.IsNullOrWhiteSpace(request.SpaceId) || string.IsNullOrWhiteSpace(request.DocumentId)) throw new KnowledgeException(KnowledgeErrors.InvalidDocument); }
    internal static void ValidateReindex(ReindexKnowledgeRequest request, bool allowDocument) { ValidateTenant(request.TenantUuid); if (string.IsNullOrWhiteSpace(request.SpaceId)) throw new KnowledgeException(KnowledgeErrors.InvalidDocument); if (!allowDocument && !string.IsNullOrWhiteSpace(request.DocumentId)) throw new KnowledgeException(KnowledgeErrors.Unsupported); }
    internal static void ValidateJobRequest(GetKnowledgeIndexJobRequest request) { ValidateTenant(request.TenantUuid); if (string.IsNullOrWhiteSpace(request.JobId)) throw new KnowledgeException(KnowledgeErrors.InvalidDocument); }
    internal static KnowledgeIndexJob ValidateJob(KnowledgeIndexJob job, string tenant, string? spaceId) { if (job is null || string.IsNullOrWhiteSpace(job.JobId) || !string.IsNullOrWhiteSpace(spaceId) && job.SpaceId != spaceId || !KnowledgeIndexStatusesValues.Contains(job.Status)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse); return job; }
    private static readonly HashSet<string> KnowledgeIndexStatusesValues = [KnowledgeIndexStatuses.Queued, KnowledgeIndexStatuses.Running, KnowledgeIndexStatuses.Succeeded, KnowledgeIndexStatuses.Failed, KnowledgeIndexStatuses.Cancelled];
}

/// <summary>Typed delegated client for Core's published /api/v1/tenant/knowledge contract.</summary>
public sealed class PowerXKnowledgeClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : IKnowledgeService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);
    private readonly string _baseUrl = string.IsNullOrWhiteSpace(baseUrl) ? throw new KnowledgeException(KnowledgeErrors.Unavailable) : baseUrl.TrimEnd('/');
    private readonly string _tenantUuid = tenantUuid;
    private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials));
    private readonly HttpClient _http = http ?? new HttpClient();
    public KnowledgeCapabilities Capabilities { get; } = new("powerx_knowledge_host", ProviderMode.Delegated, new HashSet<string> { KnowledgeOperations.Retrieve, KnowledgeOperations.Search, KnowledgeOperations.Upsert, KnowledgeOperations.Delete, KnowledgeOperations.Reindex, KnowledgeOperations.Health });
    public async Task<IReadOnlyList<KnowledgeSpace>> ListSpacesAsync(ListKnowledgeSpacesRequest request, CancellationToken ct = default)
    {
        CheckTenant(request.TenantUuid); LocalKnowledgeAdapter.ValidateList(request);
        var response = await SendAsync<SpaceListResponse>(HttpMethod.Get, "/api/v1/tenant/knowledge/spaces", null, ct);
        return response.Items.Select(item => new KnowledgeSpace(Require(item.SpaceUuid), Require(item.Name), request.TenantUuid, item.Status?.Trim() ?? string.Empty)).ToArray();
    }
    public async Task<KnowledgeSearchResult> SearchAsync(KnowledgeQuery query, CancellationToken ct = default)
    {
        CheckTenant(query.TenantUuid); LocalKnowledgeAdapter.ValidateQuery(query);
        var response = await SendAsync<SearchResponse>(HttpMethod.Post, "/api/v1/tenant/knowledge/search", new { query = query.Query.Trim(), space_uuids = query.SpaceIds ?? [], limit = query.Limit }, ct);
        var chunks = response.Items.Select(item =>
        {
            var documentId = Require(item.DocumentUuid); var spaceId = Require(item.SpaceUuid); var title = Require(item.Title); var excerpt = Require(item.Excerpt);
            return new KnowledgeChunk(documentId, spaceId, excerpt, new KnowledgeCitation(documentId, title, item.Uri?.Trim() ?? string.Empty, "powerx_knowledge_host", DateTimeOffset.UtcNow), item.Tags ?? []);
        }).ToArray();
        return new KnowledgeSearchResult("powerx_knowledge_host", chunks, chunks.Length, query.TraceId);
    }
    public async Task<KnowledgeIndexJob> UpsertDocumentAsync(KnowledgeDocument document, CancellationToken ct = default)
    {
        CheckTenant(document.TenantUuid); LocalKnowledgeAdapter.ValidateDelegatedDocument(document);
        var job = await SendAsync<HostIndexJob>(HttpMethod.Post, "/api/v1/tenant/knowledge/spaces/" + Uri.EscapeDataString(document.SpaceId) + "/documents", new { title = document.Title.Trim(), uri = document.Uri?.Trim() ?? string.Empty, content = document.Content.Trim(), content_type = document.ContentType.Trim(), checksum = document.Checksum.Trim(), version = document.Version.Trim(), tags = document.Tags ?? [] }, ct);
        return ToJob(job);
    }
    public async Task<KnowledgeIndexJob> DeleteDocumentAsync(DeleteKnowledgeDocumentRequest request, CancellationToken ct = default)
    {
        CheckTenant(request.TenantUuid); LocalKnowledgeAdapter.ValidateDelete(request);
        return ToJob(await SendAsync<HostIndexJob>(HttpMethod.Delete, "/api/v1/tenant/knowledge/spaces/" + Uri.EscapeDataString(request.SpaceId) + "/documents/" + Uri.EscapeDataString(request.DocumentId), null, ct));
    }
    public async Task<KnowledgeIndexJob> ReindexAsync(ReindexKnowledgeRequest request, CancellationToken ct = default)
    {
        CheckTenant(request.TenantUuid); LocalKnowledgeAdapter.ValidateReindex(request, allowDocument: false);
        var job = ToJob(await SendAsync<HostIndexJob>(HttpMethod.Post, "/api/v1/tenant/knowledge/spaces/" + Uri.EscapeDataString(request.SpaceId) + "/indexes:rebuild", null, ct));
        return job with { Operation = KnowledgeOperations.Reindex };
    }
    public async Task<KnowledgeIndexJob> GetIndexJobAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default)
    {
        CheckTenant(request.TenantUuid); LocalKnowledgeAdapter.ValidateJobRequest(request);
        return ToJob(await SendAsync<HostIndexJob>(HttpMethod.Get, "/api/v1/tenant/knowledge/index-jobs/" + Uri.EscapeDataString(request.JobId), null, ct));
    }
    private void CheckTenant(string tenantUuid) { LocalKnowledgeAdapter.ValidateTenant(tenantUuid); if (!string.Equals(tenantUuid, _tenantUuid, StringComparison.Ordinal)) throw new KnowledgeException(KnowledgeErrors.TenantMismatch); }
    private async Task<T> SendAsync<T>(HttpMethod method, string path, object? body, CancellationToken ct)
    {
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate(); using var request = new HttpRequestMessage(method, _baseUrl + path);
        request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value); if (body is not null) request.Content = new StringContent(JsonSerializer.Serialize(body, JsonOptions), Encoding.UTF8, "application/json");
        try { using var response = await _http.SendAsync(request, ct); var raw = await response.Content.ReadAsStringAsync(ct); if (!response.IsSuccessStatusCode) throw new KnowledgeException(response.StatusCode switch { HttpStatusCode.BadRequest => KnowledgeErrors.InvalidDocument, HttpStatusCode.Unauthorized => KnowledgeErrors.Unauthorized, HttpStatusCode.Forbidden => KnowledgeErrors.Forbidden, HttpStatusCode.NotFound => KnowledgeErrors.NotFound, HttpStatusCode.Conflict => KnowledgeErrors.Conflict, _ => KnowledgeErrors.Upstream }); using var json = JsonDocument.Parse(raw); if (!json.RootElement.TryGetProperty("data", out var data)) throw new KnowledgeException(KnowledgeErrors.InvalidResponse); return JsonSerializer.Deserialize<T>(data.GetRawText(), JsonOptions) ?? throw new KnowledgeException(KnowledgeErrors.InvalidResponse); } catch (KnowledgeException) { throw; } catch (HttpRequestException exception) { throw new KnowledgeException(KnowledgeErrors.Upstream, exception); } catch (JsonException exception) { throw new KnowledgeException(KnowledgeErrors.InvalidResponse, exception); }
    }
    private static string Require(string? value) => string.IsNullOrWhiteSpace(value) ? throw new KnowledgeException(KnowledgeErrors.InvalidResponse) : value.Trim();
    private static KnowledgeIndexJob ToJob(HostIndexJob job) { var result = new KnowledgeIndexJob(Require(job.JobUuid), Require(job.SpaceUuid), job.DocumentUuid?.Trim() ?? string.Empty, job.Operation?.Trim() ?? string.Empty, Require(job.Status), job.ErrorCode?.Trim() ?? string.Empty); return LocalKnowledgeAdapter.ValidateJob(result, string.Empty, null); }
    private sealed class SpaceListResponse { public List<HostSpace> Items { get; init; } = []; }
    private sealed class HostSpace { [JsonPropertyName("space_uuid")] public string? SpaceUuid { get; init; } public string? Name { get; init; } public string? Status { get; init; } }
    private sealed class SearchResponse { public List<HostSearchItem> Items { get; init; } = []; }
    private sealed class HostSearchItem { [JsonPropertyName("space_uuid")] public string? SpaceUuid { get; init; } [JsonPropertyName("document_uuid")] public string? DocumentUuid { get; init; } public string? Title { get; init; } public string? Uri { get; init; } public string? Excerpt { get; init; } public IReadOnlyList<string>? Tags { get; init; } }
    private sealed class HostIndexJob { [JsonPropertyName("job_uuid")] public string? JobUuid { get; init; } [JsonPropertyName("space_uuid")] public string? SpaceUuid { get; init; } [JsonPropertyName("document_uuid")] public string? DocumentUuid { get; init; } public string? Operation { get; init; } public string? Status { get; init; } [JsonPropertyName("error_code")] public string? ErrorCode { get; init; } }
}

public static class KnowledgeRuntimeExtensions
{
    public static IServiceCollection AddPowerXKnowledgeRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, IKnowledgeService> localFactory, Func<IServiceProvider, IKnowledgeService> delegatedFactory) => services.AddPowerXRuntime("knowledge", mode, localFactory, delegatedFactory);
}

public static class KnowledgeErrors
{
    public const string InvalidArgument = "FRAMEWORK_KNOWLEDGE_INVALID_ARGUMENT";
    public const string InvalidDocument = "FRAMEWORK_KNOWLEDGE_INVALID_DOCUMENT";
    public const string InvalidResponse = "FRAMEWORK_KNOWLEDGE_INVALID_RESPONSE";
    public const string TenantMismatch = "FRAMEWORK_KNOWLEDGE_TENANT_MISMATCH";
    public const string Unsupported = "FRAMEWORK_KNOWLEDGE_UNSUPPORTED";
    public const string Unavailable = "FRAMEWORK_KNOWLEDGE_UNAVAILABLE";
    public const string Unauthorized = "FRAMEWORK_KNOWLEDGE_UNAUTHORIZED";
    public const string Forbidden = "FRAMEWORK_KNOWLEDGE_FORBIDDEN";
    public const string NotFound = "FRAMEWORK_KNOWLEDGE_NOT_FOUND";
    public const string Conflict = "FRAMEWORK_KNOWLEDGE_CONFLICT";
    public const string Upstream = "FRAMEWORK_KNOWLEDGE_UPSTREAM_DEPENDENCY";
}
public sealed class KnowledgeException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; } = code; }
