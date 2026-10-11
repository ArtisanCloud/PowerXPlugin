using System.Net;
using System.Text;
using System.Text.Json;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Knowledge;
using Xunit;
namespace PowerXPlugin.Framework.Tests.Runtime.Knowledge;

public sealed class KnowledgeSnapshotTests
{
 const string Tenant = "11111111-1111-4111-8111-111111111111";
 const string Space = "22222222-2222-4222-8222-222222222222";
 const string Job = "33333333-3333-4333-8333-333333333333";
 [Theory]
 [InlineData("ApiKey")]
 [InlineData("Bearer")]
 public async Task Formal_snapshot_preserves_zero_false_arrays_and_readback(string scheme)
 {
  var client = new PowerXKnowledgeClient("https://core.example",Tenant,new StaticServiceCredentialProvider(new ServiceCredential("fixture", scheme)),new HttpClient(new Handler(request => {
   Assert.Equal(scheme+" fixture",request.Headers.Authorization!.ToString());
   if(request.Method == HttpMethod.Post) {
    using var body = JsonDocument.Parse(request.Content!.ReadAsStringAsync().GetAwaiter().GetResult());
    var settings=body.RootElement.GetProperty("ingestion");
    Assert.Equal(0,settings.GetProperty("chunk_size").GetInt32()); Assert.False(settings.GetProperty("page_priority").GetBoolean());Assert.Equal(0,settings.GetProperty("separators").GetArrayLength());Assert.Equal("page",settings.GetProperty("segment_order")[0].GetString());
    Assert.False(body.RootElement.TryGetProperty("tenant_uuid",out _));Assert.False(settings.TryGetProperty("processor_profile",out _));
    return Response(202,"{\"data\":{\"job_uuid\":\""+Job+"\",\"space_uuid\":\""+Space+"\",\"status\":\"queued\"}}");
   }
   if(request.RequestUri!.AbsolutePath.EndsWith("/chunks")) return Response(200,"{\"data\":{\"items\":[{\"uuid\":\"44444444-4444-4444-8444-444444444444\",\"space_uuid\":\""+Space+"\",\"job_uuid\":\""+Job+"\",\"document_uuid\":\"55555555-5555-4555-8555-555555555555\",\"checksum\":\"fixture\",\"content\":\"fixture\"}]}}");
   return Response(200,"{\"data\":{\"job_uuid\":\""+Job+"\",\"space_uuid\":\""+Space+"\",\"status\":\"succeeded\",\"chunk_count\":4,\"trace_id\":\"fixture\",\"snapshot_checksum\":\"fixture\",\"requested_config\":{\"schema\":\"powerx.knowledge.document-ingestion/v1\",\"chunk_size\":640},\"effective_config\":{\"schema\":\"powerx.knowledge.document-ingestion/v1\",\"index_mode\":\"lexical_chunks\",\"chunk_size\":640,\"chunk_overlap\":80}}}");
  })));
  await client.UpsertDocumentAsync(new KnowledgeDocument("",Space,Tenant,"fixture","plugin://fixture","fixture","text/plain","fixture","v1",HostIngestion:new DocumentIngestionSettings {Schema="powerx.knowledge.document-ingestion/v1",ChunkSize=0,ChunkOverlap=0,PagePriority=false,Separators=[],SegmentOrder=["page","segment","separator","size"]}));
  var job=await client.GetIndexJobAsync(new GetKnowledgeIndexJobRequest(Tenant,Job));
  Assert.Equal(4,job.ChunkCount);Assert.Equal(640,job.EffectiveConfig!.ChunkSize);Assert.Equal(80,job.EffectiveConfig.ChunkOverlap);Assert.Equal(640,job.RequestedConfig!.ChunkSize);Assert.Equal("fixture",job.TraceId);
  Assert.Single(await client.GetIndexJobChunksAsync(new GetKnowledgeIndexJobRequest(Tenant,Job)));
 }
 [Fact]
 public async Task Unsupported_setting_preserves_Core_status_code_and_trace()
 {
  var client=new PowerXKnowledgeClient("https://core.example",Tenant,new StaticServiceCredentialProvider(new ServiceCredential("fixture")),new HttpClient(new Handler(_=>Response(422,"{\"reason_code\":\"KNOWLEDGE_UNSUPPORTED_INGESTION_SETTING\",\"request_id\":\"fixture\"}"))));
  var error=await Assert.ThrowsAsync<KnowledgeException>(()=>client.UpsertDocumentAsync(new KnowledgeDocument("",Space,Tenant,"fixture","plugin://fixture","fixture","text/plain","fixture","v1",HostIngestion:new DocumentIngestionSettings{Schema="powerx.knowledge.document-ingestion/v1",SegmentMode="semantic"})));
  Assert.Equal(422,error.StatusCode);Assert.Equal("KNOWLEDGE_UNSUPPORTED_INGESTION_SETTING",error.Code);Assert.Equal("fixture",error.TraceId);
 }
 static HttpResponseMessage Response(int status,string body)=>new((HttpStatusCode)status){Content=new StringContent(body,Encoding.UTF8,"application/json")};
 sealed class Handler(Func<HttpRequestMessage,HttpResponseMessage> handler):HttpMessageHandler{protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,CancellationToken ct)=>Task.FromResult(handler(request));}
}
