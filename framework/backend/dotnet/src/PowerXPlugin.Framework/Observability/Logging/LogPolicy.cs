namespace PowerXPlugin.Framework.Observability.Logging;

public enum PolicyMode { Host, Standalone }

public enum SinkType { Stdout, Stderr, File, Loki }

public enum LogFormat { Json, Text }

public class RetryPolicy
{
    public bool Enabled { get; set; } = true;
    public int MaxAttempts { get; set; } = 3;
    public int BackoffMs { get; set; } = 200;
}

public class LogPolicy
{
    public PolicyMode Mode { get; set; } = PolicyMode.Standalone;
    public List<SinkType> Sinks { get; set; } = new() { SinkType.Stdout };
    public LogFormat Format { get; set; } = LogFormat.Json;
    public string Level { get; set; } = "info";
    public List<SinkType> AuthorizedExtraSinks { get; set; } = new();
    public RetryPolicy Retry { get; set; } = new();
}

public class FileSinkOptions
{
    public string Path { get; set; } = "logs/plugin.log";
    public int MaxSizeMb { get; set; } = 100;
    public int MaxBackups { get; set; } = 3;
    public int MaxAgeDays { get; set; } = 28;
    public bool Compress { get; set; }
}

public static class LogPolicyResolution
{
    public static bool IsHostProxyMode =>
        Environment.GetEnvironmentVariable("POWERX_PROXY") == "1";

    public static LogPolicy ResolvePolicy(LogPolicy input)
    {
        var p = new LogPolicy
        {
            Mode = input.Mode,
            Sinks = input.Sinks.Count > 0 ? input.Sinks : new List<SinkType> { SinkType.Stdout },
            Format = input.Format,
            Level = string.IsNullOrWhiteSpace(input.Level) ? "info" : input.Level.ToLower(),
            AuthorizedExtraSinks = input.AuthorizedExtraSinks,
            Retry = new RetryPolicy
            {
                Enabled = input.Retry?.Enabled ?? true,
                MaxAttempts = input.Retry?.MaxAttempts > 0 ? input.Retry.MaxAttempts : 3,
                BackoffMs = input.Retry?.BackoffMs > 0 ? input.Retry.BackoffMs : 200
            }
        };

        p.Sinks = p.Sinks.Distinct().ToList();

        if (p.Mode == PolicyMode.Host)
        {
            if (!p.Sinks.Contains(SinkType.Stdout))
                p.Sinks.Insert(0, SinkType.Stdout);
            p.Format = LogFormat.Json;
        }

        return p;
    }

    public static LogPolicy ResolveWithHostDefaults(LogPolicy input)
    {
        return ResolveWithHostMode(input, IsHostProxyMode);
    }

    public static LogPolicy ResolveWithHostMode(LogPolicy input, bool hostMode)
    {
        var resolved = ResolvePolicy(input);
        if (!hostMode) return resolved;

        resolved.Mode = PolicyMode.Host;
        resolved.Format = LogFormat.Json;
        resolved.Sinks = new List<SinkType> { SinkType.Stdout };
        resolved.AuthorizedExtraSinks.Clear();
        return resolved;
    }

    public static SinkType PrimaryOutput(LogPolicy policy)
    {
        return policy.Sinks.FirstOrDefault();
    }

    public static LogPolicy DefaultPolicy() => new()
    {
        Mode = PolicyMode.Standalone,
        Sinks = new List<SinkType> { SinkType.Stdout },
        Format = LogFormat.Json,
        Level = "info",
        Retry = new RetryPolicy { Enabled = true, MaxAttempts = 3, BackoffMs = 200 }
    };
}
