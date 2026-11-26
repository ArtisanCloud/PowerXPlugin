using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Notifications;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Notifications;

public sealed class IdempotentNotificationTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Member = "00000000-0000-4000-8000-000000000002";
    private const string Id = "00000000-0000-4000-8000-000000000003";
    private static CreateNotificationRequest Request => new("fixture_title", "fixture_content", MemberUuid: Member, IdempotencyKey: "crm.approval.wait24.fixture");
    private static Dictionary<string, object?> Receipt() => new() {
        ["contract"] = "notification_create.v1", ["idempotency_key"] = Request.IdempotencyKey, ["replayed"] = false,
        ["notification"] = new Dictionary<string, object?> { ["id"] = Id, ["userId"] = Member, ["title"] = Request.Title, ["content"] = Request.Content, ["type"] = "info", ["category"] = "system", ["isRead"] = false, ["isImportant"] = false, ["createdAt"] = "2026-11-11T02:00:00Z", ["updatedAt"] = "2026-11-11T02:00:00Z" }
    };
    private static PowerXNotificationClient Client(Func<HttpRequestMessage, HttpResponseMessage> handler, string prefix = "/api/v1")
        => new("https://core.example/deployment", Tenant, new StaticServiceCredentialProvider(new("fixture_credential")), new HttpClient(new Handler(handler)), prefix);
    private static HttpResponseMessage Ok(object receipt) => new(HttpStatusCode.OK) { Content = new StringContent(JsonSerializer.Serialize(new { data = receipt }), Encoding.UTF8, "application/json") };

    [Theory]
    [InlineData("/api/v1")] [InlineData("/api/v2")] [InlineData("/core/custom")]
    public async Task Uses_exact_member_key_contract_and_configured_prefix_without_implicit_retry(string prefix)
    {
        var count = 0;
        var client = Client(message => {
            count++; Assert.Equal("/deployment" + prefix + "/notifications/idempotent", message.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer fixture_credential", message.Headers.Authorization!.ToString());
            using var body = JsonDocument.Parse(message.Content!.ReadAsStringAsync().GetAwaiter().GetResult());
            Assert.Equal(Request.IdempotencyKey, body.RootElement.GetProperty("idempotency_key").GetString());
            Assert.Equal(Member, body.RootElement.GetProperty("member_uuid").GetString()); Assert.False(body.RootElement.TryGetProperty("tenant_uuid", out _));
            var receipt = Receipt(); receipt["replayed"] = count > 1; return Ok(receipt);
        }, prefix);
        var first = await client.CreateAsync(new(Tenant), Request); var repeated = await client.CreateAsync(new(Tenant), Request);
        Assert.Equal(Id, first.Uuid); Assert.Equal(first.Uuid, repeated.Uuid); Assert.Equal(2, count);
    }

    [Theory]
    [InlineData(404, NotificationErrors.IdempotencyUnsupported)] [InlineData(501, NotificationErrors.IdempotencyUnsupported)]
    [InlineData(409, NotificationErrors.IdempotencyConflict)] [InlineData(403, NotificationErrors.Forbidden)]
    [InlineData(401, NotificationErrors.Unauthorized)] [InlineData(503, NotificationErrors.Unavailable)]
    [InlineData(307, NotificationErrors.Upstream)]
    public async Task Host_failure_never_drops_key_or_falls_back_to_non_idempotent_route(int status, string expected)
    {
        var paths = new List<string>(); var client = Client(message => { paths.Add(message.RequestUri!.AbsolutePath); return new((HttpStatusCode)status) { Content = new StringContent("{}") }; });
        var error = await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(Tenant), Request)); Assert.Equal(expected, error.Code);
        Assert.Equal(new[] { "/deployment/api/v1/notifications/idempotent" }, paths);
    }

    [Theory]
    [InlineData("contract")] [InlineData("key")] [InlineData("member")] [InlineData("id")] [InlineData("title")] [InlineData("replayed")] [InlineData("navigation")]
    public async Task Incomplete_or_mismatched_receipt_is_not_creation_proof(string mismatch)
    {
        var client = Client(_ => {
            var receipt = Receipt(); var notice = (Dictionary<string, object?>)receipt["notification"]!;
            if (mismatch == "contract") receipt["contract"] = "old";
            if (mismatch == "key") receipt["idempotency_key"] = "other";
            if (mismatch == "member") notice["userId"] = Tenant;
            if (mismatch == "id") notice["id"] = "fixture_invalid_uuid";
            if (mismatch == "title") notice["title"] = "fixture_other";
            if (mismatch == "replayed") receipt.Remove("replayed");
            return Ok(receipt);
        });
        var input = mismatch == "navigation" ? Request with { Navigation = new("fixture_action", "/_p/plugin.crm/admin/approval") } : Request;
        Assert.Equal(NotificationErrors.InvalidResponse, (await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(Tenant), input))).Code);
    }

    [Fact]
    public async Task Caller_can_retry_lost_response_with_same_key_and_client_never_retries_itself()
    {
        var count = 0;
        var client = Client(_ => { count++; if (count == 1) throw new HttpRequestException("fixture_response_lost"); var receipt = Receipt(); receipt["replayed"] = true; return Ok(receipt); });
        Assert.Equal(NotificationErrors.Upstream, (await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(Tenant), Request))).Code); Assert.Equal(1, count);
        Assert.Equal(Id, (await client.CreateAsync(new(Tenant), Request)).Uuid); Assert.Equal(2, count);
    }

    [Theory]
    [InlineData("")] [InlineData("has space")] [InlineData("non_ascii_中")]
    public async Task Malformed_key_and_missing_recipient_fail_before_network(string key)
    {
        var client = Client(_ => throw new Xunit.Sdk.XunitException("fixture_unexpected_network"));
        Assert.Equal(NotificationErrors.InvalidArgument, (await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(Tenant), Request with { IdempotencyKey = key }))).Code);
        Assert.Equal(NotificationErrors.InvalidArgument, (await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(Tenant), Request with { MemberUuid = null }))).Code);
    }

    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handler) : HttpMessageHandler
    { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(handler(request)); }
}
