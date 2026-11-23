using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Notifications;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Notifications;

public sealed class CoreIdempotentContractTests
{
    [IsolatedNotificationContractFact]
    public async Task Actual_core_readonly_creation_lookup_verifies_original_receipt_and_keeps_unknown_or_wrong_member_distinct()
    {
        using var fixture = JsonDocument.Parse(await File.ReadAllTextAsync(Environment.GetEnvironmentVariable("CRM_NOTIFICATION_FIXTURE_FILE")!)); var root = fixture.RootElement;
        Assert.Equal("isolated_notification_contract.v1", root.GetProperty("contract").GetString()); var endpoint = new Uri(root.GetProperty("base_url").GetString()!); Assert.True(endpoint.IsLoopback);
        const string tenant = "00000000-0000-4000-8000-000000000001", member = "00000000-0000-4000-8000-000000000002";
        using var http = new HttpClient(new HttpClientHandler { UseProxy = false, AllowAutoRedirect = false }); var client = new PowerXNotificationClient(endpoint.ToString(), tenant, new StaticServiceCredentialProvider(new(root.GetProperty("token").GetString()!)), http, "/api/v2");
        var request = new CreateNotificationRequest("fixture_lookup_title", "fixture_lookup_content", MemberUuid: member, IdempotencyKey: "fixture.sdk.lookup"); var created = await client.CreateAsync(new(tenant), request);
        var proof = await client.LookupAsync(new(tenant), new(request.IdempotencyKey!, member)); Assert.True(proof.Found); Assert.Equal(created.Uuid, proof.Notification!.Uuid); Assert.Equal(created.CreatedAt, proof.Notification.CreatedAt); Assert.Equal("present", proof.NotificationState); Assert.Equal("plugin:plugin.fixture.crm", proof.ServiceOwner);
        var missing = await client.LookupAsync(new(tenant), new("fixture.sdk.never_created", member)); Assert.False(missing.Found); Assert.Null(missing.Notification);
        var wrongMember = await Assert.ThrowsAsync<NotificationException>(() => client.LookupAsync(new(tenant), new(request.IdempotencyKey!, Guid.NewGuid().ToString()))); Assert.Equal(NotificationErrors.Forbidden, wrongMember.Code);
    }
    [IsolatedNotificationContractFact]
    public async Task Actual_core_http_and_postgres_receipt_matches_sdk_for_null_metadata_navigation_and_retry()
    {
        using var fixture = JsonDocument.Parse(await File.ReadAllTextAsync(Environment.GetEnvironmentVariable("CRM_NOTIFICATION_FIXTURE_FILE")!)); var root = fixture.RootElement;
        Assert.Equal("isolated_notification_contract.v1", root.GetProperty("contract").GetString());
        var endpoint = new Uri(root.GetProperty("base_url").GetString()!); Assert.True(endpoint.IsLoopback); Assert.Equal("http", endpoint.Scheme);
        const string tenant = "00000000-0000-4000-8000-000000000001"; const string member = "00000000-0000-4000-8000-000000000002";
        Assert.Equal(tenant, root.GetProperty("tenant_uuid").GetString()); Assert.Equal(member, root.GetProperty("member_uuid").GetString());
        using var http = new HttpClient(new HttpClientHandler { UseProxy = false, AllowAutoRedirect = false });
        var client = new PowerXNotificationClient(endpoint.ToString(), tenant, new StaticServiceCredentialProvider(new(root.GetProperty("token").GetString()!)), http, "/api/v2");
        var request = new CreateNotificationRequest("fixture_title", "fixture_content", MemberUuid: member, IdempotencyKey: "fixture.sdk.null_metadata");
        var first = await client.CreateAsync(new(tenant), request); var replayed = await client.CreateAsync(new(tenant), request);
        Assert.Equal(first.Uuid, replayed.Uuid); Assert.Null(first.Metadata); Assert.Equal(first.CreatedAt, replayed.CreatedAt);
        var conflict = await Assert.ThrowsAsync<NotificationException>(() => client.CreateAsync(new(tenant), request with { Content = "fixture_changed" })); Assert.Equal(NotificationErrors.IdempotencyConflict, conflict.Code);
        using var metadata = JsonDocument.Parse("{\"amount\":9999999999999999.99}");
        request = request with { IdempotencyKey = "fixture.sdk.navigation", Metadata = new Dictionary<string, JsonElement> { ["amount"] = metadata.RootElement.GetProperty("amount").Clone() }, Navigation = new("fixture_action", "/_p/plugin.fixture.crm/admin/approval?round=fixture") };
        var navigation = await client.CreateAsync(new(tenant), request); Assert.Equal(9999999999999999.99m, navigation.Metadata!["amount"].GetDecimal()); Assert.Equal(request.Navigation.Path, Assert.Single(navigation.Actions!).Params!["path"]);
        Assert.Equal(navigation.Uuid, (await client.CreateAsync(new(tenant), request)).Uuid);
    }
}
public sealed class IsolatedNotificationContractFactAttribute : FactAttribute
{ public IsolatedNotificationContractFactAttribute() { if (string.IsNullOrWhiteSpace(Environment.GetEnvironmentVariable("CRM_NOTIFICATION_FIXTURE_FILE"))) Skip = "CRM_NOTIFICATION_FIXTURE_FILE isolated loopback fixture required"; } }
