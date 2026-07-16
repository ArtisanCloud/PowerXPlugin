namespace PowerXPlugin.Framework.Bootstrap;

public class PluginAppConfig
{
    public string Listen { get; set; } = ":8078";
    public string Env { get; set; } = "development";
    public bool Standalone { get; set; }
    public GatewayConfig Gateway { get; set; } = new();

    public static PluginAppConfig FromEnvironment()
    {
        return new PluginAppConfig
        {
            Listen = EnvOr("POWERX_LISTEN", ":8078"),
            Env = EnvOr("POWERX_ENV", "development"),
            Standalone = EnvOr("STANDALONE", "0") == "1",
            Gateway = new GatewayConfig
            {
                BaseURL = EnvOr("PX_GATEWAY_BASE_URL", ""),
                Token = EnvOr("PX_GATEWAY_TOKEN", ""),
                GRPCTarget = EnvOr("PX_GATEWAY_GRPC_TARGET", "")
            }
        };
    }

    private static string EnvOr(string key, string fallback)
    {
        var v = Environment.GetEnvironmentVariable(key);
        return string.IsNullOrWhiteSpace(v) ? fallback : v;
    }
}

public class GatewayConfig
{
    public string BaseURL { get; set; } = "";
    public string Token { get; set; } = "";
    public string GRPCTarget { get; set; } = "";
}
