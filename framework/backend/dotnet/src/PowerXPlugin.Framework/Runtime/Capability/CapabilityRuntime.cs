using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Capability;

public sealed record CapabilityProtocol(string Channel, string? Endpoint = null, string? Method = null, string? Rpc = null, string? SchemaRef = null);
public sealed record CapabilityDescriptor(string CapabilityId, string PluginId, string PluginVersion, string Title, string Source, IReadOnlyCollection<CapabilityProtocol> Protocols, string Status, string? Description = null);
public sealed record CapabilityListQuery(int Page = 1, int PageSize = 50, string? PluginId = null, string? Intent = null, string? ToolScope = null, string? Protocol = null, string? Source = null);
public sealed record CapabilityResolveQuery(string Method, string Endpoint, string? Source = null);
public sealed record CapabilityResolution(string CapabilityId, string PluginId, string Source, string Protocol, string Method, string PatternEndpoint);
public sealed record CapabilityInvokeRequest(string? CapabilityId, string? Intent, string? ToolScope, string? PreferredProtocol, string? IdempotencyKey, string? TraceId, IReadOnlyCollection<string>? ToolGrantIds, IReadOnlyDictionary<string, object?>? Payload, IReadOnlyDictionary<string, object?>? Context);
public sealed record CapabilityInvokeResult(string TraceId, string Status, string ProtocolUsed, bool FallbackUsed, IReadOnlyDictionary<string, object?>? Payload, IReadOnlyDictionary<string, object?>? Result);
public sealed record CapabilityInvocation(string TraceId, string TenantUuid, string CapabilityId, string ProtocolUsed, bool FallbackUsed, string Status, int LatencyMs);

public interface ICapabilityRegistry : ICapabilityGrantRegistry
{
    Task<IReadOnlyList<CapabilityDescriptor>> ListAsync(CapabilityListQuery query, CancellationToken ct = default);
    Task<CapabilityResolution> ResolveAsync(CapabilityResolveQuery query, CancellationToken ct = default);
    Task<CapabilityInvokeResult> InvokeAsync(CapabilityInvokeRequest request, CancellationToken ct = default);
    Task<CapabilityInvocation> GetInvocationAsync(string traceId, CancellationToken ct = default);
}

/// <summary>Plugin-owned local data/operations; Framework never supplies fake capability persistence.</summary>
public interface ILocalCapabilityStore : ICapabilityRegistry { }
public sealed class LocalCapabilityRegistry(ILocalCapabilityStore store) : ICapabilityRegistry
{
    private readonly ILocalCapabilityStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public Task<IReadOnlyList<CapabilityDescriptor>> ListAsync(CapabilityListQuery query, CancellationToken ct = default) => _store.ListAsync(query, ct);
    public Task<IReadOnlyList<CapabilityGrantStatus>> GetGrantStatusAsync(IReadOnlyList<string> ids, CancellationToken ct = default) => _store.GetGrantStatusAsync(ids, ct);
    public Task<CapabilityResolution> ResolveAsync(CapabilityResolveQuery query, CancellationToken ct = default) => _store.ResolveAsync(query, ct);
    public Task<CapabilityInvokeResult> InvokeAsync(CapabilityInvokeRequest request, CancellationToken ct = default) => _store.InvokeAsync(request, ct);
    public Task<CapabilityInvocation> GetInvocationAsync(string traceId, CancellationToken ct = default) => _store.GetInvocationAsync(traceId, ct);
}

public static class CapabilityRuntimeExtensions
{
    public static IServiceCollection AddPowerXCapabilityRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, ICapabilityRegistry> localFactory, Func<IServiceProvider, ICapabilityRegistry> delegatedFactory) => services.AddPowerXRuntime("capability", mode, localFactory, delegatedFactory);
}
