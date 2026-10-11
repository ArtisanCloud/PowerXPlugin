using PowerXPlugin.Framework.Runtime.Integration;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Integration;

public sealed class LocalIntegrationGatewayTests
{
    [Fact]
    public async Task Local_gateway_rejects_route_not_owned_by_plugin()
    {
        var gateway = new LocalIntegrationGateway(new Store(false));
        var exception = await Assert.ThrowsAsync<IntegrationAdapterException>(() => gateway.InvokeRouteAsync("foreign-route", new IntegrationRouteInvocation(new Dictionary<string, object?> { ["id"] = "1" })));
        Assert.Equal(IntegrationErrors.CodeNotFound, exception.Code);
    }

    [Fact]
    public async Task Local_gateway_invokes_verified_owned_route()
    {
        var gateway = new LocalIntegrationGateway(new Store(true));
        var result = await gateway.InvokeRouteAsync("crm-order", new IntegrationRouteInvocation(new Dictionary<string, object?> { ["id"] = "1" }));
        Assert.Equal("crm.order", result.RoutedCapabilityId);
    }

    private sealed class Store(bool owned) : ILocalIntegrationRouteStore
    {
        public Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<IntegrationRouteSummary>>([]);
        public Task<IntegrationRouteDetail?> GetRouteAsync(string routeSlug, CancellationToken ct = default) => Task.FromResult<IntegrationRouteDetail?>(null);
        public Task<bool> IsRouteOwnedAsync(string routeSlug, CancellationToken ct = default) => Task.FromResult(owned);
        public Task<IntegrationRouteResult> InvokeOwnedRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default) => Task.FromResult(new IntegrationRouteResult(new Dictionary<string, object?>(), "crm.order", "local", "trace-1", DateTimeOffset.UtcNow));
    }
}
