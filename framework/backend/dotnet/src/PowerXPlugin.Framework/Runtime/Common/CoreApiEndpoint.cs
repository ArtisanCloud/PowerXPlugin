namespace PowerXPlugin.Framework.Runtime.Common;

/// <summary>Gateway base URL and Core API prefix are separate configuration values.</summary>
public static class CoreApiEndpoint
{
    public const string DefaultAPIPrefix = "/api/v1";

    public static string BaseUrl(string baseUrl, string? apiPrefix)
    {
        var gateway = baseUrl.Trim().TrimEnd('/');
        if (!Uri.TryCreate(gateway, UriKind.Absolute, out var uri)
            || (uri.Scheme != Uri.UriSchemeHttp && uri.Scheme != Uri.UriSchemeHttps)
            || string.IsNullOrEmpty(uri.Host) || gateway.Contains('?') || gateway.Contains('#')
            || !string.IsNullOrEmpty(uri.UserInfo))
            throw new ArgumentException("FRAMEWORK_HOST_API_BASE_INVALID", nameof(baseUrl));

        var prefix = string.IsNullOrWhiteSpace(apiPrefix) ? DefaultAPIPrefix : apiPrefix.Trim();
        prefix = prefix.Trim('/');
        if (prefix.Length == 0) return gateway;
        foreach (var part in prefix.Split('/'))
        {
            if (part.Length == 0 || part is "." or ".."
                || part.Any(c => !(c is >= 'a' and <= 'z' or >= 'A' and <= 'Z' or >= '0' and <= '9' or '-' or '_' or '.' or '~')))
                throw new ArgumentException("FRAMEWORK_HOST_API_PREFIX_INVALID", nameof(apiPrefix));
        }
        return gateway + "/" + prefix;
    }
}
