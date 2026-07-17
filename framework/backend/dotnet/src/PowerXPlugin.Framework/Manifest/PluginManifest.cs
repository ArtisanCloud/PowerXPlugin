namespace PowerXPlugin.Framework.Manifest;

public record PluginManifest
{
    public string ID { get; init; } = "";
    public string Name { get; init; } = "";
    public string Version { get; init; } = "";
    public string? Description { get; init; }
    public List<PluginMenu> Menus { get; init; } = new();
    public List<string> Permissions { get; init; } = new();
}

public record PluginMenu
{
    public string Path { get; init; } = "";
    public string Title { get; init; } = "";
    public string? Icon { get; init; }
    public int Order { get; init; }
    public List<PluginMenu> Children { get; init; } = new();
}

public static class ManifestRegistration
{
    public static void Register(Bootstrap.PluginApp app, PluginManifest manifest)
    {
        if (string.IsNullOrWhiteSpace(manifest.ID))
            throw new ArgumentException("Plugin manifest must have an ID");
        if (string.IsNullOrWhiteSpace(manifest.Version))
            throw new ArgumentException("Plugin manifest must have a version");

        app.RegisterManifest(manifest);
    }
}
