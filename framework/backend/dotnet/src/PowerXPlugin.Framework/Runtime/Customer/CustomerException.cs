using System.Net;
using System.Text.Json;

namespace PowerXPlugin.Framework.Runtime.Customer;

public static partial class CustomerErrors
{
    public const string ClientAppNotFound = "CUSTOMER_CLIENT_APP_NOT_FOUND";
    public const string ClientAppForbidden = "CUSTOMER_CLIENT_APP_FORBIDDEN";
    public const string ClientPlatformUnsupported = "CUSTOMER_CLIENT_PLATFORM_UNSUPPORTED";
    public const string ClientAppNotConfigured = "CUSTOMER_CLIENT_APP_NOT_CONFIGURED";
    public const string PhoneCodeInvalid = "CUSTOMER_PHONE_CODE_INVALID";
    public const string PhoneConflict = "CUSTOMER_PHONE_CONFLICT";
}

public sealed class CustomerException : InvalidOperationException
{
    public CustomerException(string code) : this(code, null, null) { }

    public CustomerException(string code, int? statusCode, string? reasonCode) : base(code)
    {
        Code = code;
        StatusCode = statusCode;
        ReasonCode = reasonCode;
    }

    public string Code { get; }
    public int? StatusCode { get; }
    public string? ReasonCode { get; }

    internal static CustomerException FromCore(HttpStatusCode status, string raw)
    {
        string reason;
        try
        {
            using var document = JsonDocument.Parse(raw);
            reason = ReadReason(document.RootElement);
        }
        catch (JsonException)
        {
            return new(CustomerErrors.InvalidResponse, (int)status, null);
        }

        if (string.IsNullOrWhiteSpace(reason))
            return new(CustomerErrors.InvalidResponse, (int)status, null);

        var code = reason switch
        {
            CustomerErrors.ClientAppNotFound or CustomerErrors.ClientAppForbidden or
            CustomerErrors.ClientPlatformUnsupported or CustomerErrors.ClientAppNotConfigured or
            CustomerErrors.PhoneCodeInvalid or CustomerErrors.PhoneConflict => reason,
            "CUSTOMER_INVALID_ARGUMENT" => CustomerErrors.InvalidArgument,
            "CUSTOMER_CREDENTIAL_INVALID" => CustomerErrors.InvalidCredential,
            "CUSTOMER_UNAUTHORIZED" => CustomerErrors.Unauthorized,
            "CUSTOMER_FORBIDDEN" => CustomerErrors.Forbidden,
            _ => status switch
            {
                HttpStatusCode.Forbidden => CustomerErrors.Forbidden,
                HttpStatusCode.Unauthorized => CustomerErrors.Unauthorized,
                _ => CustomerErrors.Upstream
            }
        };
        return new(code, (int)status, reason);
    }

    private static string ReadReason(JsonElement root)
    {
        var error = ObjectProperty(root, "error");
        var details = ObjectProperty(root, "details");
        // Same structured Core envelope fields and precedence as Go hostcontract.ParseReasonCode.
        foreach (var value in new[]
        {
            StringProperty(error, "reason_code"), StringProperty(root, "reason_code"),
            StringProperty(details, "reason_code"), StringProperty(error, "error_code"),
            StringProperty(root, "error_code"), StringProperty(details, "error_code"),
            StringProperty(error, "code"), StringProperty(details, "code")
        })
        {
            if (!string.IsNullOrWhiteSpace(value)) return value.Trim();
        }
        return "";
    }

    private static JsonElement ObjectProperty(JsonElement value, string name) =>
        value.ValueKind == JsonValueKind.Object && value.TryGetProperty(name, out var property)
            ? property : default;

    private static string StringProperty(JsonElement value, string name) =>
        value.ValueKind == JsonValueKind.Object && value.TryGetProperty(name, out var property) &&
        property.ValueKind == JsonValueKind.String ? property.GetString() ?? "" : "";
}
