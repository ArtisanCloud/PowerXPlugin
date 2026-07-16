namespace PowerXPlugin.Framework.EventBridge;

public record EventMeta
{
    public string? TenantUUID { get; init; }
    public string? RequestID { get; init; }
    public string? SourcePlugin { get; init; }
    public string? TraceID { get; init; }
    public DateTime OccurredAt { get; init; } = DateTime.UtcNow;
    public string PayloadVersion { get; init; } = "v1";
}

public class FrameworkEvent
{
    public string Topic { get; init; } = "";
    public EventMeta Meta { get; init; } = new();
    public object? Payload { get; init; }
}

public interface IEventEmitter
{
    Task EmitAsync(FrameworkEvent evt, CancellationToken ct = default);
}

public interface IEventDispatcher
{
    Task RegisterHandlerAsync(string topic, Func<FrameworkEvent, CancellationToken, Task> handler, CancellationToken ct = default);
    Task DispatchAsync(FrameworkEvent evt, CancellationToken ct = default);
}

public enum EventBridgeMode
{
    Local,
    TaskBus,
    Dual
}
