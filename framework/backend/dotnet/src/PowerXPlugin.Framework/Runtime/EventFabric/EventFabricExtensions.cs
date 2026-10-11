using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.EventFabric;

public static class EventFabricExtensions
{
    public static IServiceCollection AddPowerXEventRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, IEventRuntime> localFactory, Func<IServiceProvider, IEventRuntime> delegatedFactory) =>
        services.AddPowerXRuntime("event-fabric", mode, localFactory, delegatedFactory);

    /// <summary>Registers the single host Event Fabric adapter selected during startup.</summary>
    public static IServiceCollection AddPowerXEventFabric(this IServiceCollection services, Func<IServiceProvider, IEventFabricSubscriber> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        services.AddSingleton(delegatedFactory);
        return services;
    }
}
