using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;

namespace PowerXPlugin.Framework.Middleware;

public class AuthGuardOptions
{
    public string Resource { get; set; } = "";
    public string Action { get; set; } = "";
    public Func<HttpContext, string?>? TenantResolver { get; set; }
    public Func<HttpContext, string?>? UserResolver { get; set; }
    public Func<HttpContext, string?>? TraceResolver { get; set; }
}

public static class AuthGuardMiddlewareExtensions
{
    public static IApplicationBuilder UseAuthGuard(this IApplicationBuilder app, AuthGuardOptions? opts = null)
    {
        return app.Use(async (ctx, next) =>
        {
            var options = opts ?? new AuthGuardOptions();
            var reg = app.ApplicationServices.GetService(typeof(IAMRegistry)) as IAMRegistry;
            if (reg == null || !reg.IsBound)
            {
                await next(ctx);
                return;
            }

            var authz = reg.Authz!;
            var tenantUuid = options.TenantResolver?.Invoke(ctx)
                ?? ctx.GetTenantUUID();
            var userId = options.UserResolver?.Invoke(ctx)
                ?? "anonymous";
            var traceId = options.TraceResolver?.Invoke(ctx)
                ?? ctx.GetTraceId()
                ?? Guid.NewGuid().ToString("N");

            if (string.IsNullOrWhiteSpace(options.Resource) || string.IsNullOrWhiteSpace(options.Action))
            {
                await next(ctx);
                return;
            }

            var req = new AuthorizationRequest
            {
                TenantUUID = tenantUuid,
                UserID = userId,
                Resource = options.Resource,
                Action = options.Action,
                TraceID = traceId
            };

            var decision = await authz.AuthorizeAsync(req);
            if (decision is not { Allowed: true })
            {
                ctx.Response.StatusCode = StatusCodes.Status403Forbidden;
                await ctx.Response.WriteAsJsonAsync(new
                {
                    code = IAMErrors.CodeForbidden,
                    message = "Access denied",
                    resource = options.Resource,
                    action = options.Action
                });
                return;
            }

            await next(ctx);
        });
    }
}
