using System.Text.Json;
using PowerXPlugin.Framework.EventBridge;

namespace PowerXPlugin.Framework.Runtime.EventFabric;

/// <summary>
/// Local implementation maps an in-process event to a delivery. A Nack throws
/// back through LocalEventBridge so the original local scheduler operation can
/// retry; it never pretends to persist or host-deliver an event.
/// </summary>
public sealed class LocalEventFabricAdapter : IEventRuntime
{
    private readonly IEventDispatcher _dispatcher;

    public LocalEventFabricAdapter(IEventDispatcher dispatcher) => _dispatcher = dispatcher ?? throw new ArgumentNullException(nameof(dispatcher));

    public async Task ConsumeAsync(EventFabricSubscription subscription, EventFabricHandler handler, CancellationToken ct = default)
    {
        ValidateSubscription(subscription);
        ArgumentNullException.ThrowIfNull(handler);
        foreach (var topic in subscription.Topics.Where(topic => !string.IsNullOrWhiteSpace(topic)).Select(topic => topic.Trim()))
        {
            await _dispatcher.RegisterHandlerAsync(topic, async (evt, token) =>
            {
                var payload = JsonSerializer.SerializeToUtf8Bytes(evt.Payload);
                var delivery = new EventFabricDelivery(
                    Guid.NewGuid().ToString("N"), Guid.NewGuid().ToString("N"), evt.Topic,
                    evt.Meta.TraceID ?? string.Empty, evt.Meta.PayloadVersion, payload, "application/json",
                    new Dictionary<string, string>(), 1, 1, 0, null, subscription.SubscriberId);
                if (await handler(delivery, token) == EventFabricDisposition.Nack)
                    throw new EventFabricAdapterException(EventFabricErrors.CodeNack, "Local event delivery was rejected.");
            }, ct);
        }
    }

    private static void ValidateSubscription(EventFabricSubscription subscription)
    {
        if (subscription is null || string.IsNullOrWhiteSpace(subscription.TenantUuid) || string.IsNullOrWhiteSpace(subscription.SubscriberId) || subscription.Topics is null || !subscription.Topics.Any(topic => !string.IsNullOrWhiteSpace(topic)))
            throw new EventFabricAdapterException(EventFabricErrors.CodeInvalidSubscription);
    }
}
