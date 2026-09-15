namespace PowerXPlugin.Framework.Runtime.TaskQueue;

public interface ITaskQueue
{
    Task EnqueueAsync(TaskQueueMessage message, CancellationToken ct = default);
    Task<IReadOnlyList<TaskQueueMessage>> DequeueAsync(TaskQueueDequeueRequest request, CancellationToken ct = default);
    Task AckAsync(TaskQueueAckRequest request, CancellationToken ct = default);
    Task NackAsync(TaskQueueNackRequest request, CancellationToken ct = default);
    Task RetryAsync(TaskQueueRetryRequest request, CancellationToken ct = default);
}
