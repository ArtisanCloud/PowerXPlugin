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

        var exception = await Assert.ThrowsAsync<EventFabricAdapterException>(() => bridge.EmitAsync(new FrameworkEvent
        {
            Topic = "orders.created", Meta = new EventMeta { TenantUUID = "tenant-a" }
        }));

        Assert.Equal(EventFabricErrors.CodeNack, exception.Code);
    }

    [Fact]
    public async Task Shared_topic_routes_only_to_the_matching_tenant_subscription()
    {
        var bridge = new LocalEventBridge();
        var runtime = new LocalEventFabricAdapter(bridge);
        var received = new List<string>();
        foreach (var tenant in new[] { "tenant-a", "tenant-b" })
        {
            await runtime.ConsumeAsync(new EventFabricSubscription(tenant, tenant, ["orders.created"]), (delivery, _) =>
            {
                received.Add(delivery.SubscriberId);
                return Task.FromResult(EventFabricDisposition.Ack);
            });
        }

        await bridge.EmitAsync(new FrameworkEvent { Topic = "orders.created", Meta = new EventMeta { TenantUUID = "tenant-a" } });
        Assert.Equal(["tenant-a"], received);
        await bridge.EmitAsync(new FrameworkEvent { Topic = "orders.created", Meta = new EventMeta { TenantUUID = "tenant-b" } });
        Assert.Equal(["tenant-a", "tenant-b"], received);
        await bridge.EmitAsync(new FrameworkEvent { Topic = "orders.created" });
        Assert.Equal(["tenant-a", "tenant-b"], received);
    }

    [Fact]
    public async Task Other_tenant_payload_is_not_serialized_or_passed_to_the_handler()
    {
        var bridge = new LocalEventBridge();
        var runtime = new LocalEventFabricAdapter(bridge);
        var calls = 0;
        await runtime.ConsumeAsync(new EventFabricSubscription("tenant-a", "plugin-a", ["orders.created"]), (_, _) =>
        {
            calls++;
            return Task.FromResult(EventFabricDisposition.Ack);
        });
        var circular = new Dictionary<string, object>();
        circular["self"] = circular;

        await bridge.EmitAsync(new FrameworkEvent
        {
            Topic = "orders.created", Meta = new EventMeta { TenantUUID = "tenant-b" }, Payload = circular
        });

        Assert.Equal(0, calls);
    }
}
