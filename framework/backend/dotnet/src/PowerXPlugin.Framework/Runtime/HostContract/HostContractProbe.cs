using Microsoft.Extensions.DependencyInjection;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.IAM;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Cache;
using PowerXPlugin.Framework.Runtime.TaskCenter;
using PowerXPlugin.Framework.Runtime.Knowledge;
using PowerXPlugin.Framework.Runtime.Media;
using PowerXPlugin.Framework.Runtime.Agent;
using PowerXPlugin.Framework.Runtime.AI;
using PowerXPlugin.Framework.Runtime.Capability;
using PowerXPlugin.Framework.Runtime.Integration;
using PowerXPlugin.Framework.Runtime.Skills;
using PowerXPlugin.Framework.Runtime.Notifications;

namespace PowerXPlugin.Framework.Runtime.HostContract;

/// <summary>Go Host Contract Lab status parity: resolves the selected adapter, never invokes business methods.</summary>
public sealed class HostContractProbe(IServiceProvider services)
{
    public static IReadOnlyList<string> Modules { get; } = ["cache", "taskcenter", "iam", "knowledge", "media", "agent", "ai", "capability_registry", "integration_gateway", "skills", "notifications", "plugin_runtime"];

    public HostContractStatus Status(string module, ProviderMode defaultMode, string traceId)
    {
        if (!Modules.Contains(module)) throw new ArgumentException("HOST_CONTRACT_UNSUPPORTED_MODULE");
        (bool available, ProviderMode mode) = module switch
        {
            "cache" => Binding<ICacheService>(defaultMode),
            "taskcenter" => Binding<ITaskCenterService>(defaultMode),
            "iam" => Binding<IAMRegistry>(defaultMode),
            "knowledge" => Binding<IKnowledgeService>(defaultMode),
            "media" => Binding<IMediaService>(defaultMode),
            "agent" => Binding<IAgentLifecycleService>(defaultMode),
            "ai" => Binding<IGenerativeService>(defaultMode),
            "capability_registry" => Binding<ICapabilityRegistry>(defaultMode),
            "integration_gateway" => Binding<IIntegrationGateway>(defaultMode),
            "skills" => Binding<ISkillInvoker>(defaultMode),
            "notifications" => Binding<INotificationPublisher>(defaultMode),
            "plugin_runtime" => Binding<PluginRuntime.IPluginRuntimeService>(defaultMode),
            _ => throw new ArgumentException("HOST_CONTRACT_UNSUPPORTED_MODULE")
        };
        return new(module, "status", mode.ToString().ToLowerInvariant(), module == "media" ? "com.corex.media.assets.read" : "", traceId,
            available ? "" : "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", new(available, false), DateTimeOffset.UtcNow);
    }

    private (bool, ProviderMode) Binding<T>(ProviderMode defaultMode) where T : class
    {
        var runtime = services.GetService<DualModeRuntime<T>>();
        return (runtime?.Service is not null, runtime?.Mode ?? defaultMode);
    }
}

public sealed record HostContractStatus(string Module, string Operation,
    [property: JsonPropertyName("provider_mode")] string ProviderMode,
    [property: JsonPropertyName("capability_id")] string CapabilityId,
    [property: JsonPropertyName("trace_id")] string TraceId,
    [property: JsonPropertyName("reason_code")] string ReasonCode, HostContractBinding Result,
    [property: JsonPropertyName("observed_at")] DateTimeOffset ObservedAt);
public sealed record HostContractBinding([property: JsonPropertyName("adapter_available")] bool AdapterAvailable,
    [property: JsonPropertyName("connectivity_verified")] bool ConnectivityVerified);
