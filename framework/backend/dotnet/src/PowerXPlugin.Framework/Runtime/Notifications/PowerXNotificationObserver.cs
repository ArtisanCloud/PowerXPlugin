using System.Net;
using System.Net.Http.Headers;
using System.Net.WebSockets;
using System.Runtime.CompilerServices;
using System.Text.Json;
using System.Text.Json.Serialization;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.WSBus;

namespace PowerXPlugin.Framework.Runtime.Notifications;

public sealed record PowerXNotificationView(
    [property: JsonPropertyName("id")] string Id,
    [property: JsonPropertyName("title")] string Title,
    [property: JsonPropertyName("content")] string Content,
    [property: JsonPropertyName("type")] string Type,
    [property: JsonPropertyName("createdAt")] DateTimeOffset CreatedAt,
    [property: JsonPropertyName("relatedId")] string? RelatedId,
    [property: JsonPropertyName("relatedType")] string? RelatedType,
    [property: JsonPropertyName("metadata")] Dictionary<string, JsonElement>? Metadata);

public interface IHostNotificationSocketConnector
{
    Task<WebSocket> ConnectAsync(Uri endpoint, ServiceCredential credential, CancellationToken ct);
}
public sealed class HostNotificationSocketConnector : IHostNotificationSocketConnector
{
    public async Task<WebSocket> ConnectAsync(Uri endpoint, ServiceCredential credential, CancellationToken ct)
    {
        var socket = new ClientWebSocket();
        socket.Options.SetRequestHeader("Authorization", new AuthenticationHeaderValue(credential.NormalizedAuthScheme, credential.Value).ToString());
        try { await socket.ConnectAsync(endpoint, ct); return socket; }
        catch { socket.Dispose(); throw; }
    }
}

/// <summary>Reads and observes existing Core notifications using service credentials. Does not create local echoes.</summary>
public sealed class PowerXNotificationObserver(string baseUrl, string apiPrefix, IServiceCredentialProvider credentials, HttpClient http, IHostNotificationSocketConnector? connector = null)
{
    public const string Topic = "_topic.system.notification";
    private readonly string _base = baseUrl.TrimEnd('/');
    private readonly string _prefix = "/" + apiPrefix.Trim('/');
    private readonly IHostNotificationSocketConnector _connector = connector ?? new HostNotificationSocketConnector();
    private static readonly JsonSerializerOptions Json = new(JsonSerializerDefaults.Web);

    public async Task<IReadOnlyList<PowerXNotificationView>> ListSchedulerNotificationsAsync(string jobId, string ownerId, CancellationToken ct = default)
    {
        var result = new List<PowerXNotificationView>(); var seen = 0; var page = 1;
        var credential = (await credentials.GetCredentialAsync(ct)).Validate();
        try
        {
            while (true)
            {
                using var request = new HttpRequestMessage(HttpMethod.Get, _base + _prefix + $"/admin/notifications?page={page}&page_size=100");
                request.Headers.Authorization = new(credential.NormalizedAuthScheme, credential.Value);
                using var response = await http.SendAsync(request, ct);
                if (!response.IsSuccessStatusCode) throw Failure(response.StatusCode);
                using var document = JsonDocument.Parse(await response.Content.ReadAsStringAsync(ct));
                var root = document.RootElement;
                if (!root.TryGetProperty("code", out var code) || code.ValueKind != JsonValueKind.Number || code.GetInt32() != 200
                    || !root.TryGetProperty("data", out var data) || data.ValueKind != JsonValueKind.Object || !data.TryGetProperty("items", out var items) || items.ValueKind != JsonValueKind.Array
                    || !data.TryGetProperty("pagination", out var pagination) || pagination.ValueKind != JsonValueKind.Object || !pagination.TryGetProperty("total", out var totalValue)
                    || totalValue.ValueKind != JsonValueKind.Number || !totalValue.TryGetInt32(out var total) || total < 0)
                    throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                foreach (var item in items.EnumerateArray())
                {
                    var value = item.Deserialize<PowerXNotificationView>(Json) ?? throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                    if (Matches(value, jobId, ownerId)) result.Add(value);
                    seen++;
                }
                if (seen >= total) return result;
                if (items.GetArrayLength() == 0) throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                page++;
            }
        }
        catch (HttpRequestException) { throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_DEPENDENCY"); }
        catch (JsonException) { throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE"); }
        catch (OperationCanceledException) when (!ct.IsCancellationRequested) { throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_DEPENDENCY"); }
    }

    public async IAsyncEnumerable<JsonElement> ObserveSchedulerAsync(string jobId, string ownerId, [EnumeratorCancellation] CancellationToken ct = default)
    {
        var credential = (await credentials.GetCredentialAsync(ct)).Validate();
        var endpoint = new UriBuilder(new Uri(_base)) { Scheme = _base.StartsWith("https:", StringComparison.Ordinal) ? "wss" : "ws", Path = "/api/ws", Query = "" }.Uri;
        WebSocket socket;
        try { socket = await _connector.ConnectAsync(endpoint, credential, ct); }
        catch (WebSocketException) { throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_DEPENDENCY"); }
        using (socket)
        {
            var welcome = await Receive(socket, ct);
            if (Type(welcome) != "welcome") throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
            yield return welcome;
            await socket.SendAsync(JsonSerializer.SerializeToUtf8Bytes(new { type = "subscribe", topics = new[] { Topic }, req_id = Guid.NewGuid().ToString("N") }), WebSocketMessageType.Text, true, ct);
            var acknowledged = false;
            while (!ct.IsCancellationRequested)
            {
                var envelope = await Receive(socket, ct);
                if (Type(envelope) == "error") throw new WSBusException(403, "FRAMEWORK_WSBUS_SUBSCRIPTION_REJECTED");
                if (Type(envelope) == "ack")
                {
                    if (!envelope.TryGetProperty("payload", out var ack) || !ack.TryGetProperty("ok", out var ok) || ok.ValueKind != JsonValueKind.True
                        || !ack.TryGetProperty("topics", out var topics) || topics.ValueKind != JsonValueKind.Array || !topics.EnumerateArray().Any(topic => topic.GetString() == Topic))
                        throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                    acknowledged = true; yield return envelope; continue;
                }
                if (Type(envelope) != "event" || !acknowledged || !envelope.TryGetProperty("topic", out var topicName) || topicName.GetString() != Topic) continue;
                if (!envelope.TryGetProperty("payload", out var payload)) throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                var notification = payload.Deserialize<PowerXNotificationView>(Json) ?? throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
                if (Matches(notification, jobId, ownerId)) yield return envelope;
            }
        }
    }

    public static bool Matches(PowerXNotificationView value, string jobId, string ownerId) => Guid.TryParse(value.Id, out var id) && id != Guid.Empty
        && value.RelatedId == jobId && value.RelatedType == "runtime_scheduler_job" && value.Metadata != null
        && Value(value.Metadata, "job_id") == jobId && Value(value.Metadata, "owner_id") == ownerId
        && Value(value.Metadata, "owner_type") == "plugin" && Value(value.Metadata, "business_action") == "framework_lab_scheduler_probe";
    private static string? Value(Dictionary<string, JsonElement> data, string key) => data.TryGetValue(key, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString() : null;
    private static string? Type(JsonElement value) => value.TryGetProperty("type", out var type) ? type.GetString() : null;
    private static WSBusException Failure(HttpStatusCode status) => new(status is HttpStatusCode.Unauthorized or HttpStatusCode.Forbidden ? (int)status : 424, "FRAMEWORK_WSBUS_UPSTREAM_REJECTED");
    private static async Task<JsonElement> Receive(WebSocket socket, CancellationToken ct)
    {
        var buffer = new byte[8192]; using var message = new MemoryStream();
        while (true)
        {
            var received = await socket.ReceiveAsync(buffer, ct);
            if (received.MessageType == WebSocketMessageType.Close) throw new WSBusException(424, "FRAMEWORK_WSBUS_UPSTREAM_CLOSED");
            if (received.MessageType != WebSocketMessageType.Text || message.Length + received.Count > 1_048_576) throw new WSBusException(424, "FRAMEWORK_WSBUS_INVALID_RESPONSE");
            message.Write(buffer, 0, received.Count);
            if (received.EndOfMessage) { using var json = JsonDocument.Parse(message.ToArray()); return json.RootElement.Clone(); }
        }
    }
}
