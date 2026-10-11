using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Common;

public sealed class DualModeRuntimeTests
{
    [Fact]
    public void Resolve_rejects_conflicting_bootstrap_sources()
    {
        var error = Assert.Throws<FrameworkAdapterException>(() => ProviderModeParser.Resolve("local", "delegated"));

        Assert.Equal(FrameworkRuntimeErrors.ProviderModeConflict, error.Code);
    }

    [Fact]
    public void Runtime_registration_constructs_only_the_local_adapter()
    {
        var localCalls = 0;
        var delegatedCalls = 0;
        var services = new ServiceCollection();
        services.AddPowerXRuntime<IProbe>(
            "probe",
            ProviderMode.Local,
            _ => { localCalls++; return new Probe("local"); },
            _ => { delegatedCalls++; return new Probe("delegated"); });

        using var provider = services.BuildServiceProvider();
        var runtime = provider.GetRequiredService<DualModeRuntime<IProbe>>();

        Assert.Equal(ProviderMode.Local, runtime.Mode);
        Assert.Equal("local", runtime.Service.Name);
        Assert.Equal(1, localCalls);
        Assert.Equal(0, delegatedCalls);
        Assert.Same(runtime.Service, provider.GetRequiredService<IProbe>());
    }

    [Fact]
    public void Runtime_registration_constructs_only_the_delegated_adapter()
    {
        var localCalls = 0;
        var delegatedCalls = 0;
        var services = new ServiceCollection();
        services.AddPowerXRuntime<IProbe>(
            "probe",
            ProviderMode.Delegated,
            _ => { localCalls++; return new Probe("local"); },
            _ => { delegatedCalls++; return new Probe("delegated"); });

        using var provider = services.BuildServiceProvider();
        var adapter = provider.GetRequiredService<IProbe>();

        Assert.Equal("delegated", adapter.Name);
        Assert.Equal(0, localCalls);
        Assert.Equal(1, delegatedCalls);
    }

    [Theory]
    [InlineData("", "Bearer")]
    [InlineData("credential", "unsupported")]
    public void Service_credential_validation_fails_closed(string value, string scheme)
    {
        Assert.Throws<FrameworkAdapterException>(() => new ServiceCredential(value, scheme).Validate());
    }

    private interface IProbe
    {
        string Name { get; }
    }

    private sealed record Probe(string Name) : IProbe;
}

