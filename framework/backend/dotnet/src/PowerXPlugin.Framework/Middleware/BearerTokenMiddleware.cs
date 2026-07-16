namespace PowerXPlugin.Framework.Middleware;

public static class BearerTokenMiddlewareExtensions
{
    public static IApplicationBuilder UseBearerToken(this IApplicationBuilder app)
    {
        return app.UseMiddleware<BearerTokenMiddleware>();
    }
}

public class BearerTokenMiddleware
{
    private readonly RequestDelegate _next;

    public BearerTokenMiddleware(RequestDelegate next) => _next = next;

    public async Task InvokeAsync(HttpContext ctx)
    {
        var auth = ctx.Request.Headers.Authorization.FirstOrDefault();
        if (!string.IsNullOrWhiteSpace(auth) && auth.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase))
        {
            ctx.Items["BearerToken"] = auth["Bearer ".Length..].Trim();
        }
        await _next(ctx);
    }
}

public static class BearerTokenContext
{
    public static string? GetBearerToken(this HttpContext ctx) =>
        ctx.Items["BearerToken"] as string;
}
