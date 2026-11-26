using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.IAM.Contracts;
using PowerXPlugin.Framework.IAM.Models;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.IAM.Delegated;

public sealed class MemberAuthorityException(int status, string code, string? traceId, string? requestId,
    string? originalCode = null, string? reasonCode = null)
    : IAMAdapterException(code)
{
    public int Status { get; } = status;
    public string? OriginalCode { get; } = originalCode;
    public string? ReasonCode { get; } = reasonCode;
    public string? TraceId { get; } = traceId;
    public string? RequestId { get; } = requestId;
}

public sealed partial class PowerXIamClient : IMemberAuthorityDirectory
{
    public async Task<MemberAuthorityBatch> ReadMemberAuthority(string tenantUuid, MemberAuthorityQuery query, CancellationToken ct = default)
    {
        query.Validate();
        if (!MemberAuthorityBatch.Canonical(tenantUuid) || string.IsNullOrWhiteSpace(_options.PluginId)) throw new IAMAdapterException("IAM_ORGANIZATION_ADAPTER_NOT_BOUND");
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(ct);
        timeout.CancelAfter(_options.Timeout > TimeSpan.Zero ? _options.Timeout : TimeSpan.FromSeconds(3));
        try
        {
            var credential = _options.Credentials is not null ? (await _options.Credentials.GetCredentialAsync(timeout.Token)).Validate() : new ServiceCredential(_options.Credential, _options.AuthScheme).Validate();
            using var request = new HttpRequestMessage(HttpMethod.Post, NormalizeBaseUrl(_options.BaseUrl) + "/tenant/invocations");
            request.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
            request.Content = new StringContent(JsonSerializer.Serialize(new {
                capability_id = MemberAuthorityBatch.CapabilityId, preferred_protocol = "core_internal",
                payload = new { body = new { operation = "read_member_authority", request = new { schema = "powerx.iam.member-authority-query.v1", member_uuids = query.MemberUuids } } }
            }), Encoding.UTF8, "application/json");
            using var response = await _http.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, timeout.Token);
            await using var stream = await response.Content.ReadAsStreamAsync(timeout.Token);
            var bytes = new byte[(1 << 20) + 1]; var length = 0;
            while (length < bytes.Length) { var count = await stream.ReadAsync(bytes.AsMemory(length), timeout.Token); if (count == 0) break; length += count; }
            string? Header(string name) => response.Headers.TryGetValues(name, out var values) ? values.FirstOrDefault() : null;
            var raw = Encoding.UTF8.GetString(bytes, 0, length);
            if (!response.IsSuccessStatusCode)
                throw MemberAuthorityError(response, length > 1 << 20 ? null : raw);
            try
            {
                if (length > 1 << 20) throw new JsonException();
                using var json = JsonDocument.Parse(raw); var root = json.RootElement;
                if (root.GetProperty("code").GetInt32() != 200) throw new JsonException();
                var result = root.GetProperty("data").GetProperty("result").GetProperty("result").Deserialize<MemberAuthorityBatch>(JsonOptions) ?? throw new JsonException();
                result.Validate(query, tenantUuid); return result;
            }
            catch (Exception error) when (error is JsonException or KeyNotFoundException or InvalidOperationException or FormatException || error is IAMAdapterException { Code: "IAM_ORGANIZATION_RESPONSE_INVALID" })
            { throw new MemberAuthorityException(502, "IAM_ORGANIZATION_RESPONSE_INVALID", Header("X-Trace-ID"), Header("X-Request-ID")); }
        }
        catch (IAMAdapterException) { throw; }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new IAMAdapterException("IAM_ORGANIZATION_UNAVAILABLE"); }
        catch (HttpRequestException) { throw new IAMAdapterException("IAM_ORGANIZATION_UNAVAILABLE"); }
        catch (FrameworkAdapterException) { throw new IAMAdapterException("IAM_ORGANIZATION_SERVICE_CREDENTIAL_INVALID"); }
    }

    private static MemberAuthorityException MemberAuthorityError(HttpResponseMessage response, string? raw)
    {
        string? Header(string name) => response.Headers.TryGetValues(name, out var values) ? values.FirstOrDefault() : null;
        string? trace = Header("X-Trace-ID"), request = Header("X-Request-ID"), originalCode = null, reasonCode = null;
        var invalid = raw is null;
        if (raw is not null)
        {
            try
            {
                using var document = JsonDocument.Parse(raw);
                var root = document.RootElement;
                if (root.ValueKind != JsonValueKind.Object) throw new JsonException();
                string? Value(JsonElement node, string field)
                {
                    if (!node.TryGetProperty(field, out var value)) return null;
                    if (value.ValueKind == JsonValueKind.String && !string.IsNullOrWhiteSpace(value.GetString())) return value.GetString();
                    invalid = true;
                    return null;
                }
                originalCode = Value(root, "error_code");
                reasonCode = Value(root, "reason_code");
                var bodyRequest = Value(root, "request_id");
                request ??= bodyRequest;
                if (root.TryGetProperty("details", out var details))
                {
                    if (details.ValueKind == JsonValueKind.Object)
                    {
                        var bodyTrace = Value(details, "trace_id");
                        trace ??= bodyTrace;
                    }
                    else invalid = true;
                }
            }
            catch (JsonException) { invalid = true; }
        }
        var code = invalid ? "IAM_ORGANIZATION_RESPONSE_INVALID" : reasonCode ?? originalCode ?? "IAM_ORGANIZATION_RESPONSE_INVALID";
        return new((int)response.StatusCode, code, trace, request, originalCode, reasonCode);
    }
}
