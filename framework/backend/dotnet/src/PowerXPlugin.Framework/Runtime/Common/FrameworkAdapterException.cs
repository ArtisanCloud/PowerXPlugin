namespace PowerXPlugin.Framework.Runtime.Common;

public static class FrameworkRuntimeErrors
{
    public const string InvalidProviderMode = "FRAMEWORK_INVALID_PROVIDER_MODE";
    public const string ProviderModeConflict = "FRAMEWORK_PROVIDER_MODE_CONFLICT";
    public const string AdapterUnavailable = "FRAMEWORK_ADAPTER_UNAVAILABLE";
    public const string InvalidCredential = "FRAMEWORK_INVALID_SERVICE_CREDENTIAL";
}

/// <summary>
/// Stable Framework-level failure. Module adapters may expose their own
/// operation errors, but bootstrap and adapter selection errors stay uniform.
/// </summary>
public sealed class FrameworkAdapterException : InvalidOperationException
{
    public FrameworkAdapterException(string code, string module, string operation, string? message = null, Exception? innerException = null)
        : base(message ?? code, innerException)
    {
        Code = code;
        Module = module;
        Operation = operation;
    }

    public string Code { get; }
    public string Module { get; }
    public string Operation { get; }
}

