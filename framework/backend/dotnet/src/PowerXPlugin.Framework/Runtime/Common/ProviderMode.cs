namespace PowerXPlugin.Framework.Runtime.Common;

/// <summary>
/// Trusted process-wide provider selection. Applications resolve this once
/// during bootstrap; request inputs must never select a provider.
/// </summary>
public enum ProviderMode
{
    Local,
    Delegated
}

public static class ProviderModeParser
{
    public static ProviderMode Parse(string? value)
    {
        return value?.Trim().ToLowerInvariant() switch
        {
            "local" => ProviderMode.Local,
            "delegated" => ProviderMode.Delegated,
            _ => throw new FrameworkAdapterException(
                FrameworkRuntimeErrors.InvalidProviderMode,
                "runtime",
                "bootstrap",
                "Provider mode must be local or delegated.")
        };
    }

    /// <summary>
    /// Resolves two bootstrap sources only when they agree. Environment access
    /// belongs to the application bootstrap, not to an individual Runtime.
    /// </summary>
    public static ProviderMode Resolve(string? configuredValue, string? environmentValue)
    {
        ProviderMode? configured = string.IsNullOrWhiteSpace(configuredValue) ? null : Parse(configuredValue);
        ProviderMode? environment = string.IsNullOrWhiteSpace(environmentValue) ? null : Parse(environmentValue);

        if (configured.HasValue && environment.HasValue && configured.Value != environment.Value)
        {
            throw new FrameworkAdapterException(
                FrameworkRuntimeErrors.ProviderModeConflict,
                "runtime",
                "bootstrap",
                "Configured provider mode conflicts with environment provider mode.");
        }

        return configured ?? environment
            ?? throw new FrameworkAdapterException(
                FrameworkRuntimeErrors.InvalidProviderMode,
                "runtime",
                "bootstrap",
                "Provider mode is required during bootstrap.");
    }
}
