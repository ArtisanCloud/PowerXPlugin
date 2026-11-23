using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.Notifications;

public sealed record NotificationCreationLookupRequest(string IdempotencyKey, string MemberUuid);
public sealed record NotificationCreationReceipt(string Contract, string TenantUuid, string ServiceOwner, string IdempotencyKey, string MemberUuid,
    bool Found, string NotificationState, DateTimeOffset CheckedAt, Notification? Notification);
public interface INotificationCreationReceiptReader
{
    Task<NotificationCreationReceipt> LookupAsync(NotificationScope scope, NotificationCreationLookupRequest request, CancellationToken ct = default);
}
public interface ILocalNotificationCreationReceiptReader : INotificationCreationReceiptReader { }
public static class NotificationCreationReceiptErrors
{
    public const string Unsupported = "FRAMEWORK_NOTIFICATION_CREATION_LOOKUP_UNSUPPORTED";
}

public static class NotificationCreationReceiptValidation
{
    private static bool Uuid(string? value) => Guid.TryParseExact(value, "D", out var id) && id != Guid.Empty && id.ToString() == value;
    public static void Request(NotificationScope scope, NotificationCreationLookupRequest request)
    {
        LocalNotificationPublisher.ValidateScope(scope);
        if (request == null || !Uuid(request.MemberUuid) || request.IdempotencyKey == null || request.IdempotencyKey.Length is < 1 or > 256 || request.IdempotencyKey.Any(c => c < 33 || c > 126)) throw new NotificationException(NotificationErrors.InvalidArgument);
    }
    public static NotificationCreationReceipt Result(NotificationScope scope, NotificationCreationLookupRequest request, NotificationCreationReceipt value)
    {
        if (value == null || value.Contract != "notification_create_receipt.v1" || value.TenantUuid != scope.TenantUuid || value.IdempotencyKey != request.IdempotencyKey || value.MemberUuid != request.MemberUuid
            || string.IsNullOrWhiteSpace(value.ServiceOwner) || value.ServiceOwner.Any(char.IsWhiteSpace) || value.ServiceOwner.Any(char.IsControl)
            || !(value.ServiceOwner.StartsWith("plugin:") && value.ServiceOwner.Length > 7 || value.ServiceOwner.StartsWith("api_key:") && value.ServiceOwner.Length > 8 || value.ServiceOwner == "local")
            || value.CheckedAt <= DateTimeOffset.UnixEpoch || value.CheckedAt.Offset != TimeSpan.Zero) throw new NotificationException(NotificationErrors.InvalidResponse);
        if (!value.Found)
        { if (value.Notification != null || value.NotificationState != "not_found") throw new NotificationException(NotificationErrors.InvalidResponse); }
        else if (value.NotificationState is not ("present" or "deleted" or "receipt_only") || value.Notification == null || !Uuid(value.Notification.Uuid) || value.Notification.MemberUuid != request.MemberUuid
            || string.IsNullOrWhiteSpace(value.Notification.Title) || string.IsNullOrWhiteSpace(value.Notification.Content) || value.Notification.CreatedAt <= DateTimeOffset.UnixEpoch || value.Notification.UpdatedAt < value.Notification.CreatedAt)
            throw new NotificationException(NotificationErrors.InvalidResponse);
        return value;
    }
}

public sealed class LocalNotificationCreationReceiptReader(ILocalNotificationCreationReceiptReader local) : INotificationCreationReceiptReader
{
    public async Task<NotificationCreationReceipt> LookupAsync(NotificationScope scope, NotificationCreationLookupRequest request, CancellationToken ct = default)
    {
        NotificationCreationReceiptValidation.Request(scope, request);
        return NotificationCreationReceiptValidation.Result(scope, request, await local.LookupAsync(scope, request, ct));
    }
}

public sealed partial class PowerXNotificationClient : INotificationCreationReceiptReader
{
    // 独立只读接口：旧Host、缺授权或格式不明均明确失败，不调用创建/已读接口兜底。
    public async Task<NotificationCreationReceipt> LookupAsync(NotificationScope scope, NotificationCreationLookupRequest request, CancellationToken ct = default)
    {
        NotificationCreationReceiptValidation.Request(scope, request);
        if (scope.TenantUuid != _tenant) throw new NotificationException(NotificationErrors.TenantMismatch);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        using var message = new HttpRequestMessage(HttpMethod.Post, _base + "/notifications/creation-receipts/lookup");
        message.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        message.Content = new StringContent(JsonSerializer.Serialize(new { idempotency_key = request.IdempotencyKey, member_uuid = request.MemberUuid }), Encoding.UTF8, "application/json");
        try
        {
            using var response = await _http.SendAsync(message, ct);
            if (!response.IsSuccessStatusCode) throw new NotificationException(response.StatusCode switch {
                HttpStatusCode.NotFound or HttpStatusCode.MethodNotAllowed or HttpStatusCode.NotImplemented => NotificationCreationReceiptErrors.Unsupported,
                HttpStatusCode.Unauthorized => NotificationErrors.Unauthorized, HttpStatusCode.Forbidden => NotificationErrors.Forbidden,
                HttpStatusCode.BadRequest => NotificationErrors.InvalidArgument, _ => NotificationErrors.Upstream });
            using var json = JsonDocument.Parse(await response.Content.ReadAsStringAsync(ct));
            if (!json.RootElement.TryGetProperty("data", out var data) || data.ValueKind != JsonValueKind.Object || !data.TryGetProperty("found", out var found) || found.ValueKind is not (JsonValueKind.True or JsonValueKind.False)) throw new NotificationException(NotificationErrors.InvalidResponse);
            Notification? original = null;
            if (data.TryGetProperty("notification", out var item) && item.ValueKind != JsonValueKind.Null)
            {
                foreach (var field in new[] { "id", "title", "content", "type", "category", "isRead", "isImportant", "createdAt", "updatedAt", "userId" })
                    if (!item.TryGetProperty(field, out _)) throw new NotificationException(NotificationErrors.InvalidResponse);
                var host = JsonSerializer.Deserialize<HostNotification>(item.GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new NotificationException(NotificationErrors.InvalidResponse);
                original = new(host.Id, host.Title!, host.Content!, host.Type!, host.Category!, host.IsRead, host.IsImportant, host.CreatedAt, host.UpdatedAt, host.UserId, host.Metadata, host.Actions);
            }
            var result = new NotificationCreationReceipt(data.GetProperty("contract").GetString()!, data.GetProperty("tenant_uuid").GetString()!, data.GetProperty("service_owner").GetString()!,
                data.GetProperty("idempotency_key").GetString()!, data.GetProperty("member_uuid").GetString()!, found.GetBoolean(), data.GetProperty("notification_state").GetString()!, data.GetProperty("checked_at").GetDateTimeOffset(), original);
            if (result.ServiceOwner == "local") throw new NotificationException(NotificationErrors.InvalidResponse);
            return NotificationCreationReceiptValidation.Result(scope, request, result);
        }
        catch (NotificationException) { throw; }
        catch (HttpRequestException e) { throw new NotificationException(NotificationErrors.Upstream, e); }
        catch (Exception e) when (e is JsonException or KeyNotFoundException or InvalidOperationException or FormatException) { throw new NotificationException(NotificationErrors.InvalidResponse, e); }
    }
}
