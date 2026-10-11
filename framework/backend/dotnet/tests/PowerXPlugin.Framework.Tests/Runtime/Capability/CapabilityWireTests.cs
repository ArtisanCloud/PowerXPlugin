using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Capability;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.WSBus;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Capability;

public sealed class CapabilityWireTests
{
    [Fact]
    public async Task Discovery_is_separate_from_granted_formal_contracts()
    {
        using var http = new HttpClient(new Handler(request =>
        {
            Assert.Equal("/api/v1/tenant/capabilities/catalog", request.RequestUri!.AbsolutePath);
            Assert.Equal("?page=2&page_size=200", request.RequestUri.Query);
            return Json("{\"code\":200,\"data\":{\"items\":[{\"capability_id\":\"com.corex.test.read\",\"title\":\"test\",\"description\":\"\",\"source\":\"corex\",\"categories\":[],\"intents\":[],\"tool_scope\":[],\"status\":\"published\"}]}}");
        }));
        var client = Client(http);
        Assert.Equal("com.corex.test.read", Assert.Single(await client.ListPublishedAsync(2)).CapabilityId);
    }

    [Theory]
    [InlineData("{}")]
    [InlineData("[]")]
    [InlineData("{\"data\":null}")]
    [InlineData("{\"success\":false,\"data\":{\"items\":[]}}")]
    [InlineData("{\"data\":{}}")]
    [InlineData("invalid-json")]
    public async Task Malformed_or_rejected_catalog_is_a_stable_failure(string body)
    {
        using var http = new HttpClient(new Handler(_ => Json(body)));
        var error = await Assert.ThrowsAsync<CapabilityGrantException>(() => Client(http).ListAsync(new()));
        Assert.Equal(CapabilityGrantErrors.InvalidResponse, error.Code);
    }

    [Fact]
    public async Task Invocation_uses_snake_case_without_rewriting_payload_keys()
    {
        using var http = new HttpClient(new Handler(request =>
        {
            using var body = JsonDocument.Parse(request.Content!.ReadAsStringAsync().Result);
            Assert.Equal("com.corex.test.read", body.RootElement.GetProperty("capability_id").GetString());
            Assert.Equal(1, body.RootElement.GetProperty("payload").GetProperty("businessKey").GetInt32());
            Assert.False(body.RootElement.TryGetProperty("capabilityId", out _));
            return Json("{\"data\":{\"trace_id\":\"trace\",\"status\":\"completed\",\"protocol_used\":\"http\",\"fallback_used\":false,\"payload\":{},\"result\":{}}}");
        }));
        var result = await Client(http).InvokeAsync(new("com.corex.test.read", null, null, "http", null, null, null, new Dictionary<string, object?> { ["businessKey"] = 1 }, null));
        Assert.Equal("trace", result.TraceId); Assert.Equal("http", result.ProtocolUsed);
    }

    [Fact]
    public async Task WS_bus_rejects_tenant_override_before_sending_credentials()
    {
        using var http = new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("unexpected host request")));
        var client = new PowerXWSBusClient("https://core.example", "00000000-0000-0000-0000-000000000001", Credentials(), http);
        var error = await Assert.ThrowsAsync<WSBusException>(() => client.GrantAndPublishAsync("_topic.system.notification", new { }, new("00000000-0000-0000-0000-000000000002", null, "trace")));
        Assert.Equal(403, error.Status);
    }

    private static PowerXCapabilityRegistryClient Client(HttpClient http) => new("https://core.example", Credentials(), http);
    private static StaticServiceCredentialProvider Credentials() => new(new ServiceCredential("test-service-key", "ApiKey"));
    private static HttpResponseMessage Json(string value) => new(HttpStatusCode.OK) { Content = new StringContent(value, Encoding.UTF8, "application/json") };
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> send) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(send(request));
    }
}
