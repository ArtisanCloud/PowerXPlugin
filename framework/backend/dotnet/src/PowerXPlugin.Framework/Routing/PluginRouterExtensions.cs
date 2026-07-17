using PowerXPlugin.Framework.Bootstrap;

namespace PowerXPlugin.Framework.Routing;

public static class PluginRouterExtensions
{
    public const string HealthzPath = "/healthz";
    public const string ApiPrefix = "/api/v1";
    public const string CapabilityInvokePath = "/api/v1/integration/capabilities/invoke";

    public static WebApplication UseFrameworkRoutes(this WebApplication app)
    {
        app.MapHealthChecks(HealthzPath);

        app.MapPost(CapabilityInvokePath, async (HttpContext ctx, PluginApp pluginApp) =>
        {
            var invoker = pluginApp.CapabilityInvoker;
            if (invoker == null)
            {
                ctx.Response.StatusCode = 501;
                await ctx.Response.WriteAsJsonAsync(new
                {
                    code = "NOT_IMPLEMENTED",
                    message = "No capability invoker registered"
                });
                return;
            }

            using var reader = new StreamReader(ctx.Request.Body);
            var body = await reader.ReadToEndAsync();
            var result = await invoker.InvokeAsync("capability_invoke", body);
            await ctx.Response.WriteAsJsonAsync(result ?? new { status = "ok" });
        });

        return app;
    }

    public static RouteGroupBuilder MapPluginRoutes(this WebApplication app)
    {
        return app.MapGroup(ApiPrefix);
    }
}
