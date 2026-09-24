using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.TaskQueue;

public static class TaskQueueExtensions
{
    /// <summary>Registers exactly one queue adapter selected at startup.</summary>
    public static IServiceCollection AddPowerXTaskQueue(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, ITaskQueue> localFactory, Func<IServiceProvider, ITaskQueue> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        ArgumentNullException.ThrowIfNull(localFactory);
        return services.AddPowerXRuntime<ITaskQueue>("task-queue", mode, localFactory, delegatedFactory);
    }
}
