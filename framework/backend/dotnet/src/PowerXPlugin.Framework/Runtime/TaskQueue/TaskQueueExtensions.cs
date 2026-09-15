using Microsoft.Extensions.DependencyInjection;

namespace PowerXPlugin.Framework.Runtime.TaskQueue;

public static class TaskQueueExtensions
{
    /// <summary>Registers exactly one queue adapter selected at startup.</summary>
    public static IServiceCollection AddPowerXTaskQueue(this IServiceCollection services, TaskQueueAdapterMode mode, Func<IServiceProvider, ITaskQueue> delegatedFactory)
    {
        ArgumentNullException.ThrowIfNull(services);
        ArgumentNullException.ThrowIfNull(delegatedFactory);
        switch (mode)
        {
            case TaskQueueAdapterMode.Local:
                services.AddSingleton<InMemoryTaskQueue>();
                services.AddSingleton<ITaskQueue>(sp => sp.GetRequiredService<InMemoryTaskQueue>());
                break;
            case TaskQueueAdapterMode.Delegated:
                services.AddSingleton(delegatedFactory);
                break;
            default:
                throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidMode);
        }
        return services;
    }
}
