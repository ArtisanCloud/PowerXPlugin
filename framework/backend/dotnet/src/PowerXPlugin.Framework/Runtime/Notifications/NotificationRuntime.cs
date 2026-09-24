using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Notifications;

public sealed record NotificationScope(string TenantUuid);
public sealed record CreateNotificationRequest(string Title, string Content, string Type = "", string Category = "", bool IsImportant = false, string? MemberUuid = null, IReadOnlyDictionary<string, JsonElement>? Metadata = null, string? IdempotencyKey = null);
public sealed record Notification(string Uuid, string Title, string Content, string Type, string Category, bool IsRead, bool IsImportant, DateTimeOffset CreatedAt, DateTimeOffset UpdatedAt, string? MemberUuid, IReadOnlyDictionary<string, JsonElement>? Metadata = null);

public interface INotificationPublisher { Task<Notification> CreateAsync(NotificationScope scope, CreateNotificationRequest request, CancellationToken ct = default); }
/// <summary>Local outbox must atomically enforce tenant plus idempotency-key uniqueness and target-member authorization.</summary>
public interface ILocalNotificationOutbox : INotificationPublisher { }
public sealed class LocalNotificationPublisher(ILocalNotificationOutbox outbox) : INotificationPublisher
{
    private readonly ILocalNotificationOutbox _outbox = outbox ?? throw new ArgumentNullException(nameof(outbox));
    public async Task<Notification> CreateAsync(NotificationScope scope, CreateNotificationRequest request, CancellationToken ct = default)
    {
        ValidateScope(scope); Validate(request);
        var notification = await _outbox.CreateAsync(scope, request, ct);
        if (notification is null || string.IsNullOrWhiteSpace(notification.Uuid) || notification.Title != request.Title.Trim() || notification.Content != request.Content.Trim() || !string.Equals(notification.MemberUuid, request.MemberUuid?.Trim(), StringComparison.Ordinal)) throw new NotificationException(NotificationErrors.InvalidResponse);
        return notification;
    }
    internal static void ValidateScope(NotificationScope scope) { if (!Guid.TryParse(scope.TenantUuid, out var id) || id == Guid.Empty || id.ToString() != scope.TenantUuid) throw new NotificationException(NotificationErrors.InvalidArgument); }
    internal static void Validate(CreateNotificationRequest request)
    {
        if (string.IsNullOrWhiteSpace(request.Title) || string.IsNullOrWhiteSpace(request.Content) || request.Title != request.Title.Trim() || request.Content != request.Content.Trim() || request.Title.Length > 512 || request.Content.Length > 32768 || request.IdempotencyKey?.Length > 256 || request.MemberUuid is not null && (!Guid.TryParse(request.MemberUuid, out var member) || member == Guid.Empty || member.ToString() != request.MemberUuid)) throw new NotificationException(NotificationErrors.InvalidArgument);
    }
}

/// <summary>Typed direct client for the currently published Core notification creation contract.</summary>
public sealed class PowerXNotificationClient(string baseUrl, string tenantUuid, IServiceCredentialProvider credentials, HttpClient? http = null) : INotificationPublisher
{
    private readonly string _base = string.IsNullOrWhiteSpace(baseUrl) ? throw new NotificationException(NotificationErrors.Unavailable) : baseUrl.TrimEnd('/'); private readonly string _tenant = tenantUuid; private readonly IServiceCredentialProvider _credentials = credentials ?? throw new ArgumentNullException(nameof(credentials)); private readonly HttpClient _http = http ?? new();
    public async Task<Notification> CreateAsync(NotificationScope scope, CreateNotificationRequest request, CancellationToken ct = default)
    {
        LocalNotificationPublisher.ValidateScope(scope); LocalNotificationPublisher.Validate(request);
        if (!string.Equals(scope.TenantUuid, _tenant, StringComparison.Ordinal)) throw new NotificationException(NotificationErrors.TenantMismatch);
        // Core's published notifications endpoint has no idempotency field/header. Do not silently drop it.
        if (!string.IsNullOrWhiteSpace(request.IdempotencyKey)) throw new NotificationException(NotificationErrors.IdempotencyUnsupported);
        var credential = (await _credentials.GetCredentialAsync(ct)).Validate();
        using var message = new HttpRequestMessage(HttpMethod.Post, _base + "/api/v1/notifications");
        message.Headers.Authorization = new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value);
        message.Content = new StringContent(JsonSerializer.Serialize(new { title = request.Title, content = request.Content, type = request.Type, category = request.Category, is_important = request.IsImportant, member_uuid = request.MemberUuid, metadata = request.Metadata }), Encoding.UTF8, "application/json");
        try { using var response = await _http.SendAsync(message, ct); var raw = await response.Content.ReadAsStringAsync(ct); if (!response.IsSuccessStatusCode) throw new NotificationException(response.StatusCode switch { HttpStatusCode.Unauthorized => NotificationErrors.Unauthorized, HttpStatusCode.Forbidden => NotificationErrors.Forbidden, HttpStatusCode.BadRequest => NotificationErrors.InvalidArgument, _ => NotificationErrors.Upstream }); using var json = JsonDocument.Parse(raw); if (!json.RootElement.TryGetProperty("data", out var data)) throw new NotificationException(NotificationErrors.InvalidResponse); var result = JsonSerializer.Deserialize<HostNotification>(data.GetRawText(), new JsonSerializerOptions(JsonSerializerDefaults.Web)) ?? throw new NotificationException(NotificationErrors.InvalidResponse); if (string.IsNullOrWhiteSpace(result.Id)) throw new NotificationException(NotificationErrors.InvalidResponse); return new Notification(result.Id, result.Title ?? string.Empty, result.Content ?? string.Empty, result.Type ?? string.Empty, result.Category ?? string.Empty, result.IsRead, result.IsImportant, result.CreatedAt, result.UpdatedAt, result.UserId, result.Metadata); } catch (NotificationException) { throw; } catch (HttpRequestException e) { throw new NotificationException(NotificationErrors.Upstream, e); } catch (JsonException e) { throw new NotificationException(NotificationErrors.InvalidResponse, e); }
    }
    private sealed class HostNotification { public string Id { get; init; } = ""; public string? Title { get; init; } public string? Content { get; init; } public string? Type { get; init; } public string? Category { get; init; } [JsonPropertyName("isRead")] public bool IsRead { get; init; } [JsonPropertyName("isImportant")] public bool IsImportant { get; init; } [JsonPropertyName("createdAt")] public DateTimeOffset CreatedAt { get; init; } [JsonPropertyName("updatedAt")] public DateTimeOffset UpdatedAt { get; init; } [JsonPropertyName("userId")] public string? UserId { get; init; } public IReadOnlyDictionary<string, JsonElement>? Metadata { get; init; } }
}
public static class NotificationRuntimeExtensions { public static IServiceCollection AddPowerXNotificationRuntime(this IServiceCollection s, ProviderMode m, Func<IServiceProvider, INotificationPublisher> l, Func<IServiceProvider, INotificationPublisher> d) => s.AddPowerXRuntime("notifications", m, l, d); }
public static class NotificationErrors { public const string InvalidArgument="FRAMEWORK_NOTIFICATION_INVALID_ARGUMENT", InvalidResponse="FRAMEWORK_NOTIFICATION_INVALID_RESPONSE", TenantMismatch="FRAMEWORK_NOTIFICATION_TENANT_MISMATCH", IdempotencyUnsupported="FRAMEWORK_NOTIFICATION_IDEMPOTENCY_UNSUPPORTED", Unavailable="FRAMEWORK_NOTIFICATION_UNAVAILABLE", Unauthorized="FRAMEWORK_NOTIFICATION_UNAUTHORIZED", Forbidden="FRAMEWORK_NOTIFICATION_FORBIDDEN", Upstream="FRAMEWORK_NOTIFICATION_UPSTREAM_DEPENDENCY"; }
public sealed class NotificationException(string code, Exception? inner = null) : InvalidOperationException(code, inner) { public string Code { get; } = code; }
