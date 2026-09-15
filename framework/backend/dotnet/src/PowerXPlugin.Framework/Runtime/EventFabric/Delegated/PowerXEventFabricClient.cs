using Grpc.Core;
using Grpc.Net.Client;
using Powerx.EventFabric.V1;

namespace PowerXPlugin.Framework.Runtime.EventFabric.Delegated;

/// <summary>
/// Typed Core Event Fabric subscriber. This is the .NET equivalent of a host
/// provider: Core's generated protobuf contract stays inside Framework, while
/// a plugin receives only Framework delivery types and acknowledgement rules.
/// </summary>
public sealed class PowerXEventFabricClient : IEventFabricSubscriber, IDisposable
{
    private readonly PowerXEventFabricClientOptions _options;
    private readonly GrpcChannel _channel;
    private readonly EventSubscriberService.EventSubscriberServiceClient _subscriber;
    private readonly EventDeliveryService.EventDeliveryServiceClient _delivery;

    public PowerXEventFabricClient(PowerXEventFabricClientOptions options)
        : this(options, null)
    {
    }

    internal PowerXEventFabricClient(PowerXEventFabricClientOptions options, GrpcChannel? channel)
    {
        _options = options ?? throw new ArgumentNullException(nameof(options));
        EnsureConfigured();
        _channel = channel ?? GrpcChannel.ForAddress(NormalizeEndpoint(_options.Endpoint));
        _subscriber = new EventSubscriberService.EventSubscriberServiceClient(_channel);
        _delivery = new EventDeliveryService.EventDeliveryServiceClient(_channel);
    }

    public async Task ConsumeAsync(EventFabricSubscription subscription, EventFabricHandler handler, CancellationToken ct = default)
    {
        ValidateSubscription(subscription);
        ArgumentNullException.ThrowIfNull(handler);

        var request = new SubscribeRequest
        {
            TenantUuid = TrustedTenant(),
            SubscriberId = subscription.SubscriberId.Trim(),
            BatchSize = Math.Clamp(subscription.BatchSize, 1, 200),
            CompatibilityMode = ToProto(subscription.CompatibilityMode)
        };
        request.Topics.Add(subscription.Topics.Where(topic => !string.IsNullOrWhiteSpace(topic)).Select(topic => topic.Trim()));
        if (subscription.SupportedVersions != null)
            request.SupportedVersions.Add(subscription.SupportedVersions.Where(version => !string.IsNullOrWhiteSpace(version)).Select(version => version.Trim()));

        try
        {
            using var call = _subscriber.Subscribe(request, Headers(), cancellationToken: ct);
            while (await call.ResponseStream.MoveNext(ct))
            {
                var message = call.ResponseStream.Current;
                var mapped = Map(message);
                EventFabricDisposition disposition;
                string? failure = null;
                try
                {
                    disposition = await handler(mapped, ct);
                }
                catch (Exception exception) when (!ct.IsCancellationRequested)
                {
                    disposition = EventFabricDisposition.Nack;
                    failure = TrimReason(exception.Message);
                }

                if (disposition == EventFabricDisposition.Ack)
                    await _delivery.AckDeliveryAsync(new AckDeliveryRequest { DeliveryId = mapped.DeliveryId, SubscriberId = mapped.SubscriberId }, Headers(), cancellationToken: ct);
                else
                    await _delivery.NackDeliveryAsync(new NackDeliveryRequest { DeliveryId = mapped.DeliveryId, SubscriberId = mapped.SubscriberId, Reason = failure ?? "consumer rejected delivery" }, Headers(), cancellationToken: ct);
            }
        }
        catch (RpcException exception) when (!ct.IsCancellationRequested)
        {
            throw new EventFabricAdapterException(MapStatus(exception.StatusCode), exception.Status.Detail, exception);
        }
    }

    public void Dispose() => _channel.Dispose();

    private Metadata Headers() => new()
    {
        { "tenant-uuid", TrustedTenant() },
        { "authorization", $"{NormalizeAuthScheme(_options.AuthScheme)} {_options.Credential.Trim()}" },
        { "user-agent", string.IsNullOrWhiteSpace(_options.UserAgent) ? "powerx-plugin-framework-dotnet" : _options.UserAgent.Trim() }
    };

    private void ValidateSubscription(EventFabricSubscription subscription)
    {
        if (subscription == null || !string.Equals(subscription.TenantUuid?.Trim(), TrustedTenant(), StringComparison.Ordinal))
            throw new EventFabricAdapterException(EventFabricErrors.CodeTenantMismatch);
        if (string.IsNullOrWhiteSpace(subscription.SubscriberId) || subscription.Topics == null || !subscription.Topics.Any(topic => !string.IsNullOrWhiteSpace(topic)))
            throw new EventFabricAdapterException(EventFabricErrors.CodeInvalidSubscription);
    }

    private void EnsureConfigured()
    {
        if (string.IsNullOrWhiteSpace(_options.Endpoint) || string.IsNullOrWhiteSpace(_options.TenantUuid) || string.IsNullOrWhiteSpace(_options.Credential))
            throw new EventFabricAdapterException(EventFabricErrors.CodeUnavailable);
    }

    private string TrustedTenant() => _options.TenantUuid.Trim();
    private static EventFabricDelivery Map(DeliveryMessage message) => new(
        message.DeliveryId, message.EventId, message.Topic, message.TraceId, message.Version,
        message.Payload.Memory, message.PayloadFormat,
        new Dictionary<string, string>(message.Headers, StringComparer.Ordinal), message.Attempt,
        message.MaxAttempts, message.RemainingAttempts,
        message.AckDeadline is null ? null : new DateTimeOffset(message.AckDeadline.ToDateTime()), message.SubscriberId);
    private static VersionCompatibilityMode ToProto(EventFabricCompatibilityMode mode) => mode switch
    {
        EventFabricCompatibilityMode.Strict => VersionCompatibilityMode.Strict,
        EventFabricCompatibilityMode.Any => VersionCompatibilityMode.Any,
        _ => VersionCompatibilityMode.Backward
    };
    private static string NormalizeEndpoint(string value) => value.Trim().StartsWith("http://", StringComparison.OrdinalIgnoreCase) || value.Trim().StartsWith("https://", StringComparison.OrdinalIgnoreCase) ? value.Trim() : "http://" + value.Trim();
    private static string NormalizeAuthScheme(string value) => value.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase) ? "ApiKey" : "Bearer";
    private static string TrimReason(string value) => string.IsNullOrWhiteSpace(value) ? "consumer failed" : value.Trim()[..Math.Min(value.Trim().Length, 512)];
    private static string MapStatus(StatusCode status) => status switch
    {
        StatusCode.Unauthenticated => EventFabricErrors.CodeUnauthorized,
        StatusCode.PermissionDenied => EventFabricErrors.CodeForbidden,
        _ => EventFabricErrors.CodeUpstreamDependency
    };
}

public sealed class PowerXEventFabricClientOptions
{
    public string Endpoint { get; init; } = string.Empty;
    public string Credential { get; init; } = string.Empty;
    public string AuthScheme { get; init; } = "Bearer";
    public string TenantUuid { get; init; } = string.Empty;
    public string? UserAgent { get; init; }
}
