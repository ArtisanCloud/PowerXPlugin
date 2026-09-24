using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.AI;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.AI;

public sealed class AiRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";

    [Fact]
    public async Task Delegated_llm_uses_service_credential_and_snake_case_contract()
    {
        HttpRequestMessage? seen = null; string? body = null;
        var client = new PowerXGenerativeClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            seen = request; body = request.Content!.ReadAsStringAsync().GetAwaiter().GetResult();
            return new(HttpStatusCode.OK) { Content = new StringContent("{\"data\":{\"type\":\"text\",\"text\":\"ok\",\"finish_reason\":\"stop\"}}", Encoding.UTF8, "application/json") };
        })));
        var output = await client.InvokeLlmAsync(new(Tenant), new("model-a", [new("text", "hello", Role: "user")]));
        Assert.Equal("ok", output.Text);
        Assert.Equal("Bearer sts", seen!.Headers.Authorization!.ToString());
        Assert.Equal("/api/v1/ai/llm/invoke", seen.RequestUri!.AbsolutePath);
        Assert.Contains("model_key", body);
    }

    [Fact]
    public async Task Delegated_rejects_tenant_override_before_transport()
    {
        var client = new PowerXGenerativeClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(_ => throw new Xunit.Sdk.XunitException("transport must not be called"))));
        var error = await Assert.ThrowsAsync<AiRuntimeException>(() => client.ListModelsAsync(new("00000000-0000-4000-8000-000000000002")));
        Assert.Equal(AiErrors.TenantMismatch, error.Code);
    }

    [Fact]
    public async Task Delegated_transport_cancellation_is_not_reclassified_as_upstream_failure()
    {
        var client = new PowerXGenerativeClient("https://core", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new CancellableHandler()));
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => client.ListModelsAsync(new(Tenant), ct: cancellation.Token));
    }

    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handle) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => Task.FromResult(handle(request)); }
    private sealed class CancellableHandler : HttpMessageHandler { protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) { await Task.Delay(Timeout.InfiniteTimeSpan, cancellationToken); throw new InvalidOperationException(); } }
}
