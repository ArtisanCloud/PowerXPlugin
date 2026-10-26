using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.IAM.Delegated;

public sealed class AuthorizationBaselineException(int status, string code, string? reasonCode,
    string? traceId, string? requestId, string? sourceTraceId, string? sourceRequestId)
    : IAMAdapterException(reasonCode ?? code)
{
    public int Status { get; } = status;
    public int HttpStatus => Status == 200 ? 502 : Status;
    public string OriginalCode { get; } = code;
    public string? ReasonCode { get; } = reasonCode;
    public string? TraceId { get; } = traceId;
    public string? RequestId { get; } = requestId;
    public string? SourceTraceId { get; } = sourceTraceId;
    public string? SourceRequestId { get; } = sourceRequestId;
}

public sealed partial class PowerXIamClient : IAuthorizationBaselineService
{
    public async Task<AuthorizationBaselineResult> ReadMemberBaselines(AuthorizationBaselineQuery query, CancellationToken ct = default)
    {
        var normalized = query.Normalize();
        if (string.IsNullOrWhiteSpace(_options.PluginId)) throw new IAMAdapterException(IAMErrors.CodeAdapterNotBound);
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(ct);
        timeout.CancelAfter(_options.Timeout > TimeSpan.Zero ? _options.Timeout : TimeSpan.FromSeconds(3));
        try
        {
            var credential = _options.Credentials is not null
                ? (await _options.Credentials.GetCredentialAsync(timeout.Token)).Validate()
                : new ServiceCredential(_options.Credential, _options.AuthScheme).Validate();
            using var request = new HttpRequestMessage(HttpMethod.Post, NormalizeBaseUrl(_options.BaseUrl) + "/tenant/invocations");
            request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
            request.Headers.Accept.ParseAdd("application/json");
            request.Content = new StringContent(JsonSerializer.Serialize(new {
                capability_id = AuthorizationBaselineContract.CapabilityId, preferred_protocol = "core_internal",
                payload = new { body = new { operation = AuthorizationBaselineContract.Operation, request = new {
                    schema = AuthorizationBaselineContract.QuerySchema, member_uuids = normalized.MemberUuids,
                    permission_codes = normalized.PermissionCodes
                } } }
            }), Encoding.UTF8, "application/json");
            using var response = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, timeout.Token);
            var bytes = new byte[(1 << 20) + 1];
            await using var stream = await response.Content.ReadAsStreamAsync(timeout.Token);
            var length = 0;
            while (length < bytes.Length)
            {
                var read = await stream.ReadAsync(bytes.AsMemory(length), timeout.Token);
                if (read == 0) break;
                length += read;
            }
            timeout.Token.ThrowIfCancellationRequested();
            if (length > 1 << 20) throw BaselineError(response, "", true);
            var raw = Encoding.UTF8.GetString(bytes, 0, length);
            if ((int)response.StatusCode != 200) throw BaselineError(response, raw, false);
            try
            {
                using var document = JsonDocument.Parse(raw);
                var root = document.RootElement;
                if (root.GetProperty("code").GetInt32() != 200) throw AuthorizationBaselineContract.Invalid();
                var result = AuthorizationBaselineContract.Decode(root.GetProperty("data").GetProperty("result").GetProperty("result"), normalized, _options.PluginId);
                timeout.Token.ThrowIfCancellationRequested();
                return result;
            }
            catch (Exception error) when (error is JsonException or KeyNotFoundException or InvalidOperationException or FormatException
                || error is IAMAdapterException { Code: "IAM_BASELINE_RESPONSE_INVALID" })
            { throw BaselineError(response, raw, true); }
        }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested)
        { throw new IAMAdapterException("IAM_BASELINE_DEPENDENCY_UNAVAILABLE"); }
        catch (HttpRequestException) { throw new IAMAdapterException("IAM_BASELINE_DEPENDENCY_UNAVAILABLE"); }
        catch (FrameworkAdapterException error)
        { throw new IAMAdapterException(error.Code == FrameworkRuntimeErrors.InvalidCredential ? "IAM_CONTEXT_SERVICE_CREDENTIAL_INVALID" : "IAM_BASELINE_DEPENDENCY_UNAVAILABLE"); }
    }

    private static AuthorizationBaselineException BaselineError(HttpResponseMessage response, string raw, bool invalid)
    {
        var diagnostic = IdentityError(response, raw);
        var code = invalid ? "IAM_BASELINE_RESPONSE_INVALID" : diagnostic.OriginalCode == "IAM_CONTEXT_UNAVAILABLE"
            ? "IAM_BASELINE_DEPENDENCY_UNAVAILABLE" : diagnostic.OriginalCode;
        return new((int)response.StatusCode, code, invalid || string.IsNullOrEmpty(diagnostic.ReasonCode) ? null : diagnostic.ReasonCode,
            diagnostic.TraceId, diagnostic.RequestId, diagnostic.SourceTraceId, diagnostic.SourceRequestId);
    }
}
