namespace PowerXPlugin.Framework.Runtime.Common;

/// <summary>Bootstrap-owned credential source for standalone/local development.</summary>
public sealed class StaticServiceCredentialProvider : IServiceCredentialProvider
{
    private readonly ServiceCredential _credential;
    public StaticServiceCredentialProvider(ServiceCredential credential) => _credential = credential.Validate();
    public ValueTask<ServiceCredential> GetCredentialAsync(CancellationToken cancellationToken = default) => ValueTask.FromResult(_credential);
}
