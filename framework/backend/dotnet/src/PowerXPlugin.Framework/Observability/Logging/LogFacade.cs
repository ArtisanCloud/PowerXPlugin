using Microsoft.Extensions.Logging;

namespace PowerXPlugin.Framework.Observability.Logging;

/// <summary>
/// Structured logging facade matching Go framework's logging.Facade.
/// Injects runtime fields (request_id, trace_id, tenant_uuid) from HttpContext.Items.
/// Register as Scoped service to auto-resolve per-request.
/// </summary>
public class LogFacade
{
    private readonly ILogger _logger;
    private readonly Dictionary<string, object?> _fields;

    public LogFacade(ILogger<LogFacade> logger, IHttpContextAccessor? httpAccessor = null)
    {
        _logger = logger;
        _fields = new Dictionary<string, object?>();

        if (httpAccessor?.HttpContext != null)
        {
            var ctx = httpAccessor.HttpContext;
            _fields[LogFields.RequestId] = ctx.Items["RequestId"] as string ?? ctx.TraceIdentifier;
            _fields[LogFields.TraceId] = ctx.Items["TraceId"] as string ?? _fields[LogFields.RequestId];
            _fields[LogFields.TenantUuid] = ctx.Items["TenantUUID"] as string ?? LogFields.FallbackUnknown;
        }

        _fields[LogFields.PluginId] = Environment.GetEnvironmentVariable("POWERX_PLUGIN_ID") ?? "unknown";
    }

    public LogFacade(ILogger logger, Dictionary<string, object?> baseFields)
    {
        _logger = logger;
        _fields = new Dictionary<string, object?>(baseFields);
    }

    public LogFacade With(Dictionary<string, object?> extra)
    {
        var merged = new Dictionary<string, object?>(_fields);
        foreach (var (k, v) in extra) merged[k] = v;
        return new LogFacade(_logger, merged);
    }

    public LogFacade With(string key, object? value) => With(new Dictionary<string, object?> { [key] = value });

    public void Info(string message, Dictionary<string, object?>? fields = null) => Emit(LogLevel.Information, message, fields);
    public void Warn(string message, Dictionary<string, object?>? fields = null) => Emit(LogLevel.Warning, message, fields);
    public void Error(string message, Dictionary<string, object?>? fields = null) => Emit(LogLevel.Error, message, fields);
    public void Debug(string message, Dictionary<string, object?>? fields = null) => Emit(LogLevel.Debug, message, fields);

    private void Emit(LogLevel level, string message, Dictionary<string, object?>? extra)
    {
        var state = new Dictionary<string, object?>(_fields);
        if (extra != null)
            foreach (var (k, v) in extra)
                state[k] = v;

        state["event_at"] = DateTime.UtcNow.ToString("O");
        ApplyFallback(state);
        _logger.Log(level, message, state.Select(kv => new KeyValuePair<string, object?>(kv.Key, kv.Value)));
    }

    private static void ApplyFallback(Dictionary<string, object?> f)
    {
        if (string.IsNullOrWhiteSpace(f.GetValueOrDefault(LogFields.TenantUuid) as string))
            f[LogFields.TenantUuid] = LogFields.FallbackUnknown;
        if (string.IsNullOrWhiteSpace(f.GetValueOrDefault(LogFields.TenantKey) as string))
            f[LogFields.TenantKey] = LogFields.FallbackUnknown;
    }

    public static LogFacade FromHttpContext(HttpContext ctx, ILogger logger) =>
        new(new Logger<LogFacade>(new LoggerFactory()), new HttpContextAccessor { HttpContext = ctx });
}
