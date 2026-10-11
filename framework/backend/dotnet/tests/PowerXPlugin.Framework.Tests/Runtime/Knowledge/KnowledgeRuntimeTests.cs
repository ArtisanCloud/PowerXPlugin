using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Knowledge;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Knowledge;

public sealed class KnowledgeRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    [Fact]
    public async Task Local_uses_plugin_store_and_preserves_tenant_boundary()
    {
        var service = new LocalKnowledgeAdapter(new Store());
        var spaces = await service.ListSpacesAsync(new ListKnowledgeSpacesRequest(Tenant));
        Assert.Single(spaces);
        var error = await Assert.ThrowsAsync<KnowledgeException>(() => service.ListSpacesAsync(new ListKnowledgeSpacesRequest("00000000-0000-4000-8000-000000000002")));
        Assert.Equal(KnowledgeErrors.InvalidResponse, error.Code);
    }
    [Fact]
    public async Task Delegated_uses_formal_routes_credential_and_rejects_tenant_override()
    {
        var requests = new List<HttpRequestMessage>();
        var client = new PowerXKnowledgeClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            requests.Add(request); var path = request.RequestUri!.AbsolutePath;
            Assert.Equal("Bearer sts", request.Headers.Authorization!.ToString());
            var body = path.EndsWith("/spaces", StringComparison.Ordinal) ? "{\"data\":{\"items\":[{\"space_uuid\":\"space-1\",\"name\":\"CRM\",\"status\":\"ready\"}]}}" : "{\"data\":{\"items\":[{\"space_uuid\":\"space-1\",\"document_uuid\":\"doc-1\",\"title\":\"One\",\"uri\":\"https://example/doc\",\"excerpt\":\"answer\",\"tags\":[]}]}}";
            return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(body, Encoding.UTF8, "application/json") };
        })));
        var spaces = await client.ListSpacesAsync(new ListKnowledgeSpacesRequest(Tenant));
        var result = await client.SearchAsync(new KnowledgeQuery(Tenant, "find", ["space-1"]));
        Assert.Equal("/api/v1/tenant/knowledge/spaces", requests[0].RequestUri!.AbsolutePath);
        Assert.Equal("/api/v1/tenant/knowledge/search", requests[1].RequestUri!.AbsolutePath);
        Assert.Single(spaces); Assert.Single(result.Chunks);
        var error = await Assert.ThrowsAsync<KnowledgeException>(() => client.SearchAsync(new KnowledgeQuery("00000000-0000-4000-8000-000000000002", "find")));
        Assert.Equal(KnowledgeErrors.TenantMismatch, error.Code);
    }
    private sealed class Store : ILocalKnowledgeStore
    {
        public KnowledgeCapabilities Capabilities { get; } = new("crm-local", ProviderMode.Local, new HashSet<string> { KnowledgeOperations.Retrieve, KnowledgeOperations.Search });
        public Task<IReadOnlyList<KnowledgeSpace>> ListSpacesAsync(ListKnowledgeSpacesRequest request, CancellationToken ct = default) => Task.FromResult<IReadOnlyList<KnowledgeSpace>>([new("local-space", "CRM", Tenant, "ready")]);
        public Task<KnowledgeSearchResult> SearchAsync(KnowledgeQuery query, CancellationToken ct = default) => Task.FromResult(new KnowledgeSearchResult("crm-local", [], 0));
        public Task<KnowledgeIndexJob> UpsertDocumentAsync(KnowledgeDocument document, CancellationToken ct = default) => throw new NotSupportedException();
        public Task<KnowledgeIndexJob> DeleteDocumentAsync(DeleteKnowledgeDocumentRequest request, CancellationToken ct = default) => throw new NotSupportedException();
        public Task<KnowledgeIndexJob> ReindexAsync(ReindexKnowledgeRequest request, CancellationToken ct = default) => throw new NotSupportedException();
        public Task<KnowledgeIndexJob> GetIndexJobAsync(GetKnowledgeIndexJobRequest request, CancellationToken ct = default) => throw new NotSupportedException();
    }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handler) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(handler(request)); }
}
