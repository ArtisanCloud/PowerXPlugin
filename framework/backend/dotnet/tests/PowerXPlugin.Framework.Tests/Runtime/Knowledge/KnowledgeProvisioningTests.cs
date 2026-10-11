using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Knowledge;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Knowledge;

public sealed class KnowledgeProvisioningTests
{
    private const string Uuid = "11111111-1111-4111-8111-111111111111";
    private const string Profile = "{\"uuid\":\"" + Uuid + "\",\"key\":\"p1_general\",\"version\":2}";
    private const string Profiles = "{\"ingestion\":" + Profile + ",\"index\":" + Profile + ",\"rag\":" + Profile + "}";
    private const string Quotas = "{\"cpu_cores\":4,\"storage_gb\":200,\"ingestion_concurrency\":2}";
    private const string Catalog = "{\"data\":{\"catalog\":{\"version\":\"1\",\"source\":\"powerx_core\",\"scenes\":[],\"strategy_packages\":[{\"key\":\"A_simple\",\"label\":\"fixture\",\"recommended_profile_key\":\"p1_general\",\"recommended_scenes\":[\"support_faq\"],\"dependencies\":{\"index\":[\"dense\"],\"runtime\":[],\"assets\":[\"embedding\"]},\"profiles\":" + Profiles + ",\"available\":true,\"unavailable_reasons\":[],\"activation_dependencies\":[\"dense\",\"embedding\"]}],\"policy_templates\":[{\"uuid\":\"" + Uuid + "\",\"name\":\"fixture\",\"version\":\"v1\"}],\"default_policy_template_uuid\":\"" + Uuid + "\",\"quota_defaults\":" + Quotas + ",\"quota_minimums\":{\"cpu_cores\":1,\"storage_gb\":50,\"ingestion_concurrency\":1},\"quota_override_allowed\":true}}}";
    private const string Created = "{\"data\":{\"item\":{\"space_uuid\":\"22222222-2222-4222-8222-222222222222\",\"name\":\"fixture\",\"status\":\"pending_iam\",\"department_uuid\":\"" + Uuid + "\",\"strategy_key\":\"A_simple\",\"scene_key\":\"support_faq\",\"policy_template_uuid\":\"" + Uuid + "\",\"profiles\":" + Profiles + ",\"quotas\":" + Quotas + "}}}";

    [Theory]
    [InlineData("ApiKey")]
    [InlineData("Bearer")]
    public async Task Typed_catalog_and_create_use_fixed_routes_service_credentials_and_no_overrides(string scheme)
    {
        var calls = 0;
        var client = Client(scheme, async request =>
        {
            calls++;
            Assert.Equal(scheme + " service-credential", request.Headers.Authorization!.ToString());
            Assert.Equal(string.Empty, request.RequestUri!.Query);
            Assert.False(request.Headers.Contains("tenant_uuid"));
            if (request.Method == HttpMethod.Get)
            {
                Assert.Equal("/api/v1/tenant/knowledge/catalog", request.RequestUri.AbsolutePath);
                Assert.Null(request.Content);
                return Response(200, Catalog);
            }
            Assert.Equal(HttpMethod.Post, request.Method);
            Assert.Equal("/api/v1/tenant/knowledge/spaces", request.RequestUri.AbsolutePath);
            using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
            Assert.Equal(3, body.RootElement.EnumerateObject().Count());
            Assert.Equal(Uuid, body.RootElement.GetProperty("department_uuid").GetString());
            Assert.Equal("A_simple", body.RootElement.GetProperty("strategy_key").GetString());
            return Response(200, Created);
        });
        Assert.True(client.Capabilities.Supports(KnowledgeOperations.Catalog));
        Assert.True(client.Capabilities.Supports(KnowledgeOperations.Create));
        IKnowledgeProvisioningService service = client;
        var catalog = await service.GetCatalogAsync();
        Assert.True(catalog.StrategyPackages[0].Available);
        Assert.Equal(2, catalog.StrategyPackages[0].Profiles.Ingestion!.Version);
        Assert.Equal(new[] { "dense", "embedding" }, catalog.StrategyPackages[0].ActivationDependencies);
        Assert.Equal(Uuid, catalog.DefaultPolicyTemplateUuid);
        Assert.True(catalog.QuotaOverrideAllowed);
        var item = await service.CreateSpaceAsync(new CreateKnowledgeSpaceRequest("fixture", Uuid, "A_simple"));
        Assert.Equal("pending_iam", item.Status);
        Assert.Equal(2, calls);
    }
    [Theory]
    [InlineData(400, "KNOWLEDGE_INVALID_ARGUMENT")]
    [InlineData(401, "KNOWLEDGE_UNAUTHORIZED")]
    [InlineData(403, "KNOWLEDGE_FORBIDDEN")]
    [InlineData(409, "KNOWLEDGE_SPACE_CONFLICT")]
    [InlineData(412, "KNOWLEDGE_STRATEGY_UNAVAILABLE")]
    [InlineData(503, "KNOWLEDGE_MIGRATION_REQUIRED")]
    [InlineData(500, "CORE_CUSTOM_FAILURE")]
    public async Task Core_status_machine_code_and_trace_survive_unchanged(int status, string code)
    {
        var calls = 0;
        var client = Client("ApiKey", request =>
        {
            calls++;
            var response = Response(status, "{\"error_code\":\"" + code + "\",\"request_id\":\"body-request\"}");
            response.Headers.Add("X-Trace-Id", "core-trace");
            return Task.FromResult(response);
        });
        var error = await Assert.ThrowsAsync<KnowledgeException>(() => client.CreateSpaceAsync(new CreateKnowledgeSpaceRequest("fixture", Uuid, "A_simple")));
        Assert.Equal(status, error.StatusCode);
        Assert.Equal(code, error.Code);
        Assert.Equal("core-trace", error.TraceId);
        Assert.Equal(1, calls);
    }
    [Fact]
    public async Task Invalid_inputs_are_rejected_without_a_host_call()
    {
        var calls = 0;
        var client = Client("ApiKey", request => { calls++; return Task.FromResult(Response(200, Created)); });
        foreach (var request in new[] {
            new CreateKnowledgeSpaceRequest(" ", Uuid, "A_simple"),
            new CreateKnowledgeSpaceRequest(string.Concat(Enumerable.Repeat("界", 43)), Uuid, "A_simple"),
            new CreateKnowledgeSpaceRequest("fixture", "1", "A_simple"),
            new CreateKnowledgeSpaceRequest("fixture", Uuid, "A_simple", PolicyTemplateUuid: "default/v1"),
            new CreateKnowledgeSpaceRequest("fixture", Uuid, "A_simple", Quotas: new(1, 49, 1)) })
        {
            var error = await Assert.ThrowsAsync<KnowledgeException>(() => client.CreateSpaceAsync(request));
            Assert.Equal(KnowledgeErrors.InvalidArgument, error.Code);
        }
        Assert.Equal(0, calls);
    }
    [Fact]
    public async Task Missing_UUIDs_are_not_synthesized_and_request_id_is_preserved()
    {
        var client = Client("ApiKey", request => Task.FromResult(Response(200, Created.Replace("22222222-2222-4222-8222-222222222222", "2", StringComparison.Ordinal))));
        var invalid = await Assert.ThrowsAsync<KnowledgeException>(() => client.CreateSpaceAsync(new CreateKnowledgeSpaceRequest("fixture", Uuid, "A_simple")));
        Assert.Equal(KnowledgeErrors.InvalidResponse, invalid.Code);
        client = Client("ApiKey", request => Task.FromResult(Response(403, "{\"reason_code\":\"DENIED\",\"request_id\":\"core-request\"}")));
        var error = await Assert.ThrowsAsync<KnowledgeException>(() => client.GetCatalogAsync());
        Assert.Equal("DENIED", error.Code);
        Assert.Equal(403, error.StatusCode);
        Assert.Equal("core-request", error.TraceId);
    }
    [Fact]
    public async Task Unavailable_strategy_can_omit_profiles_without_a_default_mapping()
    {
        var unavailable = Catalog.Replace(Profiles, "{}", StringComparison.Ordinal).Replace("\"available\":true", "\"available\":false", StringComparison.Ordinal).Replace("\"unavailable_reasons\":[]", "\"unavailable_reasons\":[\"rag_profile_not_published\"]", StringComparison.Ordinal);
        var client = Client("ApiKey", request => Task.FromResult(Response(200, unavailable)));
        var catalog = await client.GetCatalogAsync();
        Assert.False(catalog.StrategyPackages[0].Available);
        Assert.Null(catalog.StrategyPackages[0].Profiles.Rag);
        Assert.Equal(new[] { "rag_profile_not_published" }, catalog.StrategyPackages[0].UnavailableReasons);
    }
    private static PowerXKnowledgeClient Client(string scheme, Func<HttpRequestMessage, Task<HttpResponseMessage>> handler) => new("https://core.example", Uuid, new StaticServiceCredentialProvider(new ServiceCredential("service-credential", scheme)), new HttpClient(new Handler(handler)));
    private static HttpResponseMessage Response(int status, string body) => new((HttpStatusCode)status) { Content = new StringContent(body, Encoding.UTF8, "application/json") };
    private sealed class Handler(Func<HttpRequestMessage, Task<HttpResponseMessage>> handler) : HttpMessageHandler
    { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => handler(request); }
}
