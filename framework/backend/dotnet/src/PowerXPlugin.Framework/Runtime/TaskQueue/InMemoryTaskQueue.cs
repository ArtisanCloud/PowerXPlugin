using System.Collections.Concurrent;

namespace PowerXPlugin.Framework.Runtime.TaskQueue;

/// <summary>Local-only queue for standalone development; never selected as a delegated fallback.</summary>
public sealed class InMemoryTaskQueue : ITaskQueue
{
    private readonly ConcurrentDictionary<string, ConcurrentQueue<TaskQueueMessage>> _queues = new(StringComparer.Ordinal);

    public Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(message.TenantKey) || string.IsNullOrWhiteSpace(message.SubscriberId) || string.IsNullOrWhiteSpace(message.Id))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
        _queues.GetOrAdd(Key(message.TenantKey, message.SubscriberId), _ => new()).Enqueue(message);
        return Task.CompletedTask;
    }

    public Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default)
    {
        if (string.IsNullOrWhiteSpace(request.TenantKey) || string.IsNullOrWhiteSpace(request.SubscriberId))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
        var result = new List<TaskQueueMessage>();
        if (!_queues.TryGetValue(Key(request.TenantKey, request.SubscriberId), out var queue)) return Task.FromResult<IReadOnlyList<TaskQueueMessage>>(result);
        var limit = Math.Clamp(request.MaxItems, 1, 200);
        var deferred = new List<TaskQueueMessage>();
        while (result.Count < limit && queue.TryDequeue(out var message))
        {
            if (message.VisibleAt is { } visibleAt && visibleAt > DateTime.UtcNow) deferred.Add(message);
            else result.Add(message);
        }
        foreach (var message in deferred) queue.Enqueue(message);
        return Task.FromResult<IReadOnlyList<TaskQueueMessage>>(result);
    }

    public Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default) => ValidateAck(request.TenantKey, request.SubscriberId, request.MessageId);
    public async Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default)
    {
        await ValidateAck(request.TenantKey, request.SubscriberId, request.MessageId);
        // Local queue has no inflight store; callers use RetryAsync with the original message.
    }
    public Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default) => EnqueueAsync(request.Message with { VisibleAt = request.RetryAt }, ct);

    private static Task ValidateAck(string tenantKey, string subscriberId, string messageId)
    {
        if (string.IsNullOrWhiteSpace(tenantKey) || string.IsNullOrWhiteSpace(subscriberId) || string.IsNullOrWhiteSpace(messageId))
            throw new TaskQueueAdapterException(TaskQueueErrors.CodeInvalidRequest);
        return Task.CompletedTask;
    }
    private static string Key(string tenantKey, string subscriberId) => tenantKey.Trim() + ":" + subscriberId.Trim();
}
