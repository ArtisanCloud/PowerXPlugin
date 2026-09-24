using PowerXPlugin.Framework.EventBridge;
using PowerXPlugin.Framework.Runtime.EventFabric;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.EventFabric;

public sealed class LocalEventFabricAdapterTests
{
    [Fact]
    public async Task Local_delivery_acknowledges_through_registered_bridge_handler()
    {
        var bridge = new LocalEventBridge();
        var runtime = new LocalEventFabricAdapter(bridge);
        EventFabricDelivery? received = null;
        await runtime.ConsumeAsync(new EventFabricSubscription("tenant-a", "plugin-a", ["orders.created"]), (delivery, _) =>
        {
            received = delivery;
            return Task.FromResult(EventFabricDisposition.Ack);
        });

        await bridge.EmitAsync(new FrameworkEvent { Topic = "orders.created", Meta = new EventMeta { TenantUUID = "tenant-a", TraceID = "trace-1" }, Payload = new { order = "order-1" } });

        Assert.NotNull(received);
        Assert.Equal("orders.created", received!.Topic);
        Assert.Equal("plugin-a", received.SubscriberId);
    }

    [Fact]
    public async Task Local_delivery_nack_fails_synchronous_dispatch_for_retry()
    {
        var bridge = new LocalEventBridge();
        var runtime = new LocalEventFabricAdapter(bridge);
        await runtime.ConsumeAsync(new EventFabricSubscription("tenant-a", "plugin-a", ["orders.created"]), (_, _) => Task.FromResult(EventFabricDisposition.Nack));

        var exception = await Assert.ThrowsAsync<EventFabricAdapterException>(() => bridge.EmitAsync(new FrameworkEvent { Topic = "orders.created" }));

        Assert.Equal(EventFabricErrors.CodeNack, exception.Code);
    }
}
