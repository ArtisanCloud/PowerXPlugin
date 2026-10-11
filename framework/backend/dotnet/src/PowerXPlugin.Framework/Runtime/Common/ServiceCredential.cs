namespace PowerXPlugin.Framework.Runtime.Common;

/// <summary>
/// Service-to-host credential. This contract intentionally has no request
/// context or end-user bearer field.
/// </summary>
public sealed record ServiceCredential(string Value, string AuthScheme = "Bearer")
{
    public string NormalizedAuthScheme => AuthScheme.Trim().Equals("apikey", StringComparison.OrdinalIgnoreCase)
        ? "ApiKey"
        : AuthScheme.Trim().Equals("bearer", StringComparison.OrdinalIgnoreCase)
            ? "Bearer"
            : throw new FrameworkAdapterException(
                FrameworkRuntimeErrors.InvalidCredential,
                "runtime",
                "credential",
                "Service credential scheme must be ApiKey or Bearer.");

    public ServiceCredential Validate()
    {
        if (string.IsNullOrWhiteSpace(Value))
        {
            throw new FrameworkAdapterException(
                FrameworkRuntimeErrors.InvalidCredential,
                "runtime",
                "credential",
                "Service credential value is required.");
        }

        _ = NormalizedAuthScheme;
        return this;
    }
}

public interface IServiceCredentialProvider
{
    ValueTask<ServiceCredential> GetCredentialAsync(CancellationToken cancellationToken = default);
}

