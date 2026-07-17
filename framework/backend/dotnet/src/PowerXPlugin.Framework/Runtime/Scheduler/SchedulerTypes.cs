namespace PowerXPlugin.Framework.Runtime.Scheduler;

public static class SchedulerMode
{
    public const string Local = "local";
    public const string Host = "host";
    public const string Dual = "dual";
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

public record RetryPolicy(int MaxAttempts = 3, int BackoffSeconds = 60);

public record JobSpec(
    string Name,
    string OwnerType,
    string OwnerId,
    string ScheduleType,
    string ScheduleExpr,
    string? JobId = null,
    string? TenantUuid = null,
    string? Timezone = null,
    string? Topic = null,
    Dictionary<string, object?>? Payload = null,
    bool Paused = false,
    RetryPolicy? Retry = null
);

public record Job
{
    public string Uuid { get; set; } = Guid.NewGuid().ToString();
    public string JobId { get; set; } = "";
    public string Name { get; set; } = "";
    public string TenantUuid { get; set; } = "";
    public string OwnerType { get; set; } = "plugin";
    public string OwnerId { get; set; } = "";
    public string ScheduleType { get; set; } = "";
    public string ScheduleExpr { get; set; } = "";
    public string? Timezone { get; init; }
    public string Topic { get; init; } = "powerx.runtime.scheduler.triggered.v1";
    public string Status { get; set; } = JobStatus.Active;
    public DateTime NextRunAt { get; set; }
    public DateTime? LastRunAt { get; set; }
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
    public DateTime UpdatedAt { get; set; } = DateTime.UtcNow;
    public Dictionary<string, object?>? Payload { get; init; }
}
