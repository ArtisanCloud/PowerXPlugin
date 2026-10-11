using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using PowerXPlugin.Framework.Runtime.Capability;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Customer;
using PowerXPlugin.Framework.Runtime.Metadata;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Customer;

public sealed class CustomerManagementTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001", Customer = "11111111-1111-4111-8111-111111111111", Contact = "22222222-2222-4222-8222-222222222222";
    [Theory]
    [InlineData("2026-09-27 13:38:23.699407+08")]
    [InlineData("2026-09-27 13:38:23.699407+08:00")]
    [InlineData("2026-09-27T05:38:23.699407Z")]
    public async Task Core_account_timestamps_preserve_the_actual_instant(string timestamp)
    {
        using var http = new HttpClient(new Handler(_ => Task.FromResult(Invocation(new { items = new[] { new { uuid = Customer, created_at = timestamp, updated_at = timestamp } }, page = 1, page_size = 20, total = 1 }))));
        var item = Assert.Single((await Client(http).ListAsync(new(Tenant), new())).Items);
        Assert.Equal(new DateTime(2026, 9, 27, 5, 38, 23, DateTimeKind.Utc).AddTicks(6994070), item.CreatedAt.ToUniversalTime());
        Assert.Equal(item.CreatedAt, item.UpdatedAt);
    }
    [Fact]
    public async Task Malformed_core_timestamp_fails_explicitly()
    {
        using var http = new HttpClient(new Handler(_ => Task.FromResult(Invocation(new { items = new[] { new { uuid = Customer, created_at = "2026-09-27 invalid" } }, page = 1, page_size = 20, total = 1 }))));
        var error = await Assert.ThrowsAsync<CustomerManagementException>(() => Client(http).ListAsync(new(Tenant), new()));
        Assert.Equal("CUSTOMER_ACCOUNT_INVALID_RESPONSE", error.Code);
    }
    [Fact]
    public async Task Account_management_uses_fixed_service_contract_and_preserves_explicit_clearing()
    {
        using var http = new HttpClient(new Handler(async request =>
        {
            Assert.Equal("/api/v1/tenant/invocations", request.RequestUri!.AbsolutePath); Assert.Equal("ApiKey service-key", request.Headers.Authorization!.ToString());
            using var json = JsonDocument.Parse(await request.Content!.ReadAsStringAsync()); var root = json.RootElement;
            Assert.Equal("com.corex.customer.accounts.service_manage", root.GetProperty("capability_id").GetString());
            Assert.Equal("core_internal", root.GetProperty("preferred_protocol").GetString());
            var payload = root.GetProperty("payload"); Assert.Equal("INVOKE", payload.GetProperty("method").GetString()); Assert.Equal("core://customer/accounts", payload.GetProperty("endpoint").GetString());
            var body = payload.GetProperty("body"); Assert.Equal("update", body.GetProperty("operation").GetString()); Assert.Equal("", body.GetProperty("primary_email").GetString()); Assert.False(body.TryGetProperty("display_name", out _)); Assert.False(body.TryGetProperty("tenant_uuid", out _));
            return Invocation(new { item = new { uuid = Customer, type = "person", primary_contact_uuid = Contact, primary_email = "", display_name = "fixture" } });
        }));
        var item = await Client(http).UpdateAsync(new(Tenant, "test-trace"), Customer, new(PrimaryEmail: "")); Assert.Equal(Customer, item.CustomerUuid); Assert.Equal(Contact, item.PrimaryContactUuid);
    }
    [Fact]
    public async Task Contact_wire_uuid_and_customer_scope_are_checked()
    {
        using var http = new HttpClient(new Handler(async request =>
        {
            using var json = JsonDocument.Parse(await request.Content!.ReadAsStringAsync()); var root = json.RootElement;
            Assert.Equal("com.corex.customer.contacts.service_read", root.GetProperty("capability_id").GetString()); Assert.Equal("core://customer/contacts", root.GetProperty("payload").GetProperty("endpoint").GetString());
            Assert.Equal("get", root.GetProperty("payload").GetProperty("body").GetProperty("operation").GetString());
            return Invocation(new { item = new { uuid = Contact, customer_uuid = Customer, tenant_uuid = Tenant, display_name = "fixture", status = "active", roles = Array.Empty<string>(), tags = Array.Empty<string>(), createdAt = "2026-09-27T05:38:23.699407Z", updatedAt = "2026-09-27T05:38:23.699407Z" } });
        }));
        var item = await Client(http).GetAsync(new(Tenant), Customer, Contact); Assert.Equal(Contact, item.ContactUuid); Assert.Equal(Customer, item.CustomerUuid);
        Assert.Equal(2026, item.CreatedAt.Year); Assert.Equal(item.CreatedAt, item.UpdatedAt);
    }
    [Fact]
    public async Task Actual_contact_wire_from_another_tenant_is_rejected()
    {
        using var http = new HttpClient(new Handler(_ => Task.FromResult(Invocation(new { items = new[] { new { uuid = Contact, customer_uuid = Customer, tenant_uuid = Customer, createdAt = "2026-09-27T05:38:23.699407Z" } }, page = 1, page_size = 20, total = 1 }))));
        var error = await Assert.ThrowsAsync<CustomerManagementException>(() => Client(http).ListAsync(new(Tenant), Customer, new()));
        Assert.Equal("CONTACT_CUSTOMER_MISMATCH", error.Code);
    }
    [Fact]
    public async Task External_identity_inspection_uses_read_grant_and_never_creates_on_miss()
    {
        var calls = 0;
        using var http = new HttpClient(new Handler(async request =>
        {
            calls++; using var json = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
            Assert.Equal("com.corex.customer.external_identities.service_read", json.RootElement.GetProperty("capability_id").GetString());
            Assert.Equal("core://customer/external-identities", json.RootElement.GetProperty("payload").GetProperty("endpoint").GetString()); return Invocation(new { found = false });
        }));
        Assert.False((await Client(http).LookupAsync(new(Tenant), "shop:fixture:customer:1")).Found); Assert.Equal(1, calls);
    }
    [Fact]
    public async Task Tenant_mismatch_and_invalid_contact_are_rejected_before_transport()
    {
        using var http = new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("unexpected transport")));
        await Assert.ThrowsAsync<CustomerManagementException>(() => Client(http).ListAsync(new(Customer), new()));
        await Assert.ThrowsAsync<CustomerManagementException>(() => Client(http).CreateAsync(new(Tenant), Customer, new("fixture", Roles: ["purchaser"])));
    }
    [Fact]
    public async Task Host_rejection_is_propagated_without_local_fallback()
    {
        using var http = new HttpClient(new Handler(_ => Task.FromResult(new HttpResponseMessage(HttpStatusCode.Forbidden))));
        var error = await Assert.ThrowsAsync<CapabilityGrantException>(() => Client(http).ListAsync(new(Tenant), new())); Assert.Equal(CapabilityGrantErrors.Forbidden, error.Code);
    }
    [Fact]
    public async Task Metadata_reads_formal_payload_snake_fields_and_flat_display()
    {
        using var http = new HttpClient(new Handler(request =>
        {
            Assert.Equal("/api/v1/tenant/metadata/dictionaries", request.RequestUri!.AbsolutePath);
            return Task.FromResult(Response(new { data = new { payload = new { items = new[] { new { uuid = Customer, @namespace = "crm.fixture", module = "crm", name_i18n = new { en = "fixture" }, status = "enabled", item_count = 2, display_name = "fixture" } }, pagination = new { page = 1, page_size = 20, total = 1 } } } }));
        }));
        var page = await Metadata(http).ListDictionaryNamespacesAsync(new(Tenant), new()); var item = Assert.Single(page.Items); Assert.Equal(Customer, item.Uuid); Assert.Equal(2, item.ItemCount); Assert.Equal("fixture", item.Display.DisplayName);
    }
    [Fact]
    public async Task Metadata_missing_payload_is_not_reported_as_an_empty_success()
    {
        using var http = new HttpClient(new Handler(_ => Task.FromResult(Response(new { data = new { items = Array.Empty<object>() } }))));
        var error = await Assert.ThrowsAsync<MetadataException>(() => Metadata(http).ListDictionaryNamespacesAsync(new(Tenant), new())); Assert.Equal(MetadataErrors.InvalidResponse, error.Code);
    }
    [Fact]
    public async Task Metadata_tag_update_uses_typed_patch_without_tenant_or_endpoint_inputs()
    {
        using var http = new HttpClient(new Handler(async request =>
        {
            Assert.Equal(HttpMethod.Patch, request.Method); Assert.Equal("/api/v1/tenant/metadata/tags/" + Customer, request.RequestUri!.AbsolutePath);
            using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync()); Assert.False(body.RootElement.TryGetProperty("tenant_uuid", out _));
            return Response(new { data = new { payload = new { uuid = Customer, @namespace = "crm.labels", resource_type = "crm.fixture", code = "vip", status = "enabled", color = "#ffffff", display_name = "fixture" } } });
        }));
        var item = await Metadata(http).UpdateTagAsync(new(Tenant), new(Customer, Color: "#ffffff")); Assert.Equal(Customer, item.Uuid); Assert.Equal("crm.fixture", item.ResourceType);
    }
    private static PowerXCustomerManagementClient Client(HttpClient http) => new(new PowerXCapabilityRegistryClient("http://core", Credentials(), http), Tenant);
    private static PowerXMetadataClient Metadata(HttpClient http) => new("http://core", Tenant, Credentials(), http);
    private static StaticServiceCredentialProvider Credentials() => new(new("service-key", "ApiKey"));
    private static HttpResponseMessage Invocation(object payload) => Response(new { code = 200, data = new { trace_id = "fixture-trace", status = "completed", protocol_used = "core_internal", fallback_used = false, payload } });
    private static HttpResponseMessage Response(object body) => new(HttpStatusCode.OK) { Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json") };
    private sealed class Handler(Func<HttpRequestMessage, Task<HttpResponseMessage>> send) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => send(request); }
}
