using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Notifications;
using Xunit;
namespace PowerXPlugin.Framework.Tests.Runtime.Notifications;

public sealed class NotificationCreationReceiptTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001", Member = "00000000-0000-4000-8000-000000000002";
    private static Dictionary<string, object?> Proof(bool found = true) => new() {
        ["contract"] = "notification_create_receipt.v1", ["tenant_uuid"] = Tenant, ["service_owner"] = "plugin:third.party.crm", ["idempotency_key"] = "fixture.key", ["member_uuid"] = Member,
        ["found"] = found, ["notification_state"] = found ? "deleted" : "not_found", ["checked_at"] = "2026-11-12T01:00:00Z",
        ["notification"] = found ? new { id = "00000000-0000-4000-8000-000000000003", title = "fixture_title", content = "fixture_content", type = "warning", category = "crm.approval", isRead = false, isImportant = true, createdAt = "2026-11-11T01:00:00Z", updatedAt = "2026-11-11T01:00:00Z", userId = Member } : null };
    private sealed class Handler(HttpStatusCode status, Dictionary<string, object?> proof) : HttpMessageHandler
    {
        public int Calls; public string? Path, Body;
        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct)
        { Calls++; Path = request.RequestUri!.AbsolutePath; Body = await request.Content!.ReadAsStringAsync(ct); return new(status) { Content = new StringContent(JsonSerializer.Serialize(new { data = proof }), Encoding.UTF8, "application/json") }; }
    }
    private static PowerXNotificationClient Client(Handler handler) => new("http://core.fixture", Tenant, new StaticServiceCredentialProvider(new("fixture_token")), new HttpClient(handler), "/api/v2");
    [Theory][InlineData(true)][InlineData(false)]
    public async Task Exact_scoped_proof_uses_only_the_fixed_readonly_endpoint_and_distinguishes_not_found(bool found)
    {
        var handler = new Handler(HttpStatusCode.OK, Proof(found)); var result = await Client(handler).LookupAsync(new(Tenant), new("fixture.key", Member));
        Assert.Equal(found, result.Found); Assert.Equal(found ? "deleted" : "not_found", result.NotificationState); Assert.Equal(1, handler.Calls); Assert.Equal("/api/v2/notifications/creation-receipts/lookup", handler.Path);
        using var body = JsonDocument.Parse(handler.Body!); Assert.Equal(2, body.RootElement.EnumerateObject().Count()); Assert.Equal("fixture.key", body.RootElement.GetProperty("idempotency_key").GetString()); Assert.Equal(Member, body.RootElement.GetProperty("member_uuid").GetString());
    }
    [Theory][InlineData(404, NotificationCreationReceiptErrors.Unsupported)][InlineData(405, NotificationCreationReceiptErrors.Unsupported)][InlineData(401, NotificationErrors.Unauthorized)][InlineData(403, NotificationErrors.Forbidden)][InlineData(503, NotificationErrors.Upstream)]
    public async Task Unsupported_or_failed_lookup_never_falls_back_to_creation_or_acknowledgement(int status, string code)
    {
        var handler = new Handler((HttpStatusCode)status, Proof()); var error = await Assert.ThrowsAsync<NotificationException>(() => Client(handler).LookupAsync(new(Tenant), new("fixture.key", Member))); Assert.Equal(code, error.Code); Assert.Equal(1, handler.Calls);
    }
    [Theory][InlineData("tenant_uuid", "00000000-0000-4000-8000-000000000099")][InlineData("idempotency_key", "other.key")][InlineData("member_uuid", "00000000-0000-4000-8000-000000000099")][InlineData("service_owner", "local")][InlineData("contract", "unknown")][InlineData("notification_state", "not_found")][InlineData("checked_at", "bad")]
    public async Task Foreign_context_or_unverifiable_proof_is_an_error_not_a_not_found_result(string field, string value)
    {
        var proof = Proof(); proof[field] = value; var handler = new Handler(HttpStatusCode.OK, proof); var error = await Assert.ThrowsAsync<NotificationException>(() => Client(handler).LookupAsync(new(Tenant), new("fixture.key", Member))); Assert.Equal(NotificationErrors.InvalidResponse, error.Code);
    }
    [Fact]
    public async Task Missing_found_flag_or_notification_is_invalid_and_bad_inputs_make_no_http_calls()
    {
        var proof = Proof(); proof.Remove("found"); var handler = new Handler(HttpStatusCode.OK, proof); Assert.Equal(NotificationErrors.InvalidResponse, (await Assert.ThrowsAsync<NotificationException>(() => Client(handler).LookupAsync(new(Tenant), new("fixture.key", Member)))).Code);
        handler = new(HttpStatusCode.OK, Proof()); await Assert.ThrowsAsync<NotificationException>(() => Client(handler).LookupAsync(new(Tenant), new("key with spaces", Member))); await Assert.ThrowsAsync<NotificationException>(() => Client(handler).LookupAsync(new("00000000-0000-4000-8000-000000000099"), new("fixture.key", Member))); Assert.Equal(0, handler.Calls);
    }
}
