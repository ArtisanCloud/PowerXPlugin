using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Capability;

public sealed record CapabilityGrantStatus([property: JsonPropertyName("capability_id")] string CapabilityId, [property: JsonPropertyName("status")] string Status, [property: JsonPropertyName("reason_code")] string ReasonCode);
public interface ICapabilityGrantRegistry
{
    Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> capabilityIds, CancellationToken ct = default);
}
public static class CapabilityGrantErrors
{
    public const string Unavailable = "FRAMEWORK_CAPABILITY_ADAPTER_UNAVAILABLE";
    public const string InvalidRequirements = "FRAMEWORK_CAPABILITY_REQUIREMENTS_INVALID";
    public const string InvalidResponse = "FRAMEWORK_CAPABILITY_GRANT_RESPONSE_INVALID";
    public const string RequiredUnavailable = "FRAMEWORK_REQUIRED_CAPABILITY_UNAVAILABLE";
    public const string Unauthorized = "FRAMEWORK_CAPABILITY_UNAUTHORIZED";
    public const string Forbidden = "FRAMEWORK_CAPABILITY_FORBIDDEN";
    public const string Upstream = "FRAMEWORK_CAPABILITY_UPSTREAM_DEPENDENCY";
}
public sealed class CapabilityGrantException(string code, int? statusCode = null, Exception? inner = null) : InvalidOperationException(code, inner)
{ public string Code { get; } = code; public int? StatusCode { get; } = statusCode; }

public sealed class PowerXCapabilityGrantClient(PowerXCapabilityGrantClientOptions options, HttpClient? http = null) : ICapabilityGrantRegistry
{
    private readonly PowerXCapabilityGrantClientOptions _options = options ?? throw new ArgumentNullException(nameof(options));
    private readonly HttpClient _http = http ?? new HttpClient { Timeout = options.Timeout > TimeSpan.Zero ? options.Timeout : TimeSpan.FromSeconds(30) };
    public async Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> ids, CancellationToken ct = default)
    {
        Validate(ids);
        if (string.IsNullOrWhiteSpace(_options.BaseUrl) || string.IsNullOrWhiteSpace(_options.Credential)) throw new CapabilityGrantException(CapabilityGrantErrors.Unavailable);
        using var req = new HttpRequestMessage(HttpMethod.Post, BaseUrl() + "/api/v1/tenant/capabilities:grant-status") { Content = new StringContent(JsonSerializer.Serialize(new { capability_ids = ids }), Encoding.UTF8, "application/json") };
        req.Headers.Authorization = new AuthenticationHeaderValue(_options.AuthScheme.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase) ? "ApiKey" : "Bearer", _options.Credential.Trim());
        try { using var resp = await _http.SendAsync(req, ct); var raw = await resp.Content.ReadAsStringAsync(ct); if (!resp.IsSuccessStatusCode) throw new CapabilityGrantException(Map(resp.StatusCode), (int)resp.StatusCode); using var doc = JsonDocument.Parse(raw); var data = doc.RootElement.GetProperty("data"); return JsonSerializer.Deserialize<List<CapabilityGrantStatus>>(data.GetProperty("items").GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); }
        catch (CapabilityGrantException) { throw; } catch (HttpRequestException ex) { throw new CapabilityGrantException(CapabilityGrantErrors.Upstream, inner: ex); } catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new CapabilityGrantException(CapabilityGrantErrors.Upstream); }
    }
    public static async Task RequireGrantsAsync(ICapabilityGrantRegistry registry, IReadOnlyList<string> ids, CancellationToken ct = default)
    { Validate(ids); if (ids.Count == 0) return; ArgumentNullException.ThrowIfNull(registry); for (var start = 0; start < ids.Count; start += 100) { var expected = ids.Skip(start).Take(100).ToArray(); var actual = await registry.GetGrantStatusAsync(expected, ct); if (actual.Count != expected.Length) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); for (var i = 0; i < expected.Length; i++) { var item = actual[i]; if (item.CapabilityId != expected[i] || string.IsNullOrWhiteSpace(item.ReasonCode) || item.Status is not ("granted" or "not_granted" or "unknown")) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidResponse); if (item.Status != "granted") throw new CapabilityGrantException(CapabilityGrantErrors.RequiredUnavailable); } } }
    private static void Validate(IReadOnlyList<string> ids) { if (ids == null || ids.Any(x => string.IsNullOrWhiteSpace(x) || x != x.Trim()) || ids.Distinct().Count() != ids.Count) throw new CapabilityGrantException(CapabilityGrantErrors.InvalidRequirements); }
    private string BaseUrl() { var v = _options.BaseUrl.Trim().TrimEnd('/'); return v.StartsWith("http", StringComparison.OrdinalIgnoreCase) ? v : "http://" + v; }
    private static string Map(HttpStatusCode code) => code switch { HttpStatusCode.Unauthorized => CapabilityGrantErrors.Unauthorized, HttpStatusCode.Forbidden => CapabilityGrantErrors.Forbidden, _ => CapabilityGrantErrors.Upstream };
}
public sealed class PowerXCapabilityGrantClientOptions { public string BaseUrl { get; init; } = ""; public string Credential { get; init; } = ""; public string AuthScheme { get; init; } = "Bearer"; public TimeSpan Timeout { get; init; } = TimeSpan.FromSeconds(30); }
