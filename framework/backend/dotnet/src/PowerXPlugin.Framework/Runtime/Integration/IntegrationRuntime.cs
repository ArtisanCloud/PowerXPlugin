using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Integration;

/// <summary>
/// One startup-selected Integration Gateway binding. This mirrors Go's
/// runtime/integration Factory and forbids a request-time local fallback.
/// </summary>
public static class IntegrationRuntimeExtensions
{
    public static IServiceCollection AddPowerXIntegrationRuntime(
        this IServiceCollection services,
        ProviderMode mode,
        Func<IServiceProvider, IIntegrationGateway> localFactory,
        Func<IServiceProvider, IIntegrationGateway> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(localFactory);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        return services.AddPowerXRuntime("integration", mode, localFactory, delegatedFactory);
    }
}
