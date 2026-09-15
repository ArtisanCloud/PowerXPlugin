using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Scheduler;

public static class SchedulerMode
{
    public const string Local = "local";
    public const string Host = "host";
    public const string Dual = "dual";
}

/// <summary>Startup-selected scheduler transport. It is never request-selectable.</summary>
public enum SchedulerAdapterMode
{
    Local,
    Delegated
}

public static class ScheduleType
{
    public const string Once = "once";
    public const string Interval = "interval";
    public const string Cron = "cron";
}

public static class JobStatus
{
    public const string Active = "active";
    public const string Paused = "paused";
    public const string Completed = "completed";
}

public record RetryPolicy(
    [property: JsonPropertyName("max_attempts")] int MaxAttempts = 3,
    [property: JsonPropertyName("backoff_seconds")] int BackoffSeconds = 60);

public record JobSpec(
    [property: JsonPropertyName("name")] string Name,
    [property: JsonPropertyName("owner_type")] string OwnerType,
    [property: JsonPropertyName("owner_id")] string OwnerId,
    [property: JsonPropertyName("schedule_type")] string ScheduleType,
    [property: JsonPropertyName("schedule_expr")] string ScheduleExpr,
    [property: JsonPropertyName("job_id")] string? JobId = null,
    [property: JsonPropertyName("tenant_uuid")] string? TenantUuid = null,
    [property: JsonPropertyName("timezone")] string? Timezone = null,
    [property: JsonPropertyName("topic")] string? Topic = null,
    [property: JsonPropertyName("payload")] Dictionary<string, object?>? Payload = null,
    [property: JsonPropertyName("paused")] bool Paused = false,
    [property: JsonPropertyName("retry_policy")] RetryPolicy? Retry = null
);

public record Job
{
    [JsonPropertyName("uuid")]
    public string Uuid { get; set; } = Guid.NewGuid().ToString();
    [JsonPropertyName("job_id")]
    public string JobId { get; set; } = "";
    [JsonPropertyName("name")]
    public string Name { get; set; } = "";
    [JsonPropertyName("tenant_uuid")]
    public string TenantUuid { get; set; } = "";
    [JsonPropertyName("owner_type")]
    public string OwnerType { get; set; } = "plugin";
    [JsonPropertyName("owner_id")]
    public string OwnerId { get; set; } = "";
    [JsonPropertyName("schedule_type")]
    public string ScheduleType { get; set; } = "";
    [JsonPropertyName("schedule_expr")]
    public string ScheduleExpr { get; set; } = "";
    [JsonPropertyName("timezone")]
    public string? Timezone { get; init; }
    [JsonPropertyName("topic")]
    public string Topic { get; init; } = "powerx.runtime.scheduler.triggered.v1";
    [JsonPropertyName("status")]
    public string Status { get; set; } = JobStatus.Active;
    [JsonPropertyName("next_run_at")]
    public DateTime NextRunAt { get; set; }
    [JsonPropertyName("last_run_at")]
    public DateTime? LastRunAt { get; set; }
    [JsonPropertyName("created_at")]
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
    [JsonPropertyName("updated_at")]
    public DateTime UpdatedAt { get; set; } = DateTime.UtcNow;
    [JsonPropertyName("payload")]
    public Dictionary<string, object?>? Payload { get; init; }
}

/// <summary>
/// Stable payload emitted when a Scheduler job fires. Consumers must verify
/// owner and business action, then persist the idempotency key before/with
/// business state mutation. Host-triggered messages use this same envelope.
/// </summary>
public sealed record SchedulerTriggeredPayload(
    [property: JsonPropertyName("job_id")] string JobId,
    [property: JsonPropertyName("job_name")] string JobName,
    [property: JsonPropertyName("owner_type")] string OwnerType,
    [property: JsonPropertyName("owner_id")] string OwnerId,
    [property: JsonPropertyName("tenant_uuid")] string TenantUuid,
    [property: JsonPropertyName("trigger_source")] string TriggerSource,
    [property: JsonPropertyName("scheduled_at")] DateTime ScheduledAt,
    [property: JsonPropertyName("fired_at")] DateTime FiredAt,
    [property: JsonPropertyName("trace_id")] string TraceId,
    [property: JsonPropertyName("idempotency_key")] string IdempotencyKey,
    [property: JsonPropertyName("business_action")] string? BusinessAction,
    [property: JsonPropertyName("payload")] Dictionary<string, object?> Payload);

public sealed class SchedulerAdapterException : Exception
{
    public string Code { get; }
    public int? UpstreamStatusCode { get; }

    public SchedulerAdapterException(string code, int? upstreamStatusCode = null, Exception? innerException = null)
        : base(code, innerException)
    {
        Code = code;
        UpstreamStatusCode = upstreamStatusCode;
    }
}

public static class SchedulerErrors
{
    public const string CodeInvalidMode = "SCHEDULER_ADAPTER_MODE_INVALID";
    public const string CodeInvalidJob = "SCHEDULER_INVALID_JOB_SPEC";
    public const string CodeUnavailable = "SCHEDULER_ADAPTER_UNAVAILABLE";
    public const string CodeUnauthorized = "SCHEDULER_UNAUTHORIZED";
    public const string CodeForbidden = "SCHEDULER_FORBIDDEN";
    public const string CodeUpstreamDependency = "SCHEDULER_UPSTREAM_DEPENDENCY";
}
