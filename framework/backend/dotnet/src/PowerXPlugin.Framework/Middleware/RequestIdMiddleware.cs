
namespace PowerXPlugin.Framework.Middleware;

public static class RequestIdMiddlewareExtensions
{
    public static IApplicationBuilder UseRequestId(this IApplicationBuilder app)
    {
        return app.UseMiddleware<RequestIdMiddleware>();
    }
}

public class RequestIdMiddleware
{
    private readonly RequestDelegate _next;

    public RequestIdMiddleware(RequestDelegate next) => _next = next;

    public async Task InvokeAsync(HttpContext ctx)
    {
        var requestId = ctx.Request.Headers["X-Request-ID"].FirstOrDefault()
                        ?? Guid.NewGuid().ToString("N");

        var traceId = ctx.Request.Headers["traceparent"].FirstOrDefault()
                      ?? ctx.Request.Headers["X-Trace-Id"].FirstOrDefault()
                      ?? requestId;

        ctx.Items["RequestId"] = requestId;
        ctx.Items["TraceId"] = traceId;

        ctx.Response.Headers["X-Request-ID"] = requestId;
        ctx.Response.Headers["X-Trace-Id"] = traceId;

        await _next(ctx);
    }
}

public static class RequestIdContext
{
    public static string? GetTraceId(this HttpContext ctx) =>
        ctx.Items["TraceId"] as string;

    public static string? GetRequestId(this HttpContext ctx) =>
        ctx.Items["RequestId"] as string;
}
