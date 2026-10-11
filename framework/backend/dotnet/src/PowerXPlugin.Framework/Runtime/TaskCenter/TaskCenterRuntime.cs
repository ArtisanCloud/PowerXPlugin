using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.TaskCenter;
public enum TaskState { Queued, Running, Succeeded, Failed, Cancelled }
public sealed record TaskScope(string TenantUuid);
public sealed record TaskCreateRequest(string Type, string IdempotencyKey, JsonElement Payload);
public sealed record TaskUpdateRequest(string TaskUuid, long ExpectedRevision, TaskState State, int Progress, string? MessageKey, JsonElement Result);
public sealed record TaskRecord(string TaskUuid, string TenantUuid, string Type, TaskState State, int Progress, long Revision, string? MessageKey, JsonElement Result, DateTimeOffset CreatedAt, DateTimeOffset UpdatedAt, DateTimeOffset? CompletedAt);
public interface ITaskCenterService { Task<TaskRecord> CreateAsync(TaskScope scope, TaskCreateRequest request, CancellationToken ct = default); Task<TaskRecord> GetAsync(TaskScope scope, string taskUuid, CancellationToken ct = default); Task<TaskRecord> UpdateAsync(TaskScope scope, TaskUpdateRequest request, CancellationToken ct = default); }
/// <summary>Plugin store must perform UpdateAsync atomically with revision CAS.</summary>
public interface ILocalTaskCenterStore : ITaskCenterService { }
public sealed class LocalTaskCenterAdapter(ILocalTaskCenterStore store) : ITaskCenterService
{
    private readonly ILocalTaskCenterStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public async Task<TaskRecord> CreateAsync(TaskScope s, TaskCreateRequest r, CancellationToken ct = default) { ValidateScope(s); if (string.IsNullOrWhiteSpace(r.Type) || string.IsNullOrWhiteSpace(r.IdempotencyKey) || !JsonValid(r.Payload)) throw new TaskCenterException(TaskCenterErrors.InvalidArgument); return Check(await _store.CreateAsync(s, r, ct), s, null); }
    public async Task<TaskRecord> GetAsync(TaskScope s, string id, CancellationToken ct = default) { ValidateScope(s); ValidateUuid(id); return Check(await _store.GetAsync(s, id, ct), s, id); }
    public async Task<TaskRecord> UpdateAsync(TaskScope s, TaskUpdateRequest r, CancellationToken ct = default) { ValidateScope(s); ValidateUpdate(r); var x = Check(await _store.UpdateAsync(s, r, ct), s, r.TaskUuid); if (x.Revision != r.ExpectedRevision + 1 || x.State != r.State || x.Progress != r.Progress) throw new TaskCenterException(TaskCenterErrors.InvalidResponse); return x; }
    internal static void ValidateScope(TaskScope s) { if (!Guid.TryParse(s.TenantUuid, out var id) || id == Guid.Empty || id.ToString() != s.TenantUuid) throw new TaskCenterException(TaskCenterErrors.InvalidArgument); }
    internal static void ValidateUuid(string v) { if (!Guid.TryParse(v, out var id) || id == Guid.Empty || id.ToString() != v) throw new TaskCenterException(TaskCenterErrors.InvalidArgument); }
    internal static void ValidateUpdate(TaskUpdateRequest r) { ValidateUuid(r.TaskUuid); if (r.ExpectedRevision < 1 || r.Progress is < 0 or > 100 || r.State == TaskState.Succeeded && r.Progress != 100 || !JsonValid(r.Result)) throw new TaskCenterException(TaskCenterErrors.InvalidArgument); }
    internal static bool JsonValid(JsonElement e) => e.ValueKind != JsonValueKind.Undefined;
    internal static TaskRecord Check(TaskRecord r, TaskScope s, string? id) { if (r is null || r.TenantUuid != s.TenantUuid || id is not null && r.TaskUuid != id || r.Revision < 1 || r.Progress is < 0 or > 100 || r.State == TaskState.Succeeded && r.Progress != 100 || IsTerminal(r.State) != (r.CompletedAt is not null)) throw new TaskCenterException(TaskCenterErrors.InvalidResponse); return r; }
    internal static bool IsTerminal(TaskState s) => s is TaskState.Succeeded or TaskState.Failed or TaskState.Cancelled;
}
public static class TaskCenterTransitions
{
    /// <summary>Local stores call this within their atomic CAS update.</summary>
    public static void Validate(TaskRecord current, TaskScope scope, TaskUpdateRequest update)
    {
        LocalTaskCenterAdapter.ValidateUpdate(update);
        if (current is null || current.TenantUuid != scope.TenantUuid || current.TaskUuid != update.TaskUuid) throw new TaskCenterException(TaskCenterErrors.NotFound);
        if (current.Revision != update.ExpectedRevision || current.Revision >= long.MaxValue || LocalTaskCenterAdapter.IsTerminal(current.State) || update.Progress < current.Progress || current.State == TaskState.Queued && update.State == TaskState.Succeeded || current.State == TaskState.Running && update.State == TaskState.Queued) throw new TaskCenterException(TaskCenterErrors.Conflict);
    }
}
public sealed class PowerXTaskCenterClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : ITaskCenterService
{
    private readonly string _base = string.IsNullOrWhiteSpace(baseUrl) ? throw new TaskCenterException(TaskCenterErrors.Unavailable) : baseUrl.TrimEnd('/'); private readonly string _tenant = tenantUuid; private readonly IServiceCredentialProvider _credentials = credentials; private readonly HttpClient _http = http ?? new();
    public Task<TaskRecord> CreateAsync(TaskScope s, TaskCreateRequest r, CancellationToken ct = default) { LocalTaskCenterAdapter.ValidateScope(s); if (s.TenantUuid != _tenant) throw new TaskCenterException(TaskCenterErrors.TenantMismatch); return SendAsync(HttpMethod.Post, "/api/v1/tenant/runtime/tasks", new { type = r.Type, idempotency_key = r.IdempotencyKey, payload = r.Payload }, s, null, ct); }
    public Task<TaskRecord> GetAsync(TaskScope s, string id, CancellationToken ct = default) { LocalTaskCenterAdapter.ValidateScope(s); LocalTaskCenterAdapter.ValidateUuid(id); if (s.TenantUuid != _tenant) throw new TaskCenterException(TaskCenterErrors.TenantMismatch); return SendAsync(HttpMethod.Get, "/api/v1/tenant/runtime/tasks/" + id, null, s, id, ct); }
    public Task<TaskRecord> UpdateAsync(TaskScope s, TaskUpdateRequest r, CancellationToken ct = default) { LocalTaskCenterAdapter.ValidateScope(s); LocalTaskCenterAdapter.ValidateUpdate(r); if (s.TenantUuid != _tenant) throw new TaskCenterException(TaskCenterErrors.TenantMismatch); return SendAsync(HttpMethod.Patch, "/api/v1/tenant/runtime/tasks/" + r.TaskUuid, new { expected_revision = r.ExpectedRevision, state = r.State.ToString().ToLowerInvariant(), progress = r.Progress, message_key = r.MessageKey ?? "", result = r.Result }, s, r.TaskUuid, ct); }
    private async Task<TaskRecord> SendAsync(HttpMethod m, string path, object? body, TaskScope s, string? id, CancellationToken ct) { var c = (await _credentials.GetCredentialAsync(ct)).Validate(); using var r = new HttpRequestMessage(m, _base + path); r.Headers.Authorization = new AuthenticationHeaderValue(c.NormalizedAuthScheme, c.Value); if (body is not null) r.Content = new StringContent(JsonSerializer.Serialize(body), Encoding.UTF8, "application/json"); try { using var resp = await _http.SendAsync(r, ct); var raw = await resp.Content.ReadAsStringAsync(ct); if (!resp.IsSuccessStatusCode) throw new TaskCenterException(resp.StatusCode == HttpStatusCode.Conflict ? TaskCenterErrors.Conflict : resp.StatusCode == HttpStatusCode.Forbidden ? TaskCenterErrors.Forbidden : TaskCenterErrors.Upstream); using var d = JsonDocument.Parse(raw); var task = JsonSerializer.Deserialize<TaskRecord>(d.RootElement.GetProperty("data").GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new TaskCenterException(TaskCenterErrors.InvalidResponse); return LocalTaskCenterAdapter.Check(task, s, id); } catch (TaskCenterException) { throw; } catch (HttpRequestException e) { throw new TaskCenterException(TaskCenterErrors.Upstream, e); } }
}
public static class TaskCenterRuntimeExtensions { public static IServiceCollection AddPowerXTaskCenterRuntime(this IServiceCollection s, ProviderMode m, Func<IServiceProvider, ITaskCenterService> l, Func<IServiceProvider, ITaskCenterService> d) => s.AddPowerXRuntime("taskcenter", m, l, d); }
public static class TaskCenterErrors { public const string InvalidArgument="FRAMEWORK_TASKCENTER_INVALID_ARGUMENT", InvalidResponse="FRAMEWORK_TASKCENTER_INVALID_RESPONSE", NotFound="FRAMEWORK_TASKCENTER_NOT_FOUND", Conflict="FRAMEWORK_TASKCENTER_CONFLICT", TenantMismatch="FRAMEWORK_TASKCENTER_TENANT_MISMATCH", Unavailable="FRAMEWORK_TASKCENTER_UNAVAILABLE", Forbidden="FRAMEWORK_TASKCENTER_FORBIDDEN", Upstream="FRAMEWORK_TASKCENTER_UPSTREAM_DEPENDENCY"; }
public sealed class TaskCenterException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; }=code; }
