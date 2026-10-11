using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Agent;

/// <summary>
/// Delegated service-session adapter for the fixed Core Agent Host Contract.
/// It accepts no caller credential or tenant override; both are bound at
/// composition time. A broken or unfinished SSE stream always fails closed.
/// </summary>
public sealed class PowerXAgentSessionClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : IAgentSessionService
{
    private const string Root = "/api/v1/tenant/agent/sessions";
    private readonly string _baseUrl = baseUrl.TrimEnd('/');
    private readonly string _tenantUuid = tenantUuid;
    private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials));
    private readonly HttpClient _http = http ?? new HttpClient();
    private static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web);

    public async Task<AgentSession> CreateSessionAsync(AgentTenantScope scope, CreateAgentSessionRequest request, CancellationToken ct = default) =>
        Session(await SendAsync<SessionWire>(scope, HttpMethod.Post, Root, new { agent_uuid = request.AgentUuid, title = request.Title }, HttpStatusCode.Created, ct), scope);

    public async Task<AgentPage<AgentSession>> ListSessionsAsync(AgentTenantScope scope, AgentPageRequest page, CancellationToken ct = default)
    {
        Validate(scope); Validate(page);
        var wire = await SendAsync<SessionPageWire>(scope, HttpMethod.Get, $"{Root}?page={page.Page}&page_size={page.PageSize}", null, HttpStatusCode.OK, ct);
        return new AgentPage<AgentSession>(wire.Items?.Select(x => Session(x, scope)).ToArray() ?? throw InvalidResponse(), wire.Total, wire.Page, wire.PageSize);
    }

    public async Task<AgentSession> GetSessionAsync(AgentSessionScope scope, CancellationToken ct = default) => Session(await SendAsync<SessionWire>(scope, HttpMethod.Get, Path(scope), null, HttpStatusCode.OK, ct), scope);
    public async Task<AgentSession> RenameSessionAsync(AgentSessionScope scope, RenameAgentSessionRequest request, CancellationToken ct = default) => Session(await SendAsync<SessionWire>(scope, HttpMethod.Patch, Path(scope), new { title = request.Title }, HttpStatusCode.OK, ct), scope);
    public async Task<AgentSession> ArchiveSessionAsync(AgentSessionScope scope, CancellationToken ct = default) => Session(await SendAsync<SessionWire>(scope, HttpMethod.Post, Path(scope) + "/archive", null, HttpStatusCode.OK, ct), scope);
    public async Task<AgentSession> DeleteSessionAsync(AgentSessionScope scope, CancellationToken ct = default) => Session(await SendAsync<SessionWire>(scope, HttpMethod.Delete, Path(scope), null, HttpStatusCode.OK, ct), scope);

    public async Task<AgentMessage> AppendMessageAsync(AgentSessionScope scope, AppendAgentMessageRequest request, CancellationToken ct = default)
    {
        Validate(scope); ValidateMessageRequest(request);
        return Message(await SendAsync<MessageWire>(scope, HttpMethod.Post, Path(scope) + "/messages", new { role = "user", content = request.Content }, HttpStatusCode.Created, ct, request.IdempotencyKey), scope);
    }

    public async Task<AgentPage<AgentMessage>> ListMessagesAsync(AgentSessionScope scope, AgentPageRequest page, CancellationToken ct = default)
    {
        Validate(scope); Validate(page);
        var wire = await SendAsync<MessagePageWire>(scope, HttpMethod.Get, $"{Path(scope)}/messages?page={page.Page}&page_size={page.PageSize}", null, HttpStatusCode.OK, ct);
        return new AgentPage<AgentMessage>(wire.Items?.Select(x => Message(x, scope)).ToArray() ?? throw InvalidResponse(), wire.Total, wire.Page, wire.PageSize);
    }

    public async Task<AgentInvocation> StartAsync(AgentSessionScope scope, StartAgentInvocationRequest request, CancellationToken ct = default)
    {
        Validate(scope); ValidateInvocationRequest(request);
        return Invocation(await SendAsync<InvocationWire>(scope, HttpMethod.Post, Path(scope) + "/invocations", new { message_uuid = request.MessageUuid }, HttpStatusCode.Accepted, ct, request.IdempotencyKey), scope, request.MessageUuid);
    }

    public async Task<AgentInvocation> GetInvocationAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default)
    {
        Validate(scope); ValidateUuid(invocationUuid);
        return Invocation(await SendAsync<InvocationWire>(scope, HttpMethod.Get, InvocationPath(scope, invocationUuid), null, HttpStatusCode.OK, ct), scope, null, invocationUuid);
    }

    public async Task<AgentInvocation> CancelAsync(AgentSessionScope scope, string invocationUuid, CancellationToken ct = default)
    {
        Validate(scope); ValidateUuid(invocationUuid);
        var invocation = Invocation(await SendAsync<InvocationWire>(scope, HttpMethod.Post, InvocationPath(scope, invocationUuid) + "/cancel", null, HttpStatusCode.Accepted, ct), scope, null, invocationUuid);
        return invocation.State is AgentInvocationState.Cancelling or AgentInvocationState.Cancelled ? invocation : throw InvalidResponse();
    }

    public async Task StreamEventsAsync(AgentSessionScope scope, string invocationUuid, Func<AgentSessionEvent, CancellationToken, Task> onEvent, CancellationToken ct = default)
    {
        Validate(scope); ValidateUuid(invocationUuid);
        if (onEvent is null) throw InvalidArgument();
        var cursor = 0; var attempts = 0; var finalSeen = false; string? terminalError = null;
        while (true)
        {
            using var request = await RequestAsync(new AgentTenantScope(scope.TenantUuid), HttpMethod.Get, InvocationPath(scope, invocationUuid) + "/events", null, null, ct, acceptEvents: true, lastEventId: cursor == 0 ? null : cursor.ToString());
            using var response = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, ct);
            if (!response.IsSuccessStatusCode) throw TransportError(response.StatusCode);
            if (!string.Equals(response.Content.Headers.ContentType?.MediaType, "text/event-stream", StringComparison.OrdinalIgnoreCase)) throw InvalidResponse();
            await using var body = await response.Content.ReadAsStreamAsync(ct);
            using var reader = new StreamReader(body, Encoding.UTF8, true, 4096, leaveOpen: false);
            var type = ""; var data = new StringBuilder(); var eventId = 0; var ended = false;
            async Task FlushAsync()
            {
                if (data.Length == 0) { type = ""; return; }
                if (eventId is < 1 or > 3 || eventId <= cursor) throw InvalidResponse();
                var ev = DecodeEvent(type, data.ToString(), scope, invocationUuid, ref finalSeen, ref terminalError);
                await onEvent(ev, ct);
                cursor = eventId; data.Clear(); type = ""; eventId = 0;
                if (ev.Type == "end") ended = true;
            }
            while (await reader.ReadLineAsync(ct) is { } line)
            {
                if (line.Length == 0) { await FlushAsync(); if (ended) break; continue; }
                if (line.StartsWith(':')) continue;
                var separator = line.IndexOf(':'); var field = separator < 0 ? line : line[..separator]; var value = separator < 0 ? "" : line[(separator + 1)..].TrimStart(' ');
                if (field == "id" && (!int.TryParse(value, out eventId) || eventId is < 1 or > 3)) throw InvalidResponse();
                else if (field == "event") type = value;
                else if (field == "data") { if (data.Length + value.Length > 1_048_576) throw InvalidResponse(); if (data.Length > 0) data.Append('\n'); data.Append(value); }
            }
            if (ended) { if (terminalError is not null) throw new AgentRuntimeException(terminalError); return; }
            if (ct.IsCancellationRequested) ct.ThrowIfCancellationRequested();
            if (++attempts > 3) throw new AgentRuntimeException(AgentRuntimeErrors.StreamInterrupted);
            await Task.Delay(TimeSpan.FromMilliseconds(100 * attempts), ct);
        }
    }

    private async Task<T> SendAsync<T>(AgentTenantScope scope, HttpMethod method, string path, object? body, HttpStatusCode expected, CancellationToken ct, string? idempotencyKey = null)
    {
        Validate(scope);
        using var request = await RequestAsync(scope, method, path, body, idempotencyKey, ct);
        using var response = await _http.SendAsync(request, ct);
        var raw = await response.Content.ReadAsStringAsync(ct);
        if (!response.IsSuccessStatusCode) throw TransportError(response.StatusCode);
        if (response.StatusCode != expected) throw InvalidResponse();
        try
        {
            using var document = JsonDocument.Parse(raw);
            if (!document.RootElement.TryGetProperty("code", out var code) || code.GetInt32() != (int)expected || !document.RootElement.TryGetProperty("data", out var data) || data.ValueKind is JsonValueKind.Null or JsonValueKind.Undefined) throw InvalidResponse();
            return JsonSerializer.Deserialize<T>(data.GetRawText(), Json) ?? throw InvalidResponse();
        }
        catch (JsonException) { throw InvalidResponse(); }
    }

    private Task<T> SendAsync<T>(AgentSessionScope scope, HttpMethod method, string path, object? body, HttpStatusCode expected, CancellationToken ct, string? idempotencyKey = null)
    {
        Validate(scope);
        return SendAsync<T>(new AgentTenantScope(scope.TenantUuid), method, path, body, expected, ct, idempotencyKey);
    }

    private async Task<HttpRequestMessage> RequestAsync(AgentTenantScope scope, HttpMethod method, string path, object? body, string? idempotencyKey, CancellationToken ct, bool acceptEvents = false, string? lastEventId = null)
    {
        Validate(scope);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        var request = new HttpRequestMessage(method, _baseUrl + path);
        request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        if (acceptEvents) request.Headers.Accept.ParseAdd("text/event-stream");
        if (lastEventId is not null) request.Headers.TryAddWithoutValidation("Last-Event-ID", lastEventId);
        if (!string.IsNullOrEmpty(idempotencyKey)) request.Headers.TryAddWithoutValidation("Idempotency-Key", idempotencyKey);
        if (body is not null) request.Content = new StringContent(JsonSerializer.Serialize(body, Json), Encoding.UTF8, "application/json");
        return request;
    }

    private void Validate(AgentTenantScope scope) { if (scope is null || scope.TenantUuid != _tenantUuid || !IsUuid(scope.TenantUuid)) throw new AgentRuntimeException(scope?.TenantUuid == _tenantUuid ? AgentRuntimeErrors.InvalidArgument : "FRAMEWORK_AGENT_TENANT_MISMATCH"); }
    private void Validate(AgentSessionScope scope) { if (scope is null || scope.TenantUuid != _tenantUuid || !IsUuid(scope.TenantUuid) || !IsUuid(scope.SessionUuid)) throw new AgentRuntimeException(scope?.TenantUuid == _tenantUuid ? AgentRuntimeErrors.InvalidArgument : "FRAMEWORK_AGENT_TENANT_MISMATCH"); }
    private static void Validate(AgentPageRequest page) { if (page.Page is < 1 or > 1_000_000 || page.PageSize is < 1 or > 100) throw InvalidArgument(); }
    private static void ValidateUuid(string value) { if (!IsUuid(value)) throw InvalidArgument(); }
    private static void ValidateMessageRequest(AppendAgentMessageRequest request) { if (request is null || string.IsNullOrWhiteSpace(request.Content) || request.Content.Length > 65536 || !Key(request.IdempotencyKey) || request.IdempotencyKey.StartsWith("invoke:", StringComparison.Ordinal)) throw InvalidArgument(); }
    private static void ValidateInvocationRequest(StartAgentInvocationRequest request) { if (request is null || !IsUuid(request.MessageUuid) || !Key(request.IdempotencyKey)) throw InvalidArgument(); }
    private static string Path(AgentSessionScope scope) => Root + "/" + scope.SessionUuid;
    private static string InvocationPath(AgentSessionScope scope, string invocationUuid) => Path(scope) + "/invocations/" + invocationUuid;
    private static AgentSession Session(SessionWire wire, AgentTenantScope scope) => wire.SessionUuid is { } id && IsUuid(id) && IsUuid(wire.AgentUuid) && wire.Revision > 0 && wire.CreatedAt != default && wire.UpdatedAt != default ? new(id, scope.TenantUuid, wire.AgentUuid, wire.Title ?? "", wire.Status switch { "active" => AgentSessionStatus.Active, "archived" => AgentSessionStatus.Archived, "deleted" => AgentSessionStatus.Deleted, _ => throw InvalidResponse() }, wire.Revision, wire.CreatedAt, wire.UpdatedAt) : throw InvalidResponse();
    private static AgentSession Session(SessionWire wire, AgentSessionScope scope) => Session(wire, new AgentTenantScope(scope.TenantUuid)).SessionUuid == scope.SessionUuid ? Session(wire, new AgentTenantScope(scope.TenantUuid)) : throw InvalidResponse();
    private static AgentMessage Message(MessageWire wire, AgentSessionScope scope) => wire.MessageUuid is { } id && wire.SessionUuid == scope.SessionUuid && IsUuid(id) && wire.Sequence > 0 && wire.CreatedAt != default && wire.Role is "user" or "assistant" ? new(id, scope.TenantUuid, scope.SessionUuid, wire.Role == "user" ? AgentMessageRole.User : AgentMessageRole.Assistant, wire.Content ?? "", wire.Sequence, wire.CreatedAt) : throw InvalidResponse();
    private static AgentInvocation Invocation(InvocationWire wire, AgentSessionScope scope, string? messageUuid = null, string? invocationUuid = null) => wire.InvocationUuid is { } id && wire.SessionUuid == scope.SessionUuid && IsUuid(id) && IsUuid(wire.MessageUuid) && IsUuid(wire.TraceUuid) && wire.CreatedAt != default && wire.DeadlineAt != default && (messageUuid is null || wire.MessageUuid == messageUuid) && (invocationUuid is null || id == invocationUuid) ? new(id, scope.TenantUuid, scope.SessionUuid, wire.MessageUuid, wire.TraceUuid, wire.Status switch { "running" => AgentInvocationState.Running, "cancelling" => AgentInvocationState.Cancelling, "succeeded" => AgentInvocationState.Succeeded, "failed" => AgentInvocationState.Failed, "cancelled" => AgentInvocationState.Cancelled, _ => throw InvalidResponse() }, "", wire.CreatedAt, wire.DeadlineAt, wire.FinishedAt, wire.ReasonCode ?? "", wire.Output ?? "") : throw InvalidResponse();
    private static AgentSessionEvent DecodeEvent(string type, string raw, AgentSessionScope scope, string invocationUuid, ref bool finalSeen, ref string? terminalError)
    {
        try
        {
            if (type is "state" or "final")
            {
                var invocation = Invocation(JsonSerializer.Deserialize<InvocationWire>(raw, Json) ?? throw InvalidResponse(), scope, null, invocationUuid);
                if (type == "final" && (invocation.State != AgentInvocationState.Succeeded || finalSeen || terminalError is not null)) throw InvalidResponse();
                if (type == "final") finalSeen = true;
                return new(type, invocation, invocation.State.ToString().ToLowerInvariant());
            }
            if (type == "error") { var error = JsonSerializer.Deserialize<ErrorWire>(raw, Json)?.ReasonCode; if (string.IsNullOrWhiteSpace(error) || finalSeen || terminalError is not null) throw InvalidResponse(); terminalError = error; return new(type, null, "", error); }
            if (type == "end") { var status = JsonSerializer.Deserialize<EndWire>(raw, Json)?.Status; if (status is not ("succeeded" or "failed" or "cancelled")) throw InvalidResponse(); if ((status == "succeeded" && (!finalSeen || terminalError is not null)) || (status != "succeeded" && terminalError is null)) throw InvalidResponse(); return new(type, null, status); }
        }
        catch (JsonException) { throw InvalidResponse(); }
        throw InvalidResponse();
    }
    private static bool IsUuid(string? value) => Guid.TryParse(value, out var uuid) && uuid != Guid.Empty && uuid.ToString() == value;
    private static bool Key(string? value) => value is { Length: > 0 and <= 128 } && value == value.Trim() && !value.Contains('\r') && !value.Contains('\n');
    private static AgentRuntimeException InvalidArgument() => new(AgentRuntimeErrors.InvalidArgument);
    private static AgentRuntimeException InvalidResponse() => new(AgentRuntimeErrors.InvalidResponse);
    private static AgentRuntimeException TransportError(HttpStatusCode code) => new(code == HttpStatusCode.Forbidden ? "FRAMEWORK_AGENT_FORBIDDEN" : "FRAMEWORK_AGENT_UPSTREAM_DEPENDENCY");

    private sealed record SessionWire([property: JsonPropertyName("session_uuid")] string? SessionUuid, [property: JsonPropertyName("agent_uuid")] string AgentUuid, string? Title, string? Status, long Revision, [property: JsonPropertyName("created_at")] DateTimeOffset CreatedAt, [property: JsonPropertyName("updated_at")] DateTimeOffset UpdatedAt);
    private sealed record MessageWire([property: JsonPropertyName("message_uuid")] string? MessageUuid, [property: JsonPropertyName("session_uuid")] string? SessionUuid, string? Role, string? Content, long Sequence, [property: JsonPropertyName("created_at")] DateTimeOffset CreatedAt);
    private sealed record InvocationWire([property: JsonPropertyName("invocation_uuid")] string? InvocationUuid, [property: JsonPropertyName("session_uuid")] string? SessionUuid, [property: JsonPropertyName("message_uuid")] string MessageUuid, [property: JsonPropertyName("trace_uuid")] string TraceUuid, string? Status, [property: JsonPropertyName("reason_code")] string? ReasonCode, string? Output, [property: JsonPropertyName("created_at")] DateTimeOffset CreatedAt, [property: JsonPropertyName("deadline_at")] DateTimeOffset DeadlineAt, [property: JsonPropertyName("finished_at")] DateTimeOffset? FinishedAt);
    private sealed record SessionPageWire(IReadOnlyList<SessionWire>? Items, long Total, int Page, [property: JsonPropertyName("page_size")] int PageSize);
    private sealed record MessagePageWire(IReadOnlyList<MessageWire>? Items, long Total, int Page, [property: JsonPropertyName("page_size")] int PageSize);
    private sealed record ErrorWire([property: JsonPropertyName("reason_code")] string? ReasonCode);
    private sealed record EndWire(string? Status);
}
