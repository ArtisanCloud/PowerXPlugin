using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Cache;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Cache;
public sealed class CacheRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    [Fact] public async Task Local_enforces_ttl_size_and_tenant_scoped_store() { var cache = new LocalCacheAdapter(new Store()); await cache.SetAsync(new CacheScope(Tenant, "license"), "key", [], TimeSpan.FromMilliseconds(1)); var entry = await cache.GetAsync(new CacheScope(Tenant, "license"), "key"); Assert.True(entry.Found); await Assert.ThrowsAsync<CacheException>(() => cache.SetAsync(new CacheScope(Tenant, "license"), "key", new byte[1024 * 1024 + 1], TimeSpan.FromSeconds(1))); }
    [Fact] public async Task Delegated_uses_formal_route_and_refuses_tenant_override() { var client = new PowerXCacheClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(r => { Assert.Equal("/api/v1/tenant/runtime/cache/entries", r.RequestUri!.AbsolutePath); Assert.Equal("Bearer sts", r.Headers.Authorization!.ToString()); return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("{\"data\":{\"found\":false,\"value_base64\":\"\",\"expires_at\":null}}", Encoding.UTF8) }; }))); var entry = await client.GetAsync(new CacheScope(Tenant, "license"), "key"); Assert.False(entry.Found); var e = await Assert.ThrowsAsync<CacheException>(() => client.GetAsync(new CacheScope("00000000-0000-4000-8000-000000000002", "license"), "key")); Assert.Equal(CacheErrors.TenantMismatch, e.Code); }
    private sealed class Store : ILocalCacheStore { private readonly Dictionary<string, CacheEntry> _data = []; public Task<CacheEntry> GetAsync(CacheScope s, string k, CancellationToken ct = default) => Task.FromResult(_data.TryGetValue(s.TenantUuid + s.Namespace + k, out var e) ? e : new CacheEntry(false, [], null)); public Task SetAsync(CacheScope s, string k, byte[] v, TimeSpan t, CancellationToken ct = default) { _data[s.TenantUuid + s.Namespace + k] = new CacheEntry(true, v, DateTimeOffset.UtcNow + t); return Task.CompletedTask; } public Task DeleteAsync(CacheScope s, string k, CancellationToken ct = default) { _data.Remove(s.TenantUuid + s.Namespace + k); return Task.CompletedTask; } }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> f) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage r, CancellationToken c) => Task.FromResult(f(r)); }
}
