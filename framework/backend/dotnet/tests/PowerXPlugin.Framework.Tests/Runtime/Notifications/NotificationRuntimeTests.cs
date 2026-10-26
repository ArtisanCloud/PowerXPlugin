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
    [Fact] public async Task Delegated_preserves_optional_navigation_and_response_actions()
    {
        const string path = "/_p/plugin.test/admin/order?target=opaque";
        var client = new PowerXNotificationClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(r => {
            using var request = System.Text.Json.JsonDocument.Parse(r.Content!.ReadAsStringAsync().GetAwaiter().GetResult());
            Assert.Equal(path, request.RootElement.GetProperty("navigation").GetProperty("path").GetString());
            Assert.Equal("fixture_action", request.RootElement.GetProperty("navigation").GetProperty("label").GetString());
            var response = new { data = new { id = "fixture", actions = new[] { new { id = "view_business", label = "fixture_action", type = "primary", action = "navigate", @params = new { path } } } } };
            return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(System.Text.Json.JsonSerializer.Serialize(response), Encoding.UTF8) };
        })));
        var notice = await client.CreateAsync(new NotificationScope(Tenant), new CreateNotificationRequest("fixture_title", "fixture_content", Navigation: new NotificationNavigation("fixture_action", path)));
        Assert.Equal(path, Assert.Single(notice.Actions!).Params!["path"]);
    }
    [Theory]
    [InlineData(false)]
    [InlineData(true)]
    public async Task Delegated_read_receipt_is_member_and_exact_range_bound(bool widen)
    {
        const string member = "00000000-0000-4000-8000-000000000003";
        const string id = "00000000-0000-4000-8000-000000000004";
        var client = new PowerXNotificationClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(r => {
            Assert.Equal("/api/v1/notifications/read-receipts", r.RequestUri!.AbsolutePath);
            using var body = System.Text.Json.JsonDocument.Parse(r.Content!.ReadAsStringAsync().GetAwaiter().GetResult());
            Assert.Equal(member, body.RootElement.GetProperty("member_uuid").GetString());
            var result = new { data = new { contract = "notification_read_receipt.v1", member_uuid = member, notification_uuids = widen ? new[] { id, "00000000-0000-4000-8000-000000000005" } : new[] { id } } };
            return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(System.Text.Json.JsonSerializer.Serialize(result)) };
        })));
        var action = () => client.AcknowledgeAsync(new NotificationScope(Tenant), new NotificationReadRequest(member, new[] { id }));
        if (widen) { var error = await Assert.ThrowsAsync<NotificationException>(action); Assert.Equal(NotificationErrors.InvalidResponse, error.Code); }
        else { Assert.Equal(id, Assert.Single((await action()).NotificationUuids)); }
    }
    [Theory]
    [InlineData("", "/api/v1")]
    [InlineData("/api/v1", "/api/v1")]
    [InlineData("/api/v2", "/api/v2")]
    [InlineData("api/v2/", "/api/v2")]
    [InlineData("/core/custom", "/core/custom")]
    [InlineData("/", "")]
    public async Task Commands_share_configured_core_api_prefix(string prefix, string expected)
    {
        const string member = "11111111-1111-4111-8111-111111111111";
        const string id = "22222222-2222-4222-8222-222222222222";
        var paths = new List<string>();
        var client = new PowerXNotificationClient("https://core.example/deployment/", Tenant,
            new StaticServiceCredentialProvider(new ServiceCredential("sts")),
            new HttpClient(new Handler(r => {
                paths.Add(r.RequestUri!.AbsolutePath);
                Assert.Equal("Bearer sts", r.Headers.Authorization!.ToString());
                object data = r.RequestUri.AbsolutePath.EndsWith("/read-receipts", StringComparison.Ordinal)
                    ? new { contract = "notification_read_receipt.v1", member_uuid = member, notification_uuids = new[] { id } }
                    : new { id };
                return new(HttpStatusCode.OK) { Content = new StringContent(System.Text.Json.JsonSerializer.Serialize(new { data })) };
            })), apiPrefix: prefix);
        await client.CreateAsync(new(Tenant), new("fixture_title", "fixture_content"));
        await client.AcknowledgeAsync(new(Tenant), new(member, new[] { id }));
        Assert.Equal(new[] { "/deployment" + expected + "/notifications", "/deployment" + expected + "/notifications/read-receipts" }, paths);
    }

    [Fact]
    public async Task Invalid_read_arguments_fail_before_host_delivery()
    {
        var client = new PowerXNotificationClient("https://core.example", Tenant,
            new StaticServiceCredentialProvider(new ServiceCredential("sts")),
            new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("unexpected host delivery"))));
        var error = await Assert.ThrowsAsync<NotificationException>(() => client.AcknowledgeAsync(new(Tenant), new(Tenant, Array.Empty<string>())));
        Assert.Equal(NotificationErrors.InvalidArgument, error.Code);
    }

    [Theory]
    [InlineData("/api/../v1")]
    [InlineData("/api//v1")]
    [InlineData("/api/v1?token=invalid")]
    [InlineData("/api/%2fv1")]
    public void Invalid_api_prefix_rejected_at_construction(string prefix)
    {
        Assert.Throws<ArgumentException>(() => new PowerXNotificationClient("https://core.example", Tenant,
            new StaticServiceCredentialProvider(new ServiceCredential("sts")), apiPrefix: prefix));
    }

    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> f) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage r, CancellationToken ct) => Task.FromResult(f(r)); }
}
