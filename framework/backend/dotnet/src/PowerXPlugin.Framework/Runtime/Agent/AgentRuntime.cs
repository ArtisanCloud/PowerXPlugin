using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Agent;

/// <summary>
/// Typed service-session contract shared by local and delegated Agent Runtime
/// adapters. A service actor never supplies an identity override: tenant scope
/// is injected by the plugin runtime and checked on every returned object.
/// </summary>
public enum AgentSessionStatus { Active, Archived, Deleted }
public enum AgentMessageRole { User, Assistant }
public enum AgentInvocationState { Running, Cancelling, Succeeded, Failed, Cancelled }

public sealed record AgentTenantScope(string TenantUuid);
public sealed record AgentSessionScope(string TenantUuid, string SessionUuid);
public sealed record AgentPageRequest(int Page = 1, int PageSize = 20);
public sealed record AgentPage<T>(IReadOnlyList<T> Items, long Total, int Page, int PageSize);

public sealed record AgentSession(string SessionUuid, string TenantUuid, string AgentUuid, string Title, AgentSessionStatus Status, long Revision, DateTimeOffset CreatedAt, DateTimeOffset UpdatedAt);
public sealed record AgentMessage(string MessageUuid, string TenantUuid, string SessionUuid, AgentMessageRole Role, string Content, long Sequence, DateTimeOffset CreatedAt);
public sealed record AgentInvocation(string InvocationUuid, string TenantUuid, string SessionUuid, string MessageUuid, string TraceUuid, AgentInvocationState State, string IdempotencyKey, DateTimeOffset CreatedAt, DateTimeOffset DeadlineAt, DateTimeOffset? FinishedAt = null, string ReasonCode = "", string Output = "");

public sealed record CreateAgentSessionRequest(string AgentUuid, string Title);
public sealed record RenameAgentSessionRequest(string Title);
public sealed record AppendAgentMessageRequest(string Content, string IdempotencyKey);
public sealed record StartAgentInvocationRequest(string MessageUuid, string IdempotencyKey);
public sealed record AgentLifecycleScope(string TenantUuid, string AgentUuid);
public sealed record AgentHealthMetrics(double ThroughputPerMinute, double SuccessRate, int P95LatencyMs, double ResourceUtilizationPercent, double ErrorRate);
public sealed record AgentHealthSummary(string TenantUuid, string AgentUuid, string Status, int HealthScore, DateTimeOffset UpdatedAt, int WindowDurationSeconds, AgentHealthMetrics Metrics, IReadOnlyList<string> Recommendations, IReadOnlyList<string> AnomalyTraceUuids);
public sealed record AgentHealthHistory(IReadOnlyList<AgentHealthSummary> Snapshots);
public sealed record AgentBridgeControlRequest(string Reason = "", string TraceUuid = "");
public sealed record AgentBridgeRebalanceRequest(int TargetCapacityInstances, string Reason = "", string TraceUuid = "");
public sealed record AgentBridgeIdentity(string AgentUuid, string TenantUuid, string Alias, string Status);
public sealed record AgentBridgeLifecycleResult(AgentBridgeIdentity Agent, object? Event = null);
public sealed record AgentSessionEvent(string Type, AgentInvocation? Invocation = null, string Status = "", string ReasonCode = "");

/// <summary>
/// StartAsync is the local transaction boundary. It must atomically return an
/// idempotency replay or reject a second running/cancelling invocation in a
/// tenant/session. The framework deliberately has no in-memory fallback.
/// </summary>
public enum AgentStartDisposition { Started, IdempotentReplay, ActiveInvocationConflict }
public sealed record AgentStartResult(AgentStartDisposition Disposition, AgentInvocation? Invocation);

public interface ILocalAgentStore
{
    Task<AgentSession> CreateSessionAsync(AgentTenantScope scope, CreateAgentSessionRequest request, CancellationToken ct = default);
    Task<AgentPage<AgentSession>> ListSessionsAsync(AgentTenantScope scope, AgentPageRequest page, CancellationToken ct = default);
    Task<AgentSession> GetSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentSession> RenameSessionAsync(AgentSessionScope scope, RenameAgentSessionRequest request, CancellationToken ct = default);
    Task<AgentSession> ArchiveSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentSession> DeleteSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentMessage> AppendMessageAsync(AgentSessionScope scope, AppendAgentMessageRequest request, CancellationToken ct = default);
    Task<AgentPage<AgentMessage>> ListMessagesAsync(AgentSessionScope scope, AgentPageRequest page, CancellationToken ct = default);
    Task<AgentStartResult> StartAsync(AgentSessionScope scope, StartAgentInvocationRequest request, CancellationToken ct = default);
    Task<AgentInvocation> GetInvocationAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default);
    Task<AgentInvocation> CancelAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default);
    Task StreamEventsAsync(AgentSessionScope scope, string invocationUuid, Func<AgentSessionEvent, CancellationToken, Task> onEvent, CancellationToken ct = default);
}

public interface IAgentSessionService
{
    Task<AgentSession> CreateSessionAsync(AgentTenantScope scope, CreateAgentSessionRequest request, CancellationToken ct = default);
    Task<AgentPage<AgentSession>> ListSessionsAsync(AgentTenantScope scope, AgentPageRequest page, CancellationToken ct = default);
    Task<AgentSession> GetSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentSession> RenameSessionAsync(AgentSessionScope scope, RenameAgentSessionRequest request, CancellationToken ct = default);
    Task<AgentSession> ArchiveSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentSession> DeleteSessionAsync(AgentSessionScope scope, CancellationToken ct = default);
    Task<AgentMessage> AppendMessageAsync(AgentSessionScope scope, AppendAgentMessageRequest request, CancellationToken ct = default);
    Task<AgentPage<AgentMessage>> ListMessagesAsync(AgentSessionScope scope, AgentPageRequest page, CancellationToken ct = default);
    Task<AgentInvocation> StartAsync(AgentSessionScope scope, StartAgentInvocationRequest request, CancellationToken ct = default);
    Task<AgentInvocation> GetInvocationAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default);
    Task<AgentInvocation> CancelAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default);
    Task StreamEventsAsync(AgentSessionScope scope, string invocationUuid, Func<AgentSessionEvent, CancellationToken, Task> onEvent, CancellationToken ct = default);
}

/// <summary>
/// Lifecycle administration is kept separate from service-session execution.
/// Local stores must remain tenant-isolated; startup mode chooses the local or
/// delegated implementation without an invocation-time override.
/// </summary>
public interface IAgentLifecycleService
{
    Task<AgentHealthSummary> GetHealthSummaryAsync(AgentLifecycleScope scope, CancellationToken ct = default);
    Task<AgentHealthHistory> ListHealthHistoryAsync(AgentLifecycleScope scope, int rangeHours, int limit, CancellationToken ct = default);
    Task<object> GetBridgeStateAsync(AgentLifecycleScope scope, int limit, CancellationToken ct = default);
    Task<AgentBridgeLifecycleResult> FreezeAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default);
    Task<AgentBridgeLifecycleResult> RecoverAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default);
    Task<AgentBridgeLifecycleResult> RebalanceAsync(AgentLifecycleScope scope, AgentBridgeRebalanceRequest request, CancellationToken ct = default);
}

public interface ILocalAgentLifecycleStore : IAgentLifecycleService { }

public sealed class LocalAgentLifecycleService(ILocalAgentLifecycleStore store) : IAgentLifecycleService
{
    private readonly ILocalAgentLifecycleStore _store = store ?? throw new ArgumentNullException(nameof(store));

    public async Task<AgentHealthSummary> GetHealthSummaryAsync(AgentLifecycleScope scope, CancellationToken ct = default) => Validate(await _store.GetHealthSummaryAsync(Validate(scope), ct), scope);
    public async Task<AgentHealthHistory> ListHealthHistoryAsync(AgentLifecycleScope scope, int rangeHours, int limit, CancellationToken ct = default)
    {
        Validate(scope);
        if (rangeHours is < 0 or > 8784 || limit is < 0 or > 1000) throw InvalidArgument();
        var history = await _store.ListHealthHistoryAsync(scope, rangeHours, limit, ct);
        if (history?.Snapshots is null) throw InvalidResponse();
        foreach (var item in history.Snapshots) Validate(item, scope);
        return history;
    }
    public async Task<object> GetBridgeStateAsync(AgentLifecycleScope scope, int limit, CancellationToken ct = default)
    {
        Validate(scope);
        if (limit is < 0 or > 1000) throw InvalidArgument();
        return await _store.GetBridgeStateAsync(scope, limit, ct) ?? throw InvalidResponse();
    }
    public async Task<AgentBridgeLifecycleResult> FreezeAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default) => Validate(await _store.FreezeAsync(Validate(scope), Validate(request), ct), scope);
    public async Task<AgentBridgeLifecycleResult> RecoverAsync(AgentLifecycleScope scope, AgentBridgeControlRequest request, CancellationToken ct = default) => Validate(await _store.RecoverAsync(Validate(scope), Validate(request), ct), scope);
    public async Task<AgentBridgeLifecycleResult> RebalanceAsync(AgentLifecycleScope scope, AgentBridgeRebalanceRequest request, CancellationToken ct = default)
    {
        Validate(scope);
        if (request is null || request.TargetCapacityInstances is < 1 or > 100_000 || !IsOptionalUuid(request.TraceUuid)) throw InvalidArgument();
        return Validate(await _store.RebalanceAsync(scope, request, ct), scope);
    }
    private static AgentLifecycleScope Validate(AgentLifecycleScope scope)
    {
        if (scope is null || !IsUuid(scope.TenantUuid) || !IsUuid(scope.AgentUuid)) throw InvalidArgument();
        return scope;
    }
    private static AgentBridgeControlRequest Validate(AgentBridgeControlRequest request)
    {
        if (request is null || !IsOptionalUuid(request.TraceUuid)) throw InvalidArgument();
        return request;
    }
    private static AgentHealthSummary Validate(AgentHealthSummary? summary, AgentLifecycleScope scope)
    {
        if (summary is null || summary.TenantUuid != scope.TenantUuid || summary.AgentUuid != scope.AgentUuid || summary.HealthScore is < 0 or > 100 || summary.UpdatedAt == default || summary.WindowDurationSeconds < 0 || summary.Recommendations is null || summary.AnomalyTraceUuids is null) throw InvalidResponse();
        return summary;
    }
    private static AgentBridgeLifecycleResult Validate(AgentBridgeLifecycleResult? result, AgentLifecycleScope scope)
    {
        if (result?.Agent is null || result.Agent.TenantUuid != scope.TenantUuid || result.Agent.AgentUuid != scope.AgentUuid) throw InvalidResponse();
        return result;
    }
    private static bool IsUuid(string? value) => Guid.TryParse(value, out var uuid) && uuid != Guid.Empty && uuid.ToString() == value;
    private static bool IsOptionalUuid(string? value) => string.IsNullOrEmpty(value) || IsUuid(value);
    private static AgentRuntimeException InvalidArgument() => new(AgentRuntimeErrors.InvalidArgument);
    private static AgentRuntimeException InvalidResponse() => new(AgentRuntimeErrors.InvalidResponse);
}

public sealed class LocalAgentSessionService(ILocalAgentStore store) : IAgentSessionService
{
    private readonly ILocalAgentStore _store = store ?? throw new ArgumentNullException(nameof(store));

    public async Task<AgentSession> CreateSessionAsync(AgentTenantScope scope, CreateAgentSessionRequest request, CancellationToken ct = default)
    {
        Validate(scope);
        if (!IsUuid(request.AgentUuid) || !IsTitle(request.Title)) throw InvalidArgument();
        return Validate(await _store.CreateSessionAsync(scope, request, ct), scope);
    }

    public async Task<AgentPage<AgentSession>> ListSessionsAsync(AgentTenantScope scope, AgentPageRequest page, CancellationToken ct = default)
    {
        Validate(scope); Validate(page);
        return Validate(await _store.ListSessionsAsync(scope, page, ct), scope);
    }

    public async Task<AgentSession> GetSessionAsync(AgentSessionScope scope, CancellationToken ct = default)
    {
        Validate(scope);
        return Validate(await _store.GetSessionAsync(scope, ct), scope);
    }

    public async Task<AgentSession> RenameSessionAsync(AgentSessionScope scope, RenameAgentSessionRequest request, CancellationToken ct = default)
    {
        Validate(scope);
        if (!IsTitle(request.Title)) throw InvalidArgument();
        return Validate(await _store.RenameSessionAsync(scope, request, ct), scope);
    }

    public async Task<AgentSession> ArchiveSessionAsync(AgentSessionScope scope, CancellationToken ct = default)
    {
        Validate(scope);
        var session = Validate(await _store.ArchiveSessionAsync(scope, ct), scope);
        return session.Status is AgentSessionStatus.Archived ? session : throw InvalidResponse();
    }

    public async Task<AgentSession> DeleteSessionAsync(AgentSessionScope scope, CancellationToken ct = default)
    {
        Validate(scope);
        var session = Validate(await _store.DeleteSessionAsync(scope, ct), scope);
        return session.Status is AgentSessionStatus.Deleted ? session : throw InvalidResponse();
    }

    public async Task<AgentMessage> AppendMessageAsync(AgentSessionScope scope, AppendAgentMessageRequest request, CancellationToken ct = default)
    {
        Validate(scope);
        if (string.IsNullOrWhiteSpace(request.Content) || request.Content.Length > 65536 || !IsIdempotencyKey(request.IdempotencyKey) || request.IdempotencyKey.StartsWith("invoke:", StringComparison.Ordinal)) throw InvalidArgument();
        return Validate(await _store.AppendMessageAsync(scope, request, ct), scope);
    }

    public async Task<AgentPage<AgentMessage>> ListMessagesAsync(AgentSessionScope scope, AgentPageRequest page, CancellationToken ct = default)
    {
        Validate(scope); Validate(page);
        return Validate(await _store.ListMessagesAsync(scope, page, ct), scope);
    }

    public async Task<AgentInvocation> StartAsync(AgentSessionScope scope, StartAgentInvocationRequest request, CancellationToken ct = default)
    {
        Validate(scope);
        if (!IsUuid(request.MessageUuid) || !IsIdempotencyKey(request.IdempotencyKey)) throw InvalidArgument();
        var result = await _store.StartAsync(scope, request, ct);
        if (result.Disposition is AgentStartDisposition.ActiveInvocationConflict) throw new AgentRuntimeException(AgentRuntimeErrors.SessionActive);
        return Validate(result.Invocation, scope, request.MessageUuid);
    }

    public async Task<AgentInvocation> GetInvocationAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default)
    {
        Validate(scope); if (!IsUuid(invocationUuid)) throw InvalidArgument();
        var invocation = Validate(await _store.GetInvocationAsync(scope, invocationUuid, ct), scope);
        return invocation.InvocationUuid == invocationUuid ? invocation : throw InvalidResponse();
    }

    public async Task<AgentInvocation> CancelAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default)
    {
        Validate(scope); if (!IsUuid(invocationUuid)) throw InvalidArgument();
        var invocation = Validate(await _store.CancelAsync(scope, invocationUuid, ct), scope);
        if (invocation.InvocationUuid != invocationUuid || invocation.State is not (AgentInvocationState.Cancelling or AgentInvocationState.Cancelled)) throw InvalidResponse();
        return invocation;
    }

    public async Task StreamEventsAsync(AgentSessionScope scope, string invocationUuid, Func<AgentSessionEvent, CancellationToken, Task> onEvent, CancellationToken ct = default)
    {
        Validate(scope); if (!IsUuid(invocationUuid) || onEvent is null) throw InvalidArgument();
        var ended = false;
        await _store.StreamEventsAsync(scope, invocationUuid, async (ev, callbackCt) =>
        {
            Validate(ev, scope, invocationUuid);
            if (ev.Type == "end") ended = true;
            await onEvent(ev, callbackCt);
        }, ct);
        if (!ended) throw new AgentRuntimeException("FRAMEWORK_AGENT_SESSION_STREAM_INTERRUPTED");
    }

    private static void Validate(AgentTenantScope scope) { if (scope is null || !IsUuid(scope.TenantUuid)) throw InvalidArgument(); }
    private static void Validate(AgentSessionScope scope) { if (scope is null || !IsUuid(scope.TenantUuid) || !IsUuid(scope.SessionUuid)) throw InvalidArgument(); }
    private static void Validate(AgentPageRequest page) { if (page.Page is < 1 or > 1_000_000 || page.PageSize is < 1 or > 100) throw InvalidArgument(); }
    private static AgentSession Validate(AgentSession? session, AgentTenantScope scope)
    {
        if (session is null || session.TenantUuid != scope.TenantUuid || !IsUuid(session.SessionUuid) || !IsUuid(session.AgentUuid) || session.Revision < 1 || session.CreatedAt == default || session.UpdatedAt == default) throw InvalidResponse();
        return session;
    }
    private static AgentSession Validate(AgentSession? session, AgentSessionScope scope) => Validate(session, new AgentTenantScope(scope.TenantUuid)).SessionUuid == scope.SessionUuid ? session! : throw InvalidResponse();
    private static AgentMessage Validate(AgentMessage? message, AgentSessionScope scope)
    {
        if (message is null || message.TenantUuid != scope.TenantUuid || message.SessionUuid != scope.SessionUuid || !IsUuid(message.MessageUuid) || message.Sequence < 1 || message.CreatedAt == default || message.Role is not (AgentMessageRole.User or AgentMessageRole.Assistant)) throw InvalidResponse();
        return message;
    }
    private static AgentInvocation Validate(AgentInvocation? invocation, AgentSessionScope scope, string? messageUuid = null)
    {
        if (invocation is null || invocation.TenantUuid != scope.TenantUuid || invocation.SessionUuid != scope.SessionUuid || !IsUuid(invocation.InvocationUuid) || !IsUuid(invocation.MessageUuid) || !IsUuid(invocation.TraceUuid) || invocation.CreatedAt == default || invocation.DeadlineAt == default || (messageUuid is not null && invocation.MessageUuid != messageUuid)) throw InvalidResponse();
        return invocation;
    }
    private static void Validate(AgentSessionEvent? ev, AgentSessionScope scope, string invocationUuid)
    {
        if (ev is null || ev.Type is not ("state" or "final" or "error" or "end")) throw InvalidResponse();
        if (ev.Type is "state" or "final")
        {
            _ = Validate(ev.Invocation, scope);
            if (ev.Invocation!.InvocationUuid != invocationUuid) throw InvalidResponse();
        }
        if (ev.Type == "error" && string.IsNullOrWhiteSpace(ev.ReasonCode)) throw InvalidResponse();
    }
    private static AgentPage<AgentSession> Validate(AgentPage<AgentSession>? page, AgentTenantScope scope)
    {
        if (page is null || page.Items is null || page.Total < 0 || page.Page < 1 || page.PageSize < 1) throw InvalidResponse();
        foreach (var item in page.Items) Validate(item, scope);
        return page;
    }
    private static AgentPage<AgentMessage> Validate(AgentPage<AgentMessage>? page, AgentSessionScope scope)
    {
        if (page is null || page.Items is null || page.Total < 0 || page.Page < 1 || page.PageSize < 1) throw InvalidResponse();
        foreach (var item in page.Items) Validate(item, scope);
        return page;
    }
    private static bool IsUuid(string? value) => Guid.TryParse(value, out var uuid) && uuid != Guid.Empty && uuid.ToString() == value;
    private static bool IsTitle(string? value) => value is not null && value.Length <= 255;
    private static bool IsIdempotencyKey(string? value) => value is { Length: > 0 and <= 128 } && value == value.Trim() && !value.Contains('\r') && !value.Contains('\n');
    private static AgentRuntimeException InvalidArgument() => new(AgentRuntimeErrors.InvalidArgument);
    private static AgentRuntimeException InvalidResponse() => new(AgentRuntimeErrors.InvalidResponse);
}

public static class AgentRuntimeExtensions
{
    public static IServiceCollection AddPowerXAgentSessionRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, IAgentSessionService> local, Func<IServiceProvider, IAgentSessionService> delegated) => services.AddPowerXRuntime("agent.sessions", mode, local, delegated);
    public static IServiceCollection AddPowerXAgentLifecycleRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, IAgentLifecycleService> local, Func<IServiceProvider, IAgentLifecycleService> delegated) => services.AddPowerXRuntime("agent.lifecycle", mode, local, delegated);
}

public static class AgentRuntimeErrors
{
    public const string InvalidArgument = "FRAMEWORK_AGENT_INVALID_ARGUMENT";
    public const string InvalidResponse = "FRAMEWORK_AGENT_INVALID_RESPONSE";
    public const string SessionActive = "FRAMEWORK_AGENT_SESSION_ACTIVE";
    public const string StreamInterrupted = "FRAMEWORK_AGENT_SESSION_STREAM_INTERRUPTED";
}

public sealed class AgentRuntimeException(string code) : InvalidOperationException(code) { public string Code { get; } = code; }
