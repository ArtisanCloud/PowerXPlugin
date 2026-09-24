using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Capability;

/// <summary>Typed Core tenant Capability Registry client, using service credentials only.</summary>
public sealed class PowerXCapabilityRegistryClient : ICapabilityRegistry
{
    private static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web);
    private readonly HttpClient _http; private readonly string _baseUrl; private readonly IServiceCredentialProvider _credentials;
    public PowerXCapabilityRegistryClient(string baseUrl, IServiceCredentialProvider credentials, HttpClient? http = null)
    { _baseUrl = string.IsNullOrWhiteSpace(baseUrl) ? throw new CapabilityGrantException(CapabilityGrantErrors.Unavailable) : baseUrl.TrimEnd('/'); _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials)); _http = http ?? new HttpClient(); }
    public async Task<IReadOnlyList<CapabilityDescriptor>> ListAsync(CapabilityListQuery q, CancellationToken ct = default) => (await SendAsync<ListEnvelope>(HttpMethod.Get, "/api/v1/tenant/capabilities" + Query(new Dictionary<string, string?> { ["page"] = q.Page.ToString(), ["page_size"] = q.PageSize.ToString(), ["plugin_id"] = q.PluginId, ["intent"] = q.Intent, ["channel"] = q.ToolScope, ["protocol"] = q.Protocol, ["source"] = q.Source }), null, ct)).Items;
    public async Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> ids, CancellationToken ct = default)
    { if (ids is null || ids.Count == 0 || ids.Any(string.IsNullOrWhiteSpace) || ids.Distinct().Count() != ids.Count) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidRequirements); return (await SendAsync<GrantEnvelope>(HttpMethod.Post, "/api/v1/tenant/capabilities:grant-status", new { capability_ids = ids }, ct)).Items; }
    public async Task<CapabilityResolution> ResolveAsync(CapabilityResolveQuery q, CancellationToken ct = default)
    { if (string.IsNullOrWhiteSpace(q.Method) || string.IsNullOrWhiteSpace(q.Endpoint)) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidRequirements); return (await SendAsync<ResolveEnvelope>(HttpMethod.Get, "/api/v1/tenant/capabilities/resolve" + Query(new Dictionary<string, string?> { ["method"] = q.Method.ToUpperInvariant(), ["endpoint"] = q.Endpoint, ["source"] = q.Source }), null, ct)).PrimaryMatch ?? throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); }
    public Task<CapabilityInvokeResult> InvokeAsync(CapabilityInvokeRequest request, CancellationToken ct = default) { if (string.IsNullOrWhiteSpace(request.CapabilityId) && string.IsNullOrWhiteSpace(request.Intent)) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidRequirements); return SendAsync<CapabilityInvokeResult>(HttpMethod.Post, "/api/v1/tenant/invocations", request, ct); }
    public Task<CapabilityInvocation> GetInvocationAsync(string traceId, CancellationToken ct = default) { if (string.IsNullOrWhiteSpace(traceId)) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidRequirements); return SendAsync<CapabilityInvocation>(HttpMethod.Get, "/api/v1/tenant/invocations/" + Uri.EscapeDataString(traceId), null, ct); }
    private async Task<T> SendAsync<T>(HttpMethod method, string path, object? body, CancellationToken ct)
    { var credential = (await _credentials.GetCredentialAsync(ct)).Validate(); using var req = new HttpRequestMessage(method, _baseUrl + path); req.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value); if (body is not null) req.Content = new StringContent(JsonSerializer.Serialize(body, Json), Encoding.UTF8, "application/json"); try { using var response = await _http.SendAsync(req, ct); var raw = await response.Content.ReadAsStringAsync(ct); if (!response.IsSuccessStatusCode) throw new CapabilityGrantException(Map(response.StatusCode), (int)response.StatusCode); using var doc = JsonDocument.Parse(raw); if (!doc.RootElement.TryGetProperty("data", out var data)) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); return JsonSerializer.Deserialize<T>(data.GetRawText(), Json) ?? throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); } catch (CapabilityGrantException) { throw; } catch (HttpRequestException ex) { throw new CapabilityGrantException(CapabilityGrantErrors.Upstream, inner: ex); } }
    private static string Query(IReadOnlyDictionary<string, string?> values) { var parts = values.Where(x => !string.IsNullOrWhiteSpace(x.Value)).Select(x => Uri.EscapeDataString(x.Key) + "=" + Uri.EscapeDataString(x.Value!)); var q = string.Join("&", parts); return q.Length == 0 ? "" : "?" + q; }
    private static string Map(HttpStatusCode status) => status == HttpStatusCode.Unauthorized ? CapabilityGrantErrors.Unauthorized : status == HttpStatusCode.Forbidden ? CapabilityGrantErrors.Forbidden : CapabilityGrantErrors.Upstream;
    private sealed record ListEnvelope(IReadOnlyList<CapabilityDescriptor> Items);
    private sealed record GrantEnvelope(IReadOnlyList<CapabilityGrantStatus> Items);
    private sealed record ResolveEnvelope(CapabilityResolution? PrimaryMatch);
}
