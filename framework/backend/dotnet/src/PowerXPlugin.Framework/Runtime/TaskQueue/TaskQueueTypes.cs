using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.TaskQueue;

public enum TaskQueueAdapterMode { Local, Delegated }

public sealed record TaskQueueMessage(
    [property: JsonPropertyName("id")] string Id,
    [property: JsonPropertyName("tenant_key")] string TenantKey,
    [property: JsonPropertyName("subscriber_id")] string SubscriberId,
    [property: JsonPropertyName("topic")] string Topic,
    [property: JsonPropertyName("payload")] byte[] Payload,
    [property: JsonPropertyName("headers")] Dictionary<string, string>? Headers = null,
    [property: JsonPropertyName("attempt")] int Attempt = 0,
    [property: JsonPropertyName("trace_id")] string? TraceId = null,
    [property: JsonPropertyName("visible_at")] DateTime? VisibleAt = null,
    [property: JsonPropertyName("metadata")] Dictionary<string, string>? Metadata = null,
    [property: JsonPropertyName("idempotency_key")] string? IdempotencyKey = null);

public sealed record TaskQueueDequeueRequest(string TenantKey, string SubscriberId, int MaxItems = 50, TimeSpan WaitTimeout = default);
public sealed record TaskQueueAckRequest(string TenantKey, string SubscriberId, string MessageId, Dictionary<string, string>? Metadata = null);
public sealed record TaskQueueNackRequest(string TenantKey, string SubscriberId, string MessageId, string? Reason = null, DateTime? RetryAt = null, Dictionary<string, string>? Metadata = null);
public sealed record TaskQueueRetryRequest(TaskQueueMessage Message, DateTime? RetryAt = null, string? Reason = null);

public sealed class TaskQueueAdapterException : Exception
{
    public string Code { get; }
    public int? UpstreamStatusCode { get; }
    public TaskQueueAdapterException(string code, int? upstreamStatusCode = null, Exception? innerException = null) : base(code, innerException)
    {
        Code = code;
        UpstreamStatusCode = upstreamStatusCode;
    }
}

public static class TaskQueueErrors
{
    public const string CodeInvalidMode = "TASK_QUEUE_ADAPTER_MODE_INVALID";
    public const string CodeInvalidRequest = "TASK_QUEUE_INVALID_REQUEST";
    public const string CodeUnavailable = "TASK_QUEUE_ADAPTER_UNAVAILABLE";
    public const string CodeTenantMismatch = "TASK_QUEUE_TENANT_MISMATCH";
    public const string CodeUnauthorized = "TASK_QUEUE_UNAUTHORIZED";
    public const string CodeForbidden = "TASK_QUEUE_FORBIDDEN";
    public const string CodeUpstreamDependency = "TASK_QUEUE_UPSTREAM_DEPENDENCY";
}
