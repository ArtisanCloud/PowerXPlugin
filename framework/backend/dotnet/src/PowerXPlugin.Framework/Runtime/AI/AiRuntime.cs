using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.DependencyInjection;
using PowerXPlugin.Framework.Runtime.Common;

namespace PowerXPlugin.Framework.Runtime.AI;

public sealed record AiTenantScope(string TenantUuid);
public sealed record AiContentItem(string Type, string Content = "", string Url = "", string Role = "");
public sealed record AiLlmInvokeRequest(string ModelKey, IReadOnlyList<AiContentItem> Inputs, IReadOnlyDictionary<string, JsonElement>? Params = null);
public sealed record AiLlmInvokeResult(string Type, string Text, string FinishReason = "", IReadOnlyDictionary<string, JsonElement>? Usage = null);
public sealed record AiModel(string ModelKey, string Provider, string Model, string Label, string Source, bool Configured, bool ProfileConfigured, IReadOnlyDictionary<string, JsonElement>? Defaults = null, IReadOnlyList<string>? Tags = null);
public sealed record AiModelList(string Environment, IReadOnlyList<AiModel> Items);
public sealed record AiCreateSessionRequest(string ModelKey, string Title = "");
public sealed record AiSession(string SessionId);
public sealed record AiAppendSessionMessageRequest(string Role, IReadOnlyList<AiContentItem> Content);
public sealed record AiEmbeddingRequest(string ModelKey, IReadOnlyList<string> Inputs, IReadOnlyDictionary<string, JsonElement>? Params = null);
public sealed record AiEmbeddingResult(IReadOnlyList<IReadOnlyList<float>> Vectors, IReadOnlyDictionary<string, JsonElement>? Usage = null, IReadOnlyDictionary<string, string>? Extra = null);
public sealed record AiModalRequest(string ModelKey, IReadOnlyList<AiContentItem> Inputs, IReadOnlyDictionary<string, JsonElement>? Params = null);
public sealed record AiModalResult(JsonElement Data);
public sealed record AiStreamRequest(AiLlmInvokeRequest Invocation, bool IncludeUsage = false);
public sealed record AiStreamEvent(string Type, string TraceId = "", string Delta = "", int Index = 0, string Text = "", string FinishReason = "", string Code = "", string Message = "");

public interface IGenerativeService
{
    Task<AiModelList> ListModelsAsync(AiTenantScope scope, string provider = "", CancellationToken ct = default);
    Task<AiLlmInvokeResult> InvokeLlmAsync(AiTenantScope scope, AiLlmInvokeRequest request, CancellationToken ct = default);
    Task StreamLlmAsync(AiTenantScope scope, AiStreamRequest request, Func<AiStreamEvent, CancellationToken, Task> onEvent, CancellationToken ct = default);
    Task<AiSession> CreateSessionAsync(AiTenantScope scope, AiCreateSessionRequest request, CancellationToken ct = default);
    Task AppendSessionMessageAsync(AiTenantScope scope, string sessionId, AiAppendSessionMessageRequest request, CancellationToken ct = default);
    Task StreamSessionAsync(AiTenantScope scope, string sessionId, Func<AiStreamEvent, CancellationToken, Task> onEvent, CancellationToken ct = default);
    Task<AiEmbeddingResult> EmbedAsync(AiTenantScope scope, AiEmbeddingRequest request, CancellationToken ct = default);
    Task<AiModalResult> InvokeVlmAsync(AiTenantScope scope, AiModalRequest request, CancellationToken ct = default);
    Task<AiModalResult> InvokeImageAsync(AiTenantScope scope, AiModalRequest request, CancellationToken ct = default);
    Task<AiModalResult> InvokeVideoAsync(AiTenantScope scope, AiModalRequest request, CancellationToken ct = default);
    Task<AiModalResult> InvokeTtsAsync(AiTenantScope scope, AiModalRequest request, CancellationToken ct = default);
}
public interface ILocalGenerativeStore : IGenerativeService { }
public sealed class LocalGenerativeService(ILocalGenerativeStore store) : IGenerativeService
{
    private readonly ILocalGenerativeStore _store = store ?? throw new ArgumentNullException(nameof(store));
    public Task<AiModelList> ListModelsAsync(AiTenantScope s,string p="",CancellationToken ct=default){ V(s);return _store.ListModelsAsync(s,p,ct); }
    public Task<AiLlmInvokeResult> InvokeLlmAsync(AiTenantScope s,AiLlmInvokeRequest r,CancellationToken ct=default){ V(s); Llm(r);return _store.InvokeLlmAsync(s,r,ct); }
    public Task StreamLlmAsync(AiTenantScope s,AiStreamRequest r,Func<AiStreamEvent,CancellationToken,Task> f,CancellationToken ct=default){ V(s);Llm(r.Invocation);return f is null?Task.FromException(new AiRuntimeException(AiErrors.InvalidArgument)):_store.StreamLlmAsync(s,r,f,ct); }
    public Task<AiSession>CreateSessionAsync(AiTenantScope s,AiCreateSessionRequest r,CancellationToken ct=default){V(s);Model(r.ModelKey);return _store.CreateSessionAsync(s,r,ct);}
    public Task AppendSessionMessageAsync(AiTenantScope s,string id,AiAppendSessionMessageRequest r,CancellationToken ct=default){V(s);Session(id);if(r is null||r.Role is not("user"or"assistant")||r.Content is null||r.Content.Count==0)throw new AiRuntimeException(AiErrors.InvalidArgument);return _store.AppendSessionMessageAsync(s,id,r,ct);}
    public Task StreamSessionAsync(AiTenantScope s,string id,Func<AiStreamEvent,CancellationToken,Task> f,CancellationToken ct=default){V(s);Session(id);return f is null?Task.FromException(new AiRuntimeException(AiErrors.InvalidArgument)):_store.StreamSessionAsync(s,id,f,ct);}
    public Task<AiEmbeddingResult>EmbedAsync(AiTenantScope s,AiEmbeddingRequest r,CancellationToken ct=default){V(s);Model(r.ModelKey);if(r.Inputs is null||r.Inputs.Count==0)throw new AiRuntimeException(AiErrors.InvalidArgument);return _store.EmbedAsync(s,r,ct);}
    public Task<AiModalResult>InvokeVlmAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,r,_store.InvokeVlmAsync,ct);
    public Task<AiModalResult>InvokeImageAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,r,_store.InvokeImageAsync,ct);
    public Task<AiModalResult>InvokeVideoAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,r,_store.InvokeVideoAsync,ct);
    public Task<AiModalResult>InvokeTtsAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,r,_store.InvokeTtsAsync,ct);
    private static Task<AiModalResult>Modal(AiTenantScope s,AiModalRequest r,Func<AiTenantScope,AiModalRequest,CancellationToken,Task<AiModalResult>> f,CancellationToken ct){V(s);Model(r.ModelKey);if(r.Inputs is null||r.Inputs.Count==0)throw new AiRuntimeException(AiErrors.InvalidArgument);return f(s,r,ct);}
    internal static void V(AiTenantScope s){if(s is null||!Guid.TryParse(s.TenantUuid,out var v)||v==Guid.Empty||v.ToString()!=s.TenantUuid)throw new AiRuntimeException(AiErrors.InvalidArgument);} internal static void Model(string? x){if(string.IsNullOrWhiteSpace(x))throw new AiRuntimeException(AiErrors.InvalidArgument);} internal static void Session(string? x){if(string.IsNullOrWhiteSpace(x))throw new AiRuntimeException(AiErrors.InvalidArgument);} internal static void Llm(AiLlmInvokeRequest r){if(r is null||r.Inputs is null||r.Inputs.Count==0)throw new AiRuntimeException(AiErrors.InvalidArgument);Model(r.ModelKey);}
}

public sealed class PowerXGenerativeClient(string baseUrl,string tenantUuid,IServiceCredentialProvider credentials,HttpClient? http=null):IGenerativeService
{
    private readonly string _base=baseUrl.TrimEnd('/');private readonly string _tenant=tenantUuid;private readonly IServiceCredentialProvider _credentials=credentials;private readonly HttpClient _http=http??new();private static readonly JsonSerializerOptions Json=new(JsonSerializerDefaults.Web){PropertyNamingPolicy=JsonNamingPolicy.SnakeCaseLower};
    public Task<AiModelList>ListModelsAsync(AiTenantScope s,string provider="",CancellationToken ct=default)=>Send<AiModelList>(s,HttpMethod.Get,"/api/v1/ai/llm/models"+(string.IsNullOrWhiteSpace(provider)?"":"?provider="+Uri.EscapeDataString(provider)),null,ct);
    public Task<AiLlmInvokeResult>InvokeLlmAsync(AiTenantScope s,AiLlmInvokeRequest r,CancellationToken ct=default){LocalGenerativeService.Llm(r);return Send<AiLlmInvokeResult>(s,HttpMethod.Post,"/api/v1/ai/llm/invoke",r,ct);}
    public Task<AiSession>CreateSessionAsync(AiTenantScope s,AiCreateSessionRequest r,CancellationToken ct=default){LocalGenerativeService.Model(r.ModelKey);return Send<AiSession>(s,HttpMethod.Post,"/api/v1/ai/llm/sessions",r,ct);}
    public Task AppendSessionMessageAsync(AiTenantScope s,string id,AiAppendSessionMessageRequest r,CancellationToken ct=default){LocalGenerativeService.V(s);LocalGenerativeService.Session(id);return Send<object>(s,HttpMethod.Post,"/api/v1/ai/llm/sessions/"+Uri.EscapeDataString(id)+"/messages",r,ct);}
    public Task<AiEmbeddingResult>EmbedAsync(AiTenantScope s,AiEmbeddingRequest r,CancellationToken ct=default){LocalGenerativeService.Model(r.ModelKey);return Send<AiEmbeddingResult>(s,HttpMethod.Post,"/api/v1/ai/embedding/invoke",r,ct);}
    public Task<AiModalResult>InvokeVlmAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,"/api/v1/ai/vlm/invoke",r,ct);public Task<AiModalResult>InvokeImageAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,"/api/v1/ai/image/invoke",r,ct);public Task<AiModalResult>InvokeVideoAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,"/api/v1/ai/video/invoke",r,ct);public Task<AiModalResult>InvokeTtsAsync(AiTenantScope s,AiModalRequest r,CancellationToken ct=default)=>Modal(s,"/api/v1/ai/tts/invoke",r,ct);
    public Task StreamLlmAsync(AiTenantScope s,AiStreamRequest r,Func<AiStreamEvent,CancellationToken,Task>f,CancellationToken ct=default)=>Stream(s,HttpMethod.Post,"/api/v1/ai/llm/stream",new{model_key=r.Invocation.ModelKey,inputs=r.Invocation.Inputs,@params=r.Invocation.Params,stream_options=new{include_usage=r.IncludeUsage}},f,ct);
    public Task StreamSessionAsync(AiTenantScope s,string id,Func<AiStreamEvent,CancellationToken,Task>f,CancellationToken ct=default){LocalGenerativeService.Session(id);return Stream(s,HttpMethod.Get,"/api/v1/ai/llm/sessions/"+Uri.EscapeDataString(id)+"/stream",null,f,ct);}
    private Task<AiModalResult>Modal(AiTenantScope s,string p,AiModalRequest r,CancellationToken ct){LocalGenerativeService.Model(r.ModelKey);return Send<AiModalResult>(s,HttpMethod.Post,p,r,ct);}
    private async Task<T>Send<T>(AiTenantScope s,HttpMethod m,string p,object?b,CancellationToken ct){V(s);using var q=await Request(s,m,p,b,ct,false);using var r=await _http.SendAsync(q,ct);var raw=await r.Content.ReadAsStringAsync(ct);if(!r.IsSuccessStatusCode)throw new AiRuntimeException(r.StatusCode==HttpStatusCode.Forbidden?AiErrors.Forbidden:AiErrors.Upstream);try{using var j=JsonDocument.Parse(raw);if(!j.RootElement.TryGetProperty("data",out var d)||d.ValueKind==JsonValueKind.Null)throw new AiRuntimeException(AiErrors.InvalidResponse);return JsonSerializer.Deserialize<T>(d.GetRawText(),Json)??throw new AiRuntimeException(AiErrors.InvalidResponse);}catch(JsonException){throw new AiRuntimeException(AiErrors.InvalidResponse);}}
    private async Task Stream(AiTenantScope s,HttpMethod m,string p,object?b,Func<AiStreamEvent,CancellationToken,Task>f,CancellationToken ct){V(s);if(f is null)throw new AiRuntimeException(AiErrors.InvalidArgument);using var q=await Request(s,m,p,b,ct,true);using var r=await _http.SendAsync(q,HttpCompletionOption.ResponseHeadersRead,ct);if(!r.IsSuccessStatusCode)throw new AiRuntimeException(r.StatusCode==HttpStatusCode.Forbidden?AiErrors.Forbidden:AiErrors.Upstream);if(!string.Equals(r.Content.Headers.ContentType?.MediaType,"text/event-stream",StringComparison.OrdinalIgnoreCase))throw new AiRuntimeException(AiErrors.InvalidResponse);await using var stream=await r.Content.ReadAsStreamAsync(ct);using var reader=new StreamReader(stream);var type="";var data=new StringBuilder();while(await reader.ReadLineAsync(ct) is {}line){if(line.Length==0){if(data.Length>0){try{using var j=JsonDocument.Parse(data.ToString());var e=j.RootElement;await f(new(type,e.TryGetProperty("traceId",out var t)?t.GetString()??"":"",e.TryGetProperty("delta",out var d)?d.GetString()??"":"",e.TryGetProperty("index",out var i)?i.GetInt32():0,e.TryGetProperty("text",out var x)?x.GetString()??"":"",e.TryGetProperty("finish_reason",out var z)?z.GetString()??"":"",e.TryGetProperty("code",out var c)?c.GetString()??"":"",e.TryGetProperty("message",out var msg)?msg.GetString()??"":""),ct);}catch(JsonException){throw new AiRuntimeException(AiErrors.InvalidResponse);}data.Clear();type="";}continue;}if(line.StartsWith("event:"))type=line[6..].Trim();else if(line.StartsWith("data:")){if(data.Length>0)data.Append('\n');data.Append(line[5..].TrimStart());}}}
    private async Task<HttpRequestMessage>Request(AiTenantScope s,HttpMethod m,string p,object?b,CancellationToken ct,bool stream){var credential=(await _credentials.GetCredentialAsync(ct)).Validate();var q=new HttpRequestMessage(m,_base+p);q.Headers.Authorization=new AuthenticationHeaderValue(credential.NormalizedAuthScheme,credential.Value);if(stream)q.Headers.Accept.ParseAdd("text/event-stream");if(b is not null)q.Content=new StringContent(JsonSerializer.Serialize(b,Json),Encoding.UTF8,"application/json");return q;}
    private void V(AiTenantScope s){LocalGenerativeService.V(s);if(s.TenantUuid!=_tenant)throw new AiRuntimeException(AiErrors.TenantMismatch);}
}
public static class AiRuntimeExtensions{public static IServiceCollection AddPowerXGenerativeRuntime(this IServiceCollection s,ProviderMode m,Func<IServiceProvider,IGenerativeService>l,Func<IServiceProvider,IGenerativeService>d)=>s.AddPowerXRuntime("ai.generative",m,l,d);}public static class AiErrors{public const string InvalidArgument="FRAMEWORK_AI_INVALID_ARGUMENT",InvalidResponse="FRAMEWORK_AI_INVALID_RESPONSE",TenantMismatch="FRAMEWORK_AI_TENANT_MISMATCH",Forbidden="FRAMEWORK_AI_FORBIDDEN",Upstream="FRAMEWORK_AI_UPSTREAM_DEPENDENCY";}public sealed class AiRuntimeException(string code):InvalidOperationException(code){public string Code{get;}=code;}
