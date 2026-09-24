using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Notifications;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Notifications;
public sealed class NotificationRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    [Fact] public async Task Delegated_uses_published_route_service_credential_and_never_serializes_tenant()
    {
        string? body = null; HttpRequestMessage? seen = null;
        var client = new PowerXNotificationClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(r => { seen = r; body = r.Content!.ReadAsStringAsync().GetAwaiter().GetResult(); return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("{\"data\":{\"id\":\"notification-1\",\"title\":\"Title\",\"content\":\"Body\",\"isRead\":false,\"isImportant\":false,\"createdAt\":\"2026-01-01T00:00:00Z\",\"updatedAt\":\"2026-01-01T00:00:00Z\"}}", Encoding.UTF8) }; })));
        await client.CreateAsync(new NotificationScope(Tenant), new CreateNotificationRequest("Title", "Body"));
        Assert.Equal("/api/v1/notifications", seen!.RequestUri!.AbsolutePath); Assert.Equal("Bearer sts", seen.Headers.Authorization!.ToString()); Assert.DoesNotContain("tenant", body, StringComparison.OrdinalIgnoreCase);
    }
    [Fact] public async Task Delegated_rejects_unsupported_idempotency_and_cross_tenant_before_delivery()
    {
        var client = new PowerXNotificationClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("must not deliver"))));
        var key = await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new NotificationScope(Tenant), new CreateNotificationRequest("Title", "Body", IdempotencyKey: "one"))); Assert.Equal(NotificationErrors.IdempotencyUnsupported, key.Code);
        var tenant = await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new NotificationScope("00000000-0000-4000-8000-000000000002"), new CreateNotificationRequest("Title", "Body"))); Assert.Equal(NotificationErrors.TenantMismatch, tenant.Code);
    }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> f) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage r, CancellationToken ct) => Task.FromResult(f(r)); }
}
