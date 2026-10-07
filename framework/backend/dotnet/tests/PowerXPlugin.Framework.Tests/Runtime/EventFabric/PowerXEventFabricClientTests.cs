using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using GrpcMetadata = Grpc.Core.Metadata;
using PowerXPlugin.Framework.Runtime.EventFabric;
using PowerXPlugin.Framework.Runtime.EventFabric.Delegated;
using Powerx.EventFabric.V1;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.EventFabric;

public sealed class PowerXEventFabricClientTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    private const string Other = "00000000-0000-4000-8000-000000000002";

    [Fact]
    public async Task Subscribe_binds_service_identity_and_acknowledges_or_nacks_each_delivery()
    {
        var subscriber = new Subscriber([Delivery("one"), Delivery("two")]);
        var delivery = new DeliveryClient();
        using var client = Client(subscriber, delivery);
        var handled = new List<string>();

        await client.ConsumeAsync(new(Tenant, "crm", ["powerx.runtime.scheduler.triggered.v1"]), (message, _) =>
        {
            handled.Add(message.EventId);
            Assert.Equal("application/json", message.PayloadFormat);
            return Task.FromResult(message.EventId == "one" ? EventFabricDisposition.Ack : EventFabricDisposition.Nack);
        });

        Assert.Equal(["one", "two"], handled);
        Assert.Equal(Tenant, subscriber.Request!.TenantUuid);
        Assert.Equal("crm", subscriber.Request.SubscriberId);
        Assert.Equal("Bearer service-token", subscriber.Headers!.GetValue("authorization"));
        Assert.Equal(Tenant, subscriber.Headers.GetValue("tenant-uuid"));
        Assert.Equal("one", Assert.Single(delivery.Acks).DeliveryId);
        Assert.Equal("two", Assert.Single(delivery.Nacks).DeliveryId);
        Assert.Equal("consumer rejected delivery", delivery.Nacks[0].Reason);
        Assert.All(delivery.Headers, headers =>
        {
            Assert.Equal("Bearer service-token", headers.GetValue("authorization"));
            Assert.Equal(Tenant, headers.GetValue("tenant-uuid"));
        });
    }

    [Fact]
    public async Task Handler_exception_nacks_without_falsely_acknowledging()
    {
        var subscriber = new Subscriber([Delivery("one")]);
        var delivery = new DeliveryClient();
        using var client = Client(subscriber, delivery);

        await client.ConsumeAsync(new(Tenant, "crm", ["topic"]), (_, _) => throw new InvalidOperationException("consumer failure"));

        Assert.Empty(delivery.Acks);
        Assert.Equal("consumer failure", Assert.Single(delivery.Nacks).Reason);
    }

    [Fact]
    public async Task Tenant_mismatch_never_opens_a_stream()
    {
        var subscriber = new Subscriber([]);
        using var client = Client(subscriber, new DeliveryClient());

        var error = await Assert.ThrowsAsync<EventFabricAdapterException>(() =>
            client.ConsumeAsync(new(Other, "crm", ["topic"]), (_, _) => Task.FromResult(EventFabricDisposition.Ack)));

        Assert.Equal(EventFabricErrors.CodeTenantMismatch, error.Code);
        Assert.Null(subscriber.Request);
    }

    [Theory]
    [InlineData(StatusCode.Unauthenticated, EventFabricErrors.CodeUnauthorized)]
    [InlineData(StatusCode.PermissionDenied, EventFabricErrors.CodeForbidden)]
    [InlineData(StatusCode.Unavailable, EventFabricErrors.CodeUpstreamDependency)]
    public async Task Grpc_rejection_preserves_error_category(StatusCode status, string code)
    {
        var subscriber = new Subscriber([], status);
        using var client = Client(subscriber, new DeliveryClient());

        var error = await Assert.ThrowsAsync<EventFabricAdapterException>(() =>
            client.ConsumeAsync(new(Tenant, "crm", ["topic"]), (_, _) => Task.FromResult(EventFabricDisposition.Ack)));

        Assert.Equal(code, error.Code);
    }

    private static PowerXEventFabricClient Client(Subscriber subscriber, DeliveryClient delivery) => new(new()
    {
        Endpoint = "http://core:9001", TenantUuid = Tenant, Credential = "service-token", AuthScheme = "Bearer"
    }, subscriber, delivery);

    private static DeliveryMessage Delivery(string eventId) => new()
    {
        DeliveryId = eventId, EventId = eventId, Topic = "powerx.runtime.scheduler.triggered.v1",
        TraceId = "trace", Version = "v1", PayloadFormat = "application/json",
        Payload = ByteString.CopyFromUtf8("{}"), SubscriberId = "crm"
    };

    private sealed class Subscriber(IReadOnlyList<DeliveryMessage> messages, StatusCode? reject = null) : EventSubscriberService.EventSubscriberServiceClient
    {
        public SubscribeRequest? Request { get; private set; }
        public GrpcMetadata? Headers { get; private set; }
        public override AsyncServerStreamingCall<DeliveryMessage> Subscribe(SubscribeRequest request, GrpcMetadata headers = null!,
            DateTime? deadline = null, CancellationToken cancellationToken = default)
        {
            Request = request;
            Headers = headers;
            return new AsyncServerStreamingCall<DeliveryMessage>(new Stream(messages, reject), Task.FromResult(new GrpcMetadata()),
                () => Status.DefaultSuccess, () => new GrpcMetadata(), () => { });
        }
    }

    private sealed class Stream(IReadOnlyList<DeliveryMessage> messages, StatusCode? reject) : IAsyncStreamReader<DeliveryMessage>
    {
        private int _next;
        public DeliveryMessage Current { get; private set; } = null!;
        public Task<bool> MoveNext(CancellationToken cancellationToken)
        {
            cancellationToken.ThrowIfCancellationRequested();
            if (reject.HasValue) throw new RpcException(new Status(reject.Value, "rejected"));
            if (_next >= messages.Count) return Task.FromResult(false);
            Current = messages[_next++];
            return Task.FromResult(true);
        }
    }

    private sealed class DeliveryClient : EventDeliveryService.EventDeliveryServiceClient
    {
        public List<AckDeliveryRequest> Acks { get; } = [];
        public List<NackDeliveryRequest> Nacks { get; } = [];
        public List<GrpcMetadata> Headers { get; } = [];
        public override AsyncUnaryCall<Empty> AckDeliveryAsync(AckDeliveryRequest request, GrpcMetadata headers = null!,
            DateTime? deadline = null, CancellationToken cancellationToken = default)
        {
            Acks.Add(request);
            Headers.Add(headers);
            return Unary(new Empty());
        }
        public override AsyncUnaryCall<NackDeliveryResponse> NackDeliveryAsync(NackDeliveryRequest request, GrpcMetadata headers = null!,
            DateTime? deadline = null, CancellationToken cancellationToken = default)
        {
            Nacks.Add(request);
            Headers.Add(headers);
            return Unary(new NackDeliveryResponse());
        }
        private static AsyncUnaryCall<T> Unary<T>(T response) => new(Task.FromResult(response), Task.FromResult(new GrpcMetadata()),
            () => Status.DefaultSuccess, () => new GrpcMetadata(), () => { });
    }
}
