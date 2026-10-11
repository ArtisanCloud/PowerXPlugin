using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.IAM.Delegated;

public sealed class IdentityContextException(int status, string code, string? reasonCode = null,
    string? traceId = null, string? requestId = null, string? sourceTraceId = null, string? sourceRequestId = null)
    : IAMAdapterException(reasonCode ?? code)
{
    public int Status { get; } = status;
    public string OriginalCode { get; } = code;
    public string? ReasonCode { get; } = reasonCode;
    public string? TraceId { get; } = traceId;
    public string? RequestId { get; } = requestId;
    public string? SourceTraceId { get; } = sourceTraceId;
    public string? SourceRequestId { get; } = sourceRequestId;
}

public sealed partial class PowerXIamClient : IDelegatedIdentityService,IWSIdentityRevalidationService
{
    public async Task<DelegatedIdentity> ResolveDelegatedIdentity(DelegatedIdentityQuery query, CancellationToken ct = default)
    {
        ArgumentNullException.ThrowIfNull(query); query.Validate();
        var result=await InvokeIdentity("resolve_request_identity",new {schema="powerx.iam.plugin-request-identity-query/v1",context_ticket=query.ContextTicket},ct);
        var identity=DelegatedIdentity.FromCore(result,JsonOptions);identity.Validate(_options.PluginId,DateTimeOffset.UtcNow);return identity;
    }
    public async Task<WSIdentity> RevalidateWSIdentity(WSIdentityQuery query,CancellationToken ct=default)
    {
        ArgumentNullException.ThrowIfNull(query);query.Validate();
        var result=await InvokeIdentity("revalidate_ws_identity",new {schema="powerx.iam.plugin-ws-identity-query/v1",ws_revalidation_handle=query.Handle},ct);
        var identity=new WSIdentity(DelegatedIdentity.FromCore(result,JsonOptions));identity.Validate(_options.PluginId,DateTimeOffset.UtcNow);return identity;
    }
    private async Task<JsonElement> InvokeIdentity(string operation,object body,CancellationToken ct)
    {
        if (string.IsNullOrEmpty(_options.PluginId)) throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
        ServiceCredential credential;
        try { credential = _options.Credentials != null ? (await _options.Credentials.GetCredentialAsync(ct)).Validate()
            : new ServiceCredential(_options.Credential, _options.AuthScheme).Validate(); }
        catch (FrameworkAdapterException error) { throw new IAMAdapterException(error.Code == FrameworkRuntimeErrors.InvalidCredential
            ? "IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID" : "IAM_CONTEXT_UNAVAILABLE"); }
        catch (HttpRequestException) { throw new IAMAdapterException("IAM_CONTEXT_UNAVAILABLE"); }
        using var request = new HttpRequestMessage(HttpMethod.Post, NormalizeBaseUrl(_options.BaseUrl) + "/tenant/invocations");
        request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        request.Content = new StringContent(JsonSerializer.Serialize(new {
            capability_id = "com.corex.iam.identity_context.service_read", preferred_protocol = "core_internal",
            payload = new { body = new { operation, request = body } }
        }), Encoding.UTF8, "application/json");
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(ct);
        timeout.CancelAfter(_options.Timeout > TimeSpan.Zero ? _options.Timeout : TimeSpan.FromSeconds(3));
        try
        {
            using var response = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, timeout.Token);
            var raw = await response.Content.ReadAsStringAsync(timeout.Token);
            if ((int)response.StatusCode != 200) throw IdentityError(response, raw);
            using var document = JsonDocument.Parse(raw);
            var root = document.RootElement;
            if (root.GetProperty("code").GetInt32() != 200) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
            return root.GetProperty("data").GetProperty("result").GetProperty("result").Clone();
        }
        catch (IAMAdapterException) { throw; }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new IAMAdapterException("IAM_CONTEXT_UNAVAILABLE"); }
        catch (HttpRequestException) { throw new IAMAdapterException("IAM_CONTEXT_UNAVAILABLE"); }
        catch (Exception error) when (error is JsonException or KeyNotFoundException or InvalidOperationException or FormatException)
        { throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID"); }
    }

    private static IdentityContextException IdentityError(HttpResponseMessage response, string raw)
    {
        string? code = null, reason = null, trace = Header("X-Trace-ID"), request = Header("X-Request-ID"), sourceTrace = null, sourceRequest = null;
        string? Header(string name) => response.Headers.TryGetValues(name, out var values) ? values.FirstOrDefault() : null;
        static string? Value(JsonElement node, string field) => node.ValueKind == JsonValueKind.Object && node.TryGetProperty(field, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
        try {
            using var document = JsonDocument.Parse(raw); var root = document.RootElement;
            code = Value(root, "error_code") ?? Value(root, "code"); reason = Value(root, "reason_code");
            if (root.TryGetProperty("error", out var error) && error.ValueKind == JsonValueKind.Object) { code = Value(error, "code") ?? code; reason = Value(error, "reason_code") ?? reason; }
            if (root.TryGetProperty("details", out var details)) { code = Value(details, "code") ?? code; reason = Value(details, "reason_code") ?? reason; }
            trace ??= Value(root, "trace_id"); request ??= Value(root, "request_id"); sourceTrace = Value(root,"source_trace_id"); sourceRequest = Value(root,"source_request_id");
        } catch (JsonException) { }
        return new((int)response.StatusCode, code ?? reason ?? "IAM_CONTEXT_UNAVAILABLE", reason, trace, request, sourceTrace, sourceRequest);
    }
}
