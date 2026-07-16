using System.IdentityModel.Tokens.Jwt;

namespace PowerXPlugin.Framework.Middleware;

public static class TenantContextMiddlewareExtensions
{
    public static IApplicationBuilder UseTenantContext(this IApplicationBuilder app)
    {
        return app.UseMiddleware<TenantContextMiddleware>();
    }
}

public class TenantContextMiddleware
{
    private readonly RequestDelegate _next;
    private const string DefaultTenantUuid = "00000000-0000-0000-0000-000000000001";

    public TenantContextMiddleware(RequestDelegate next) => _next = next;

    public async Task InvokeAsync(HttpContext ctx)
    {
        var tenantUuid = ResolveTenantUuid(ctx);
        ctx.Items["TenantUUID"] = tenantUuid;
        await _next(ctx);
    }

    private static string ResolveTenantUuid(HttpContext ctx)
    {
        // 1. Query param
        var q = ctx.Request.Query["tenant_uuid"].FirstOrDefault();
        if (!string.IsNullOrWhiteSpace(q)) return q;

        // 2. Header
        var h = ctx.Request.Headers["tenant_uuid"].FirstOrDefault();
        if (!string.IsNullOrWhiteSpace(h)) return h;

        // 3. JWT claims
        var auth = ctx.Request.Headers.Authorization.FirstOrDefault();
        if (!string.IsNullOrWhiteSpace(auth) && auth.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase))
        {
            var token = auth["Bearer ".Length..].Trim();
            try
            {
                var handler = new JwtSecurityTokenHandler();
                var jwt = handler.ReadJwtToken(token);
                var tid = jwt.Claims.FirstOrDefault(c => c.Type is "tid" or "tenant_uuid")?.Value;
                if (!string.IsNullOrWhiteSpace(tid)) return tid;
            }
            catch { }
        }

        return DefaultTenantUuid;
    }
}

public static class TenantContextExtensions
{
    public static string GetTenantUUID(this HttpContext ctx) =>
        ctx.Items["TenantUUID"] as string ?? "00000000-0000-0000-0000-000000000001";
}
