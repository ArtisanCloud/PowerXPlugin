using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Metadata;

public sealed record MetadataScope(string TenantUuid);
public sealed record MetadataPage<T>(IReadOnlyList<T> Items, long Total, int Page, int PageSize);
public sealed record MetadataListRequest(int Page = 1, int PageSize = 100, string Locale = "", string Module = "", string Status = "", string Query = "");
public sealed record MetadataText(string DisplayName, string DisplayDescription);
public sealed record DictionaryNamespace(string Uuid, string Namespace, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string> DescriptionI18n, string Status, long ItemCount, MetadataText Display);
public sealed record DictionaryItem(string Uuid, string NamespaceUuid, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string> DescriptionI18n, string Status, int SortOrder, long ReferenceCount, MetadataText Display);
public sealed record Taxonomy(string Uuid, string Namespace, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string> DescriptionI18n, int MaxDepth, string Status, MetadataText Display);
public sealed record TaxonomyNode(string Uuid, string TaxonomyUuid, string? ParentUuid, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string> DescriptionI18n, string Path, int Depth, int SortOrder, string Status, long ReferenceCount, long Version, MetadataText Display);
public sealed record Tag(string Uuid, string Namespace, string ResourceType, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string> DescriptionI18n, string Color, string Status, long UsageCount, MetadataText Display);
public sealed record TagBinding(string BindingUuid, string TagUuid, string ResourceType, string ResourceUuid, Tag? Tag);
public sealed record ResourceType(string Uuid, string Key, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string> DescriptionI18n, string ValidatorKey, bool BindingEnabled, string ValidatorStatus, string Status, MetadataText Display);

public sealed record CreateDictionaryNamespaceRequest(string Namespace, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null);
public sealed record UpdateDictionaryNamespaceRequest(string NamespaceUuid, IReadOnlyDictionary<string, string>? NameI18n = null, IReadOnlyDictionary<string, string>? DescriptionI18n = null, string? Status = null);
public sealed record CreateDictionaryItemRequest(string NamespaceUuid, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null, int SortOrder = 0, JsonElement? Metadata = null);
public sealed record UpdateDictionaryItemRequest(string ItemUuid, IReadOnlyDictionary<string, string>? LabelI18n = null, IReadOnlyDictionary<string, string>? DescriptionI18n = null, int? SortOrder = null, string? Status = null, JsonElement? Metadata = null);
public sealed record CreateTaxonomyRequest(string Namespace, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null, int MaxDepth = 0);
public sealed record CreateTaxonomyNodeRequest(string TaxonomyUuid, string? ParentUuid, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null, int SortOrder = 0);
public sealed record UpdateTaxonomyNodeRequest(string NodeUuid, long Version, IReadOnlyDictionary<string, string>? LabelI18n = null, IReadOnlyDictionary<string, string>? DescriptionI18n = null, int? SortOrder = null, string? Status = null);
public sealed record CreateTagRequest(string Namespace, string ResourceType, string Code, IReadOnlyDictionary<string, string> LabelI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null, string Color = "");
public sealed record CreateTagBindingRequest(string TagUuid, string ResourceType, string ResourceUuid);
public sealed record CreateResourceTypeRequest(string ResourceType, string Module, IReadOnlyDictionary<string, string> NameI18n, IReadOnlyDictionary<string, string>? DescriptionI18n = null, string ValidatorKey = "", bool BindingEnabled = false);
public sealed record UpdateResourceTypeRequest(string ResourceTypeUuid, IReadOnlyDictionary<string, string>? NameI18n = null, IReadOnlyDictionary<string, string>? DescriptionI18n = null, string? ValidatorKey = null, bool? BindingEnabled = null, string? Status = null);

/// <summary>Common Metadata business contract. Scope is mandatory locally; Core derives it from the delegated service credential.</summary>
public interface IMetadataService
{
    Task<MetadataPage<DictionaryNamespace>> ListDictionaryNamespacesAsync(MetadataScope scope, MetadataListRequest request, CancellationToken ct = default);
    Task<DictionaryNamespace> CreateDictionaryNamespaceAsync(MetadataScope scope, CreateDictionaryNamespaceRequest request, CancellationToken ct = default);
    Task<DictionaryNamespace> UpdateDictionaryNamespaceAsync(MetadataScope scope, UpdateDictionaryNamespaceRequest request, CancellationToken ct = default);
    Task<MetadataPage<DictionaryItem>> ListDictionaryItemsAsync(MetadataScope scope, string namespaceUuid, MetadataListRequest request, CancellationToken ct = default);
    Task<DictionaryItem> CreateDictionaryItemAsync(MetadataScope scope, CreateDictionaryItemRequest request, CancellationToken ct = default);
    Task<DictionaryItem> UpdateDictionaryItemAsync(MetadataScope scope, UpdateDictionaryItemRequest request, CancellationToken ct = default);
    Task<MetadataPage<Taxonomy>> ListTaxonomiesAsync(MetadataScope scope, MetadataListRequest request, CancellationToken ct = default);
    Task<Taxonomy> CreateTaxonomyAsync(MetadataScope scope, CreateTaxonomyRequest request, CancellationToken ct = default);
    Task<MetadataPage<TaxonomyNode>> ListTaxonomyNodesAsync(MetadataScope scope, string taxonomyUuid, MetadataListRequest request, CancellationToken ct = default);
    Task<TaxonomyNode> CreateTaxonomyNodeAsync(MetadataScope scope, CreateTaxonomyNodeRequest request, CancellationToken ct = default);
    Task<TaxonomyNode> UpdateTaxonomyNodeAsync(MetadataScope scope, UpdateTaxonomyNodeRequest request, CancellationToken ct = default);
    Task<MetadataPage<Tag>> ListTagsAsync(MetadataScope scope, MetadataListRequest request, string resourceType = "", string nameSpace = "", CancellationToken ct = default);
    Task<Tag> CreateTagAsync(MetadataScope scope, CreateTagRequest request, CancellationToken ct = default);
    Task<TagBinding> CreateTagBindingAsync(MetadataScope scope, CreateTagBindingRequest request, CancellationToken ct = default);
    Task DeleteTagBindingAsync(MetadataScope scope, string bindingUuid, CancellationToken ct = default);
    Task<MetadataPage<ResourceType>> ListResourceTypesAsync(MetadataScope scope, MetadataListRequest request, CancellationToken ct = default);
    Task<ResourceType> CreateResourceTypeAsync(MetadataScope scope, CreateResourceTypeRequest request, CancellationToken ct = default);
    Task<ResourceType> UpdateResourceTypeAsync(MetadataScope scope, UpdateResourceTypeRequest request, CancellationToken ct = default);
}

/// <summary>Local callers inject a durable tenant-scoped store. Framework does not include an in-memory metadata implementation.</summary>
public interface ILocalMetadataStore : IMetadataService { }
public sealed class LocalMetadataAdapter(ILocalMetadataStore store) : IMetadataService
{
    private readonly ILocalMetadataStore _store = store ?? throw new ArgumentNullException(nameof(store));
    private static void Scope(MetadataScope scope) { if (!Guid.TryParse(scope.TenantUuid, out var tenant) || tenant == Guid.Empty || tenant.ToString() != scope.TenantUuid) throw new MetadataException(MetadataErrors.InvalidArgument); }
    private static void Uuid(string value) { if (!Guid.TryParse(value, out var id) || id == Guid.Empty || id.ToString() != value) throw new MetadataException(MetadataErrors.InvalidArgument); }
    private static void Required(string value) { if (string.IsNullOrWhiteSpace(value)) throw new MetadataException(MetadataErrors.InvalidArgument); }
    public Task<MetadataPage<DictionaryNamespace>> ListDictionaryNamespacesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) { Scope(s); return _store.ListDictionaryNamespacesAsync(s, r, ct); }
    public Task<DictionaryNamespace> CreateDictionaryNamespaceAsync(MetadataScope s, CreateDictionaryNamespaceRequest r, CancellationToken ct = default) { Scope(s); Required(r.Namespace); Required(r.Module); return _store.CreateDictionaryNamespaceAsync(s, r, ct); }
    public Task<DictionaryNamespace> UpdateDictionaryNamespaceAsync(MetadataScope s, UpdateDictionaryNamespaceRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.NamespaceUuid); return _store.UpdateDictionaryNamespaceAsync(s, r, ct); }
    public Task<MetadataPage<DictionaryItem>> ListDictionaryItemsAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) { Scope(s); Uuid(id); return _store.ListDictionaryItemsAsync(s, id, r, ct); }
    public Task<DictionaryItem> CreateDictionaryItemAsync(MetadataScope s, CreateDictionaryItemRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.NamespaceUuid); Required(r.Code); return _store.CreateDictionaryItemAsync(s, r, ct); }
    public Task<DictionaryItem> UpdateDictionaryItemAsync(MetadataScope s, UpdateDictionaryItemRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.ItemUuid); return _store.UpdateDictionaryItemAsync(s, r, ct); }
    public Task<MetadataPage<Taxonomy>> ListTaxonomiesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) { Scope(s); return _store.ListTaxonomiesAsync(s, r, ct); }
    public Task<Taxonomy> CreateTaxonomyAsync(MetadataScope s, CreateTaxonomyRequest r, CancellationToken ct = default) { Scope(s); Required(r.Namespace); Required(r.Module); return _store.CreateTaxonomyAsync(s, r, ct); }
    public Task<MetadataPage<TaxonomyNode>> ListTaxonomyNodesAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) { Scope(s); Uuid(id); return _store.ListTaxonomyNodesAsync(s, id, r, ct); }
    public Task<TaxonomyNode> CreateTaxonomyNodeAsync(MetadataScope s, CreateTaxonomyNodeRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.TaxonomyUuid); Required(r.Code); if (r.ParentUuid is not null) Uuid(r.ParentUuid); return _store.CreateTaxonomyNodeAsync(s, r, ct); }
    public Task<TaxonomyNode> UpdateTaxonomyNodeAsync(MetadataScope s, UpdateTaxonomyNodeRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.NodeUuid); if (r.Version < 1) throw new MetadataException(MetadataErrors.InvalidArgument); return _store.UpdateTaxonomyNodeAsync(s, r, ct); }
    public Task<MetadataPage<Tag>> ListTagsAsync(MetadataScope s, MetadataListRequest r, string t = "", string n = "", CancellationToken ct = default) { Scope(s); return _store.ListTagsAsync(s, r, t, n, ct); }
    public Task<Tag> CreateTagAsync(MetadataScope s, CreateTagRequest r, CancellationToken ct = default) { Scope(s); Required(r.Namespace); Required(r.ResourceType); Required(r.Code); return _store.CreateTagAsync(s, r, ct); }
    public Task<TagBinding> CreateTagBindingAsync(MetadataScope s, CreateTagBindingRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.TagUuid); Uuid(r.ResourceUuid); Required(r.ResourceType); return _store.CreateTagBindingAsync(s, r, ct); }
    public Task DeleteTagBindingAsync(MetadataScope s, string id, CancellationToken ct = default) { Scope(s); Uuid(id); return _store.DeleteTagBindingAsync(s, id, ct); }
    public Task<MetadataPage<ResourceType>> ListResourceTypesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) { Scope(s); return _store.ListResourceTypesAsync(s, r, ct); }
    public Task<ResourceType> CreateResourceTypeAsync(MetadataScope s, CreateResourceTypeRequest r, CancellationToken ct = default) { Scope(s); Required(r.ResourceType); Required(r.Module); return _store.CreateResourceTypeAsync(s, r, ct); }
    public Task<ResourceType> UpdateResourceTypeAsync(MetadataScope s, UpdateResourceTypeRequest r, CancellationToken ct = default) { Scope(s); Uuid(r.ResourceTypeUuid); return _store.UpdateResourceTypeAsync(s, r, ct); }
}

/// <summary>Direct typed client for Core tenant Metadata routes. Tenant is deliberately never serialized.</summary>
public sealed class PowerXMetadataClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : IMetadataService
{
    private readonly string _base = string.IsNullOrWhiteSpace(baseUrl) ? throw new MetadataException(MetadataErrors.Unavailable) : baseUrl.TrimEnd('/'); private readonly string _tenant = tenantUuid; private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials)); private readonly HttpClient _http = http ?? new();
    private void Check(MetadataScope scope) { if (!string.Equals(scope.TenantUuid, _tenant, StringComparison.Ordinal)) throw new MetadataException(MetadataErrors.TenantMismatch); }
    private static string Path(string root, string id) => root + "/" + Uri.EscapeDataString(id);
    private static string Query(MetadataListRequest r, params (string Key, string Value)[] extra) { var values = new List<string> { "page=" + Math.Max(1, r.Page), "page_size=" + Math.Clamp(r.PageSize, 1, 100) }; foreach (var (k, v) in new[] { ("locale", r.Locale), ("module", r.Module), ("status", r.Status), ("q", r.Query) }.Concat(extra)) if (!string.IsNullOrWhiteSpace(v)) values.Add(Uri.EscapeDataString(k) + "=" + Uri.EscapeDataString(v.Trim())); return "?" + string.Join("&", values); }
    public Task<MetadataPage<DictionaryNamespace>> ListDictionaryNamespacesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => Page<DictionaryNamespace>(s, HttpMethod.Get, "/api/v1/tenant/metadata/dictionaries" + Query(r), null, ct);
    public Task<DictionaryNamespace> CreateDictionaryNamespaceAsync(MetadataScope s, CreateDictionaryNamespaceRequest r, CancellationToken ct = default) => Send<DictionaryNamespace>(s, HttpMethod.Post, "/api/v1/tenant/metadata/dictionaries", new { @namespace = r.Namespace, module = r.Module, name_i18n = r.NameI18n, description_i18n = r.DescriptionI18n }, ct);
    public Task<DictionaryNamespace> UpdateDictionaryNamespaceAsync(MetadataScope s, UpdateDictionaryNamespaceRequest r, CancellationToken ct = default) => Send<DictionaryNamespace>(s, HttpMethod.Patch, Path("/api/v1/tenant/metadata/dictionaries", r.NamespaceUuid), new { name_i18n = r.NameI18n, description_i18n = r.DescriptionI18n, status = r.Status }, ct);
    public Task<MetadataPage<DictionaryItem>> ListDictionaryItemsAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) => Page<DictionaryItem>(s, HttpMethod.Get, Path("/api/v1/tenant/metadata/dictionaries", id) + "/items" + Query(r), null, ct);
    public Task<DictionaryItem> CreateDictionaryItemAsync(MetadataScope s, CreateDictionaryItemRequest r, CancellationToken ct = default) => Send<DictionaryItem>(s, HttpMethod.Post, Path("/api/v1/tenant/metadata/dictionaries", r.NamespaceUuid) + "/items", new { code = r.Code, label_i18n = r.LabelI18n, description_i18n = r.DescriptionI18n, sort_order = r.SortOrder, metadata = r.Metadata }, ct);
    public Task<DictionaryItem> UpdateDictionaryItemAsync(MetadataScope s, UpdateDictionaryItemRequest r, CancellationToken ct = default) => Send<DictionaryItem>(s, HttpMethod.Patch, Path("/api/v1/tenant/metadata/dictionary-items", r.ItemUuid), new { label_i18n = r.LabelI18n, description_i18n = r.DescriptionI18n, sort_order = r.SortOrder, status = r.Status, metadata = r.Metadata }, ct);
    public Task<MetadataPage<Taxonomy>> ListTaxonomiesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => Page<Taxonomy>(s, HttpMethod.Get, "/api/v1/tenant/metadata/taxonomies" + Query(r), null, ct);
    public Task<Taxonomy> CreateTaxonomyAsync(MetadataScope s, CreateTaxonomyRequest r, CancellationToken ct = default) => Send<Taxonomy>(s, HttpMethod.Post, "/api/v1/tenant/metadata/taxonomies", new { @namespace = r.Namespace, module = r.Module, name_i18n = r.NameI18n, description_i18n = r.DescriptionI18n, max_depth = r.MaxDepth }, ct);
    public Task<MetadataPage<TaxonomyNode>> ListTaxonomyNodesAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) => Page<TaxonomyNode>(s, HttpMethod.Get, Path("/api/v1/tenant/metadata/taxonomies", id) + "/nodes" + Query(r), null, ct);
    public Task<TaxonomyNode> CreateTaxonomyNodeAsync(MetadataScope s, CreateTaxonomyNodeRequest r, CancellationToken ct = default) => Send<TaxonomyNode>(s, HttpMethod.Post, Path("/api/v1/tenant/metadata/taxonomies", r.TaxonomyUuid) + "/nodes", new { parent_uuid = r.ParentUuid, code = r.Code, label_i18n = r.LabelI18n, description_i18n = r.DescriptionI18n, sort_order = r.SortOrder }, ct);
    public Task<TaxonomyNode> UpdateTaxonomyNodeAsync(MetadataScope s, UpdateTaxonomyNodeRequest r, CancellationToken ct = default) => Send<TaxonomyNode>(s, HttpMethod.Patch, Path("/api/v1/tenant/metadata/taxonomy-nodes", r.NodeUuid), new { label_i18n = r.LabelI18n, description_i18n = r.DescriptionI18n, sort_order = r.SortOrder, status = r.Status, version = r.Version }, ct);
    public Task<MetadataPage<Tag>> ListTagsAsync(MetadataScope s, MetadataListRequest r, string type = "", string ns = "", CancellationToken ct = default) => Page<Tag>(s, HttpMethod.Get, "/api/v1/tenant/metadata/tags" + Query(r, ("resource_type", type), ("namespace", ns)), null, ct);
    public Task<Tag> CreateTagAsync(MetadataScope s, CreateTagRequest r, CancellationToken ct = default) => Send<Tag>(s, HttpMethod.Post, "/api/v1/tenant/metadata/tags", new { @namespace = r.Namespace, resource_type = r.ResourceType, code = r.Code, color = r.Color, label_i18n = r.LabelI18n, description_i18n = r.DescriptionI18n }, ct);
    public Task<TagBinding> CreateTagBindingAsync(MetadataScope s, CreateTagBindingRequest r, CancellationToken ct = default) => Send<TagBinding>(s, HttpMethod.Post, "/api/v1/tenant/metadata/tag-bindings", new { tag_uuid = r.TagUuid, resource_type = r.ResourceType, resource_uuid = r.ResourceUuid }, ct);
    public async Task DeleteTagBindingAsync(MetadataScope s, string id, CancellationToken ct = default) { Check(s); await Send<object>(s, HttpMethod.Delete, Path("/api/v1/tenant/metadata/tag-bindings", id), null, ct, allowEmpty: true); }
    public Task<MetadataPage<ResourceType>> ListResourceTypesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => Page<ResourceType>(s, HttpMethod.Get, "/api/v1/tenant/metadata/resource-types" + Query(r), null, ct);
    public Task<ResourceType> CreateResourceTypeAsync(MetadataScope s, CreateResourceTypeRequest r, CancellationToken ct = default) => Send<ResourceType>(s, HttpMethod.Post, "/api/v1/tenant/metadata/resource-types", new { resource_type = r.ResourceType, module = r.Module, name_i18n = r.NameI18n, description_i18n = r.DescriptionI18n, validator_key = r.ValidatorKey, binding_enabled = r.BindingEnabled }, ct);
    public Task<ResourceType> UpdateResourceTypeAsync(MetadataScope s, UpdateResourceTypeRequest r, CancellationToken ct = default) => Send<ResourceType>(s, HttpMethod.Patch, Path("/api/v1/tenant/metadata/resource-types", r.ResourceTypeUuid), new { name_i18n = r.NameI18n, description_i18n = r.DescriptionI18n, validator_key = r.ValidatorKey, binding_enabled = r.BindingEnabled, status = r.Status }, ct);
    private async Task<MetadataPage<T>> Page<T>(MetadataScope s, HttpMethod m, string path, object? body, CancellationToken ct) { var x = await Send<PagePayload<T>>(s, m, path, body, ct); return new(x.Items ?? [], x.Pagination?.Total ?? 0, x.Pagination?.Page ?? 1, x.Pagination?.PageSize ?? 0); }
    private async Task<T> Send<T>(MetadataScope s, HttpMethod m, string path, object? body, CancellationToken ct, bool allowEmpty = false) { Check(s); var c = (await _credentials.GetCredentialAsync(ct)).Validate(); using var request = new HttpRequestMessage(m, _base + path); request.Headers.Authorization = new AuthenticationHeaderValue(c.NormalizedAuthScheme, c.Value); if (body is not null) request.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json"); try { using var response = await _http.SendAsync(request, ct); var raw = await response.Content.ReadAsStringAsync(ct); if (!response.IsSuccessStatusCode) throw new MetadataException(response.StatusCode switch { HttpStatusCode.Unauthorized => MetadataErrors.Unauthorized, HttpStatusCode.Forbidden => MetadataErrors.Forbidden, HttpStatusCode.NotFound => MetadataErrors.NotFound, HttpStatusCode.Conflict => MetadataErrors.Conflict, HttpStatusCode.BadRequest => MetadataErrors.InvalidArgument, _ => MetadataErrors.Upstream }); if (allowEmpty && string.IsNullOrWhiteSpace(raw)) return default!; using var json = JsonDocument.Parse(raw); if (!json.RootElement.TryGetProperty("data", out var data)) throw new MetadataException(MetadataErrors.InvalidResponse); return JsonSerializer.Deserialize<T>(data.GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new MetadataException(MetadataErrors.InvalidResponse); } catch (MetadataException) { throw; } catch (HttpRequestException e) { throw new MetadataException(MetadataErrors.Upstream, e); } catch (JsonException e) { throw new MetadataException(MetadataErrors.InvalidResponse, e); } }
    private sealed class PagePayload<T> { public List<T>? Items { get; init; } public PaginationPayload? Pagination { get; init; } }
    private sealed class PaginationPayload { public long Total { get; init; } public int Page { get; init; } [JsonPropertyName("page_size")] public int PageSize { get; init; } }
}
public static class MetadataRuntimeExtensions { public static IServiceCollection AddPowerXMetadataRuntime(this IServiceCollection s, ProviderMode m, Func<IServiceProvider, IMetadataService> l, Func<IServiceProvider, IMetadataService> d) => s.AddPowerXRuntime("metadata", m, l, d); }
public static class MetadataErrors { public const string InvalidArgument="FRAMEWORK_METADATA_INVALID_ARGUMENT", InvalidResponse="FRAMEWORK_METADATA_INVALID_RESPONSE", TenantMismatch="FRAMEWORK_METADATA_TENANT_MISMATCH", Unavailable="FRAMEWORK_METADATA_UNAVAILABLE", Unauthorized="FRAMEWORK_METADATA_UNAUTHORIZED", Forbidden="FRAMEWORK_METADATA_FORBIDDEN", NotFound="FRAMEWORK_METADATA_NOT_FOUND", Conflict="FRAMEWORK_METADATA_CONFLICT", Upstream="FRAMEWORK_METADATA_UPSTREAM_DEPENDENCY"; }
public sealed class MetadataException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; } = code; }
