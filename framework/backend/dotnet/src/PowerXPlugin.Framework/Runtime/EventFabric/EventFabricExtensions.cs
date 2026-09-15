using Microsoft.Extensions.DependencyInjection;

namespace PowerXPlugin.Framework.Runtime.EventFabric;

public static class EventFabricExtensions
{
    /// <summary>Registers the single host Event Fabric adapter selected during startup.</summary>
    public static IServiceCollection AddPowerXEventFabric(this IServiceCollection services, Func<IServiceProvider, IEventFabricSubscriber> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        services.AddSingleton(delegatedFactory);
        return services;
    }
}
