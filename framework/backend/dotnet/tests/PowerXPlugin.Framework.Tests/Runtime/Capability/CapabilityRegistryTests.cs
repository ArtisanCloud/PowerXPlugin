using System.Net;
using System.Text;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Capability;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Capability;

public sealed class CapabilityRegistryTests
{
    [Fact]
    public async Task Delegated_list_uses_formal_catalog_and_service_credential()
    {
        var client = new PowerXCapabilityRegistryClient("https://core.example", new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            Assert.Equal("/api/v1/tenant/capabilities", request.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer sts", request.Headers.Authorization!.ToString());
            return Json("{\"data\":{\"items\":[{\"capabilityId\":\"crm.read\",\"pluginId\":\"crm\",\"pluginVersion\":\"1\",\"title\":\"Read\",\"source\":\"corex\",\"protocols\":[],\"status\":\"published\"}]}}");
        })));
        var list = await client.ListAsync(new CapabilityListQuery(Source: "corex"));
        Assert.Equal("crm.read", Assert.Single(list).CapabilityId);
    }

    [Fact]
    public void Runtime_delegated_does_not_evaluate_local_store()
    {
        var localCalled = false;
        var remote = new Store();
        var services = new ServiceCollection();
        services.AddPowerXCapabilityRuntime(ProviderMode.Delegated, _ => { localCalled = true; return new LocalCapabilityRegistry(new Store()); }, _ => remote);
        using var provider = services.BuildServiceProvider();
        Assert.Same(remote, provider.GetRequiredService<ICapabilityRegistry>());
        Assert.False(localCalled);
    }
    private static HttpResponseMessage Json(string text) => new(HttpStatusCode.OK) { Content = new StringContent(text, Encoding.UTF8, "application/json") };
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> send) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(send(request)); }
    private sealed class Store : ILocalCapabilityStore { public Task<IReadOnlyList<CapabilityDescriptor>> ListAsync(CapabilityListQuery q, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<CapabilityDescriptor>>([]); public Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> i, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<CapabilityGrantStatus>>([]); public Task<CapabilityResolution> ResolveAsync(CapabilityResolveQuery q, CancellationToken ct = default) => throw new NotImplementedException(); public Task<CapabilityInvokeResult> InvokeAsync(CapabilityInvokeRequest r, CancellationToken ct = default) => throw new NotImplementedException(); public Task<CapabilityInvocation> GetInvocationAsync(string id, CancellationToken ct = default) => throw new NotImplementedException(); }
}
