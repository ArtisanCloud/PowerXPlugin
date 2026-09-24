using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Cache;

public sealed record CacheScope(string TenantUuid, string Namespace);
public sealed record CacheEntry(bool Found, byte[] Value, DateTimeOffset? ExpiresAt);
public interface ICacheService { Task<CacheEntry> GetAsync(CacheScope scope, string key, CancellationToken ct = default); Task SetAsync(CacheScope scope, string key, byte[] value, TimeSpan ttl, CancellationToken ct = default); Task DeleteAsync(CacheScope scope, string key, CancellationToken ct = default); }
public interface ILocalCacheStore : ICacheService { }
public sealed class LocalCacheAdapter(ILocalCacheStore store) : ICacheService
{
    private readonly ILocalCacheStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public async Task<CacheEntry> GetAsync(CacheScope scope, string key, CancellationToken ct = default) { Validate(scope, key); var entry = await _store.GetAsync(scope, key, ct); if (!entry.Found && (entry.Value.Length != 0 || entry.ExpiresAt is not null)) throw new CacheException(CacheErrors.InvalidResponse); return entry; }
    public Task SetAsync(CacheScope scope, string key, byte[] value, TimeSpan ttl, CancellationToken ct = default) { Validate(scope, key); ValidateValue(value, ttl); return _store.SetAsync(scope, key, value, ttl, ct); }
    public Task DeleteAsync(CacheScope scope, string key, CancellationToken ct = default) { Validate(scope, key); return _store.DeleteAsync(scope, key, ct); }
    internal static void Validate(CacheScope s, string key) { if (!Guid.TryParse(s.TenantUuid, out var id) || id == Guid.Empty || id.ToString() != s.TenantUuid || string.IsNullOrWhiteSpace(s.Namespace) || s.Namespace != s.Namespace.Trim() || s.Namespace.Length > 128 || string.IsNullOrWhiteSpace(key) || key != key.Trim() || key.Length > 512) throw new CacheException(CacheErrors.InvalidArgument); }
    internal static void ValidateValue(byte[] value, TimeSpan ttl) { if (value is null || value.Length > 1024 * 1024 || ttl < TimeSpan.FromMilliseconds(1) || ttl > TimeSpan.FromHours(24) || ttl.Ticks % TimeSpan.TicksPerMillisecond != 0) throw new CacheException(CacheErrors.InvalidArgument); }
}
public sealed class PowerXCacheClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : ICacheService
{
    private readonly string _baseUrl = string.IsNullOrWhiteSpace(baseUrl) ? throw new CacheException(CacheErrors.Unavailable) : baseUrl.TrimEnd('/'); private readonly string _tenant = tenantUuid; private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials)); private readonly HttpClient _http = http ?? new HttpClient();
    public async Task<CacheEntry> GetAsync(CacheScope s, string key, CancellationToken ct = default) { ValidateTenant(s, key); var data = await SendAsync(HttpMethod.Get, "/api/v1/tenant/runtime/cache/entries?namespace=" + Uri.EscapeDataString(s.Namespace) + "&key=" + Uri.EscapeDataString(key), null, ct); var found = data.GetProperty("found").GetBoolean(); var value = Convert.FromBase64String(data.GetProperty("value_base64").GetString() ?? ""); DateTimeOffset? expires = data.TryGetProperty("expires_at", out var e) && e.ValueKind == JsonValueKind.String ? DateTimeOffset.Parse(e.GetString()!) : null; if (!found && (value.Length != 0 || expires is not null)) throw new CacheException(CacheErrors.InvalidResponse); return new CacheEntry(found, value, expires); }
    public async Task SetAsync(CacheScope s, string key, byte[] value, TimeSpan ttl, CancellationToken ct = default) { ValidateTenant(s, key); LocalCacheAdapter.ValidateValue(value, ttl); await SendAsync(HttpMethod.Put, "/api/v1/tenant/runtime/cache/entries", new { @namespace = s.Namespace, key, value_base64 = Convert.ToBase64String(value), ttl_ms = (long)ttl.TotalMilliseconds }, ct); }
    public async Task DeleteAsync(CacheScope s, string key, CancellationToken ct = default) { ValidateTenant(s, key); await SendAsync(HttpMethod.Delete, "/api/v1/tenant/runtime/cache/entries?namespace=" + Uri.EscapeDataString(s.Namespace) + "&key=" + Uri.EscapeDataString(key), null, ct); }
    private void ValidateTenant(CacheScope s, string key) { LocalCacheAdapter.Validate(s, key); if (!string.Equals(s.TenantUuid, _tenant, StringComparison.Ordinal)) throw new CacheException(CacheErrors.TenantMismatch); }
    private async Task<JsonElement> SendAsync(HttpMethod method, string path, object? body, CancellationToken ct) { var c = (await _credentials.GetCredentialAsync(ct)).Validate(); using var r = new HttpRequestMessage(method, _baseUrl + path); r.Headers.Authorization = new AuthenticationHeaderValue(c.NormalizedAuthScheme, c.Value); if (body is not null) r.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json"); try { using var resp = await _http.SendAsync(r, ct); var raw = await resp.Content.ReadAsStringAsync(ct); if (!resp.IsSuccessStatusCode) throw new CacheException(resp.StatusCode == HttpStatusCode.Unauthorized ? CacheErrors.Unauthorized : resp.StatusCode == HttpStatusCode.Forbidden ? CacheErrors.Forbidden : CacheErrors.Upstream); using var doc = JsonDocument.Parse(raw); return doc.RootElement.GetProperty("data").Clone(); } catch (CacheException) { throw; } catch (HttpRequestException ex) { throw new CacheException(CacheErrors.Upstream, ex); } }
}
public static class CacheRuntimeExtensions { public static IServiceCollection AddPowerXCacheRuntime(this IServiceCollection s, ProviderMode m, Func<IServiceProvider, ICacheService> l, Func<IServiceProvider, ICacheService> d) => s.AddPowerXRuntime("cache", m, l, d); }
public static class CacheErrors { public const string InvalidArgument = "FRAMEWORK_CACHE_INVALID_ARGUMENT"; public const string InvalidResponse = "FRAMEWORK_CACHE_INVALID_RESPONSE"; public const string TenantMismatch = "FRAMEWORK_CACHE_TENANT_MISMATCH"; public const string Unavailable = "FRAMEWORK_CACHE_UNAVAILABLE"; public const string Unauthorized = "FRAMEWORK_CACHE_UNAUTHORIZED"; public const string Forbidden = "FRAMEWORK_CACHE_FORBIDDEN"; public const string Upstream = "FRAMEWORK_CACHE_UPSTREAM_DEPENDENCY"; }
public sealed class CacheException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; } = code; }
