using System.Text.RegularExpressions;
using System.Text.Json.Serialization;
namespace PowerXPlugin.Framework.IAM.Models;
public sealed record WSIdentityQuery
{
    [JsonIgnore] public string Handle { get; init; } = "";
    public override string ToString() => "WSIdentityQuery{redacted}";
    public void Validate()
    {
        if(string.IsNullOrEmpty(Handle)) throw new IAMAdapterException("IAM_CONTEXT_WS_HANDLE_REQUIRED");
        if(!Regex.IsMatch(Handle,"^pxwsi_v1_[A-Za-z0-9_-]{43}$")) throw new IAMAdapterException("IAM_CONTEXT_WS_HANDLE_INVALID");
    }
}
public sealed record WSIdentity(DelegatedIdentity Identity)
{
    public void Validate(string pluginId,DateTimeOffset now)
    {
        if (Identity is null) throw new IAMAdapterException("IAM_CONTEXT_RESPONSE_INVALID");
        Identity.ValidateSchema(pluginId,now,"powerx.iam.plugin-ws-identity/v1");
    }
    public override string ToString() => "WSIdentity{redacted}";
}
