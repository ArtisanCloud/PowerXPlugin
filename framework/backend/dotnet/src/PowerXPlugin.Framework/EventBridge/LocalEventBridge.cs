using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.DependencyInjection.Extensions;

namespace PowerXPlugin.Framework.EventBridge;

/// <summary>
/// In-process EventBridge used only by a startup-selected local runtime.
/// Dispatch is synchronous so a failed consumer leaves the originating local
/// scheduler job due for a retry; it is never a substitute for a Host TaskBus.
/// </summary>
public sealed class LocalEventBridge : IEventEmitter, IEventDispatcher
{
    private readonly object _gate = new();
    private readonly Dictionary<string, List<Func<FrameworkEvent, CancellationToken, Task>>> _handlers = new(StringComparer.Ordinal);

    public Task RegisterHandlerAsync(string topic, Func<FrameworkEvent, CancellationToken, Task> handler, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(topic)) throw new EventBridgeAdapterException(EventBridgeErrors.CodeInvalidTopic);
        ArgumentNullException.ThrowIfNull(handler);
        ct.ThrowIfCancellationRequested();
        lock (_gate)
        {
            if (!_handlers.TryGetValue(topic.Trim(), out var handlers))
            {
                handlers = [];
                _handlers[topic.Trim()] = handlers;
            }
            handlers.Add(handler);
        }
        return Task.CompletedTask;
    }

    public Task EmitAsync(FrameworkEvent evt, CancellationToken ct = default) => DispatchAsync(evt, ct);

    public async Task DispatchAsync(FrameworkEvent evt, CancellationToken ct = default)
    {
        if (evt == null || string.IsNullOrWhiteSpace(evt.Topic))
            throw new EventBridgeAdapterException(EventBridgeErrors.CodeInvalidTopic);
        Func<FrameworkEvent, CancellationToken, Task>[] handlers;
        lock (_gate)
            handlers = _handlers.TryGetValue(evt.Topic.Trim(), out var registered) ? registered.ToArray() : [];
        foreach (var handler in handlers)
        {
            ct.ThrowIfCancellationRequested();
            await handler(evt, ct);
        }
    }
}

public static class EventBridgeServiceCollectionExtensions
{
    /// <summary>Registers the local bridge once; it never enables a host fallback.</summary>
    public static IServiceCollection AddLocalEventBridge(this IServiceCollection services)
    {
        ArgumentNullException.ThrowIfNull(services);
        services.TryAddSingleton<LocalEventBridge>();
        services.TryAddSingleton<IEventEmitter>(sp => sp.GetRequiredService<LocalEventBridge>());
        services.TryAddSingleton<IEventDispatcher>(sp => sp.GetRequiredService<LocalEventBridge>());
        return services;
    }
}
