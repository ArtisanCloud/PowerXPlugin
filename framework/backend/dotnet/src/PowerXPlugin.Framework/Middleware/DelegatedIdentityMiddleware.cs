using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Delegated;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.Middleware;

public sealed class DelegatedIdentityOptions
{
    public required string PluginId { get; init; }
    public string ApiPrefix { get; init; } = "/api/v1";
    public Func<HttpContext, string?>? TenantAssertion { get; init; }
    public Func<HttpContext, string?>? UserAssertion { get; init; }
}

public static class DelegatedIdentityMiddlewareExtensions
{
    private static readonly object IdentityKey = new();
    public static DelegatedIdentity? GetDelegatedIdentity(this HttpContext context)
        => context.Items.TryGetValue(IdentityKey, out var value) ? value as DelegatedIdentity : null;

    public static IApplicationBuilder UseDelegatedIdentity(this IApplicationBuilder app, DelegatedIdentityOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        return app.Use(async (context, next) => {
            try {
                var registry = app.ApplicationServices.GetService(typeof(IAMRegistry)) as IAMRegistry;
                var resolver = registry?.DelegatedIdentity ?? throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
                var query = new DelegatedIdentityQuery { ContextTicket = context.Request.Headers["X-PowerX-Identity-Context"].ToString() };
                query.Validate();
                var identity = await resolver.ResolveDelegatedIdentity(query, context.RequestAborted);
                identity.Validate(options.PluginId, DateTimeOffset.UtcNow);
                var path = context.Request.Path.Value ?? "/";
                var prefix = options.ApiPrefix.TrimEnd('/');
                if (prefix.Length > 0 && path.StartsWith(prefix + "/", StringComparison.Ordinal)) path = path[prefix.Length..];
                var transport = context.WebSockets.IsWebSocketRequest ? "ws_upgrade" : path.EndsWith("/runtime/ws-bus/grant", StringComparison.Ordinal) ? "ws_grant" : "http";
                if (identity.SourceMethod != context.Request.Method || identity.SourcePath != path || identity.Transport != transport)
                    throw new IAMAdapterException("IAM_CONTEXT_REQUEST_FORBIDDEN");
                if (options.TenantAssertion != null && options.TenantAssertion(context) != identity.TenantUuid)
                    throw new IAMAdapterException("IAM_CONTEXT_TENANT_FORBIDDEN");
                if (options.UserAssertion != null && options.UserAssertion(context) != identity.UserUuid)
                    throw new IAMAdapterException("IAM_CONTEXT_USER_FORBIDDEN");
                var requestId=context.Request.Headers["X-Request-ID"].ToString(); var traceId=context.Request.Headers["X-Trace-ID"].ToString();
                if ((requestId.Length>0 && requestId!=identity.SourceRequestId) || (traceId.Length>0 && traceId!=identity.SourceTraceId))
                    throw new IAMAdapterException("IAM_CONTEXT_REQUEST_FORBIDDEN");
                context.Items[IdentityKey] = identity;
                context.Items["TenantUUID"] = identity.TenantUuid;
            } catch (IAMAdapterException error) {
                var upstream = error as IdentityContextException;
                context.Response.StatusCode = upstream?.Status ?? IAMErrors.StatusCode(error.Code);
                await context.Response.WriteAsJsonAsync(new { error = new { code = error.Code, message = error.Code,
                    details = new { code = upstream?.OriginalCode, reason_code = upstream?.ReasonCode,
                        trace_id = upstream?.TraceId, request_id = upstream?.RequestId,
                        source_trace_id = upstream?.SourceTraceId, source_request_id = upstream?.SourceRequestId } },
                    trace_id = context.GetTraceId(), request_id = context.Request.Headers["X-Request-ID"].ToString()
                }, context.RequestAborted);
                return;
            }
            await next(context);
        });
    }
}
