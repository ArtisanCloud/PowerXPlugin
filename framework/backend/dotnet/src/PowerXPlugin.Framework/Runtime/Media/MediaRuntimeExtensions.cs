using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.Media;

public static class MediaRuntimeExtensions
{
    public static IServiceCollection AddPowerXMediaRuntime(this IServiceCollection services, ProviderMode mode, Func<IServiceProvider, IMediaService> localFactory, Func<IServiceProvider, IMediaService> delegatedFactory) =>
        services.AddPowerXRuntime("media", mode, localFactory, delegatedFactory);
}
