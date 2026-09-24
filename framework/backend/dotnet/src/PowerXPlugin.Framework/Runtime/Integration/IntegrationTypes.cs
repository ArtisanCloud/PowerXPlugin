using System.Text.Json.Serialization;

namespace PowerXPlugin.Framework.Runtime.Integration;

public sealed record IntegrationRouteSummary(
    [property: JsonPropertyName("route_id")] string RouteUuid,
    [property: JsonPropertyName("route_slug")] string RouteSlug,
    [property: JsonPropertyName("capability_id")] string CapabilityId,
    [property: JsonPropertyName("channels")] IReadOnlyCollection<string> Channels,
    [property: JsonPropertyName("lifecycle_state")] string LifecycleState,
    [property: JsonPropertyName("status")] string Status,
    [property: JsonPropertyName("updated_at")] DateTimeOffset? UpdatedAt);
public sealed record IntegrationRouteDetail(
    [property: JsonPropertyName("route_id")] string RouteUuid,
    [property: JsonPropertyName("route_slug")] string RouteSlug,
    [property: JsonPropertyName("capability_id")] string CapabilityId,
    [property: JsonPropertyName("tool_grant_ids")] IReadOnlyCollection<string> ToolGrantIds,
    [property: JsonPropertyName("channels")] IReadOnlyCollection<string> Channels,
    [property: JsonPropertyName("lifecycle_state")] string LifecycleState,
    [property: JsonPropertyName("status")] string Status,
    [property: JsonPropertyName("description")] string? Description,
    [property: JsonPropertyName("current_version")] uint CurrentVersion,
    [property: JsonPropertyName("created_at")] DateTimeOffset? CreatedAt,
    [property: JsonPropertyName("updated_at")] DateTimeOffset? UpdatedAt);
public sealed record IntegrationRouteQuery(string? CapabilityId = null, string? Channel = null);
public sealed record IntegrationRouteInvocation(IReadOnlyDictionary<string, object?> Payload, string? IdempotencyKey = null, IReadOnlyDictionary<string, object?>? Context = null);
public sealed record IntegrationRouteResult(
    [property: JsonPropertyName("result")] IReadOnlyDictionary<string, object?> Result,
    [property: JsonPropertyName("routed_capability_id")] string RoutedCapabilityId,
    [property: JsonPropertyName("routed_adapter")] string? RoutedAdapter,
    [property: JsonPropertyName("trace_id")] string TraceId,
    [property: JsonPropertyName("dispatched_at")] DateTimeOffset? DispatchedAt);

/// <summary>Framework-only typed boundary for Core tenant Integration Gateway.</summary>
public interface IIntegrationGateway
{
    Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default);
    Task<IntegrationRouteDetail> GetRouteAsync(string routeSlug, CancellationToken ct = default);
    Task<IntegrationRouteResult> InvokeRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default);
}

/// <summary>
/// Plugin-owned route registry. The Framework never accepts a free endpoint,
/// capability ID, or action as a replacement for verified route ownership.
/// </summary>
public interface ILocalIntegrationRouteStore
{
    Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default);
    Task<IntegrationRouteDetail?> GetRouteAsync(string routeSlug, CancellationToken ct = default);
    Task<bool> IsRouteOwnedAsync(string routeSlug, CancellationToken ct = default);
    Task<IntegrationRouteResult> InvokeOwnedRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default);
}

public sealed class LocalIntegrationGateway(ILocalIntegrationRouteStore store) : IIntegrationGateway
{
    private readonly ILocalIntegrationRouteStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public Task<IReadOnlyList<IntegrationRouteSummary>> ListRoutesAsync(IntegrationRouteQuery query, CancellationToken ct = default) => _store.ListRoutesAsync(query, ct);
    public async Task<IntegrationRouteDetail> GetRouteAsync(string routeSlug, CancellationToken ct = default) => await _store.GetRouteAsync(RequireRoute(routeSlug), ct) ?? throw new IntegrationAdapterException(IntegrationErrors.CodeNotFound);
    public async Task<IntegrationRouteResult> InvokeRouteAsync(string routeSlug, IntegrationRouteInvocation invocation, CancellationToken ct = default)
    {
        routeSlug = RequireRoute(routeSlug);
        if (invocation.Payload is null || invocation.Payload.Count == 0) throw new IntegrationAdapterException(IntegrationErrors.CodeInvalidRequest);
        if (!await _store.IsRouteOwnedAsync(routeSlug, ct)) throw new IntegrationAdapterException(IntegrationErrors.CodeNotFound);
        return await _store.InvokeOwnedRouteAsync(routeSlug, invocation, ct);
    }
    private static string RequireRoute(string value) => string.IsNullOrWhiteSpace(value) ? throw new IntegrationAdapterException(IntegrationErrors.CodeInvalidRequest) : value.Trim();
}

public enum IntegrationAdapterMode { Local, Delegated }

public static class IntegrationErrors
{
    public const string CodeUnavailable = "FRAMEWORK_INTEGRATION_ADAPTER_UNAVAILABLE";
    public const string CodeInvalidRequest = "FRAMEWORK_INTEGRATION_INVALID_REQUEST";
    public const string CodeUnauthorized = "FRAMEWORK_INTEGRATION_UNAUTHORIZED";
    public const string CodeForbidden = "FRAMEWORK_INTEGRATION_FORBIDDEN";
    public const string CodeNotFound = "FRAMEWORK_INTEGRATION_NOT_FOUND";
    public const string CodeUpstreamDependency = "FRAMEWORK_INTEGRATION_UPSTREAM_DEPENDENCY";
}

public sealed class IntegrationAdapterException : InvalidOperationException
{
    public IntegrationAdapterException(string code, int? upstreamStatusCode = null, Exception? innerException = null) : base(code, innerException)
        => (Code, UpstreamStatusCode) = (code, upstreamStatusCode);
    public string Code { get; }
    public int? UpstreamStatusCode { get; }
}
