namespace PowerXPlugin.Framework.Runtime.EventFabric;

public enum EventFabricCompatibilityMode
{
    Strict,
    Backward,
    Any
}

/// <summary>
/// Stable, tenant-scoped Core Event Fabric subscription. Consumers never pass
/// an end-user bearer; the Framework applies the configured service identity.
/// </summary>
public sealed record EventFabricSubscription(
    string TenantUuid,
    string SubscriberId,
    IReadOnlyCollection<string> Topics,
    int BatchSize = 50,
    EventFabricCompatibilityMode CompatibilityMode = EventFabricCompatibilityMode.Backward,
    IReadOnlyCollection<string>? SupportedVersions = null);

public sealed record EventFabricDelivery(
    string DeliveryId,
    string EventId,
    string Topic,
    string TraceId,
    string Version,
    ReadOnlyMemory<byte> Payload,
    string PayloadFormat,
    IReadOnlyDictionary<string, string> Headers,
    int Attempt,
    int MaxAttempts,
    int RemainingAttempts,
    DateTimeOffset? AckDeadline,
    string SubscriberId);

public enum EventFabricDisposition
{
    Ack,
    Nack
}

public delegate Task<EventFabricDisposition> EventFabricHandler(EventFabricDelivery delivery, CancellationToken ct);

public interface IEventFabricSubscriber
{
    /// <summary>Runs until cancellation or the host stream terminates.</summary>
    Task ConsumeAsync(EventFabricSubscription subscription, EventFabricHandler handler, CancellationToken ct = default);
}

/// <summary>
/// Startup-selected Event Fabric subscription boundary. Publishing remains a
/// separate Core contract; this runtime deliberately models only the shared
/// delivery, acknowledgement and failure semantics available in both modes.
/// </summary>
public interface IEventRuntime : IEventFabricSubscriber
{
}

public static class EventFabricErrors
{
    public const string CodeUnavailable = "FRAMEWORK_EVENT_FABRIC_UNAVAILABLE";
    public const string CodeTenantMismatch = "FRAMEWORK_EVENT_FABRIC_TENANT_MISMATCH";
    public const string CodeInvalidSubscription = "FRAMEWORK_EVENT_FABRIC_INVALID_SUBSCRIPTION";
    public const string CodeUnauthorized = "FRAMEWORK_EVENT_FABRIC_UNAUTHORIZED";
    public const string CodeForbidden = "FRAMEWORK_EVENT_FABRIC_FORBIDDEN";
    public const string CodeUpstreamDependency = "FRAMEWORK_EVENT_FABRIC_UPSTREAM_DEPENDENCY";
    public const string CodeNack = "FRAMEWORK_EVENT_FABRIC_NACK";
}

public sealed class EventFabricAdapterException : InvalidOperationException
{
    public EventFabricAdapterException(string code, string? message = null, Exception? innerException = null)
        : base(message ?? code, innerException) => Code = code;

    public string Code { get; }
}
