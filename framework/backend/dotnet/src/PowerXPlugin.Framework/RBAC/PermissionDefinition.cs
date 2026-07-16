namespace PowerXPlugin.Framework.RBAC;

public record PermissionDefinition
{
    public string Key { get; init; } = "";
    public string Scope { get; init; } = "";
    public string Description { get; init; } = "";
}

public interface IPermissionReporter
{
    Task RegisterPermissionsAsync(IReadOnlyList<PermissionDefinition> permissions, CancellationToken ct = default);
}

public static class PermissionValidator
{
    private static readonly HashSet<string> ValidScopes = new(StringComparer.OrdinalIgnoreCase)
    {
        "global", "tenant", "plugin", "user"
    };

    public static List<string> Validate(IReadOnlyList<PermissionDefinition> permissions)
    {
        var errors = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        for (var i = 0; i < permissions.Count; i++)
        {
            var p = permissions[i];
            if (string.IsNullOrWhiteSpace(p.Key))
                errors.Add($"permission[{i}]: key is required");
            else if (!seen.Add(p.Key))
                errors.Add($"permission[{i}]: duplicate key '{p.Key}'");

            if (string.IsNullOrWhiteSpace(p.Scope))
                errors.Add($"permission[{i}]: scope is required");
            else if (!ValidScopes.Contains(p.Scope))
                errors.Add($"permission[{i}]: invalid scope '{p.Scope}'");

            if (string.IsNullOrWhiteSpace(p.Description))
                errors.Add($"permission[{i}]: description is required");
        }

        return errors;
    }

    public static async Task<List<string>> ReportAsync(
        IPermissionReporter reporter,
        IReadOnlyList<PermissionDefinition> permissions,
        CancellationToken ct = default)
    {
        var errors = Validate(permissions);
        if (errors.Count > 0) return errors;

        await reporter.RegisterPermissionsAsync(permissions, ct);
        return errors;
    }
}
