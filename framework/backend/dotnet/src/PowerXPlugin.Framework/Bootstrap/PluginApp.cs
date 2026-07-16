using PowerXPlugin.Framework.Manifest;
using PowerXPlugin.Framework.IAM;

namespace PowerXPlugin.Framework.Bootstrap;

public class PluginApp
{
    public PluginAppConfig Config { get; }
    public Manifest.PluginManifest? Manifest { get; private set; }
    public IAMRegistry IAMRegistry { get; } = new();

    public PluginApp(PluginAppConfig config)
    {
        Config = config;
    }

    public static PluginApp FromEnvironment()
    {
        return new PluginApp(PluginAppConfig.FromEnvironment());
    }

    public void RegisterManifest(Manifest.PluginManifest manifest)
    {
        Manifest = manifest;
    }

    public void RegisterCapabilityInvoker(ICapabilityInvoker invoker)
    {
        CapabilityInvoker = invoker;
    }

    public ICapabilityInvoker? CapabilityInvoker { get; private set; }
}

public interface ICapabilityInvoker
{
    bool CanInvoke(string capabilityId);
    Task<object?> InvokeAsync(string capabilityId, object? parameters, CancellationToken ct = default);
}
