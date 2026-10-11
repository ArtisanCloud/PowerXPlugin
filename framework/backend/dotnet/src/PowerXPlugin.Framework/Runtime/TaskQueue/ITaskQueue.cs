namespace PowerXPlugin.Framework.Runtime.TaskQueue;

public interface ITaskQueue
{
    Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default);
    Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default);
    Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default);
    Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default);
    Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default);
}

/// <summary>
/// Plugin-owned local queue persistence. Production implementations must keep
/// delivery/inflight/ack state durable; TaskCenter is not a queue substitute.
/// </summary>
public interface ILocalTaskQueueStore : ITaskQueue { }

public sealed class LocalTaskQueueAdapter(ILocalTaskQueueStore store) : ITaskQueue
{
    private readonly ILocalTaskQueueStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default) { Validate(message); return _store.EnqueueAsync(message, ct); }
    public Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default) { Validate(request.TenantKey, request.SubscriberId); return _store.DequeueAsync(request with { MaxItems = Math.Clamp(request.MaxItems, 1, 200) }, ct); }
    public Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default) { Validate(request.TenantKey, request.SubscriberId, request.MessageId); return _store.AckAsync(request, ct); }
    public Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default) { Validate(request.TenantKey, request.SubscriberId, request.MessageId); return _store.NackAsync(request, ct); }
    public Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default) { Validate(request.Message); return _store.RetryAsync(request, ct); }
    private static void Validate(TaskQueueMessage m) => Validate(m.TenantKey, m.SubscriberId, m.Id, m.Topic);
    private static void Validate(string tenant, string subscriber, string? id = null, string? topic = null) { if (string.IsNullOrWhiteSpace(tenant) || string.IsNullOrWhiteSpace(subscriber) || id is not null && string.IsNullOrWhiteSpace(id) || topic is not null && string.IsNullOrWhiteSpace(topic)) throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest); }
}
