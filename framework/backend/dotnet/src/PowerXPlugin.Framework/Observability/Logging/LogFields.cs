namespace PowerXPlugin.Framework.Observability.Logging;

public static class LogFields
{
    public const string TraceId = "trace_id";
    public const string TaskId = "task_id";
    public const string TenantUuid = "tenant_uuid";
    public const string TenantKey = "tenant_key";
    public const string SubscriberId = "subscriber_id";
    public const string Topic = "topic";
    public const string Status = "status";
    public const string Reason = "reason";
    public const string PluginId = "plugin_id";
    public const string Component = "component";
    public const string GatewayAuthScheme = "gateway_auth_scheme";
    public const string TokenSource = "outbound_token_source";
    public const string RequestId = "request_id";

    public const string StatusQueued = "queued";
    public const string StatusProcessing = "processing";
    public const string StatusSucceeded = "succeeded";
    public const string StatusFailed = "failed";
    public const string StatusSkipped = "skipped";

    public const string FallbackUnknown = "unknown";
    public const string ReasonMissingContext = "missing_context";

    public static readonly string[] RequiredFields = { TraceId, TaskId, TenantUuid, TenantKey, SubscriberId, Topic, Status };
}
