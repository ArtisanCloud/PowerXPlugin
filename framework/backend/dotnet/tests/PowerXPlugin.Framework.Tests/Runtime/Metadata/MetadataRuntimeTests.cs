using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Metadata;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Metadata;

public sealed class MetadataRuntimeTests
{
    private const string Tenant = "00000000-0000-4000-8000-000000000001";
    [Fact]
    public async Task Delegated_uses_published_tag_binding_contract_without_tenant_payload()
    {
        HttpRequestMessage? seen = null; string? payload = null;
        var client = new PowerXMetadataClient("https://core.example", Tenant, new StaticServiceCredentialProvider(new ServiceCredential("sts")), new HttpClient(new Handler(request =>
        {
            seen = request; payload = request.Content!.ReadAsStringAsync().GetAwaiter().GetResult();
            return new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("{\"data\":{\"binding_uuid\":\"binding-1\",\"tag_uuid\":\"11111111-1111-4111-8111-111111111111\",\"resource_type\":\"crm.customer\",\"resource_uuid\":\"22222222-2222-4222-8222-222222222222\"}}", Encoding.UTF8, "application/json") };
        })));
        await client.CreateTagBindingAsync(new MetadataScope(Tenant), new CreateTagBindingRequest("11111111-1111-4111-8111-111111111111", "crm.customer", "22222222-2222-4222-8222-222222222222"));
        Assert.Equal(HttpMethod.Post, seen!.Method);
        Assert.Equal("/api/v1/tenant/metadata/tag-bindings", seen.RequestUri!.AbsolutePath);
        Assert.Equal("Bearer sts", seen.Headers.Authorization!.ToString());
        Assert.DoesNotContain("tenant", payload, StringComparison.OrdinalIgnoreCase);
    }
    [Fact]
    public async Task Local_validates_scope_before_store_is_called()
    {
        var local = new LocalMetadataAdapter(new Store());
        var error = await Assert.ThrowsAsync<MetadataException>(() => local.ListTagsAsync(new MetadataScope("not-a-uuid"), new MetadataListRequest()));
        Assert.Equal(MetadataErrors.InvalidArgument, error.Code);
    }
    private sealed class Store : ILocalMetadataStore
    {
        private static Exception Missing() => new Xunit.Sdk.XunitException("store should not be called");
        public Task<MetadataPage<DictionaryNamespace>> ListDictionaryNamespacesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => throw Missing(); public Task<DictionaryNamespace> CreateDictionaryNamespaceAsync(MetadataScope s, CreateDictionaryNamespaceRequest r, CancellationToken ct = default) => throw Missing(); public Task<DictionaryNamespace> UpdateDictionaryNamespaceAsync(MetadataScope s, UpdateDictionaryNamespaceRequest r, CancellationToken ct = default) => throw Missing(); public Task<MetadataPage<DictionaryItem>> ListDictionaryItemsAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) => throw Missing(); public Task<DictionaryItem> CreateDictionaryItemAsync(MetadataScope s, CreateDictionaryItemRequest r, CancellationToken ct = default) => throw Missing(); public Task<DictionaryItem> UpdateDictionaryItemAsync(MetadataScope s, UpdateDictionaryItemRequest r, CancellationToken ct = default) => throw Missing(); public Task<MetadataPage<Taxonomy>> ListTaxonomiesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => throw Missing(); public Task<Taxonomy> CreateTaxonomyAsync(MetadataScope s, CreateTaxonomyRequest r, CancellationToken ct = default) => throw Missing(); public Task<MetadataPage<TaxonomyNode>> ListTaxonomyNodesAsync(MetadataScope s, string id, MetadataListRequest r, CancellationToken ct = default) => throw Missing(); public Task<TaxonomyNode> CreateTaxonomyNodeAsync(MetadataScope s, CreateTaxonomyNodeRequest r, CancellationToken ct = default) => throw Missing(); public Task<TaxonomyNode> UpdateTaxonomyNodeAsync(MetadataScope s, UpdateTaxonomyNodeRequest r, CancellationToken ct = default) => throw Missing(); public Task<MetadataPage<Tag>> ListTagsAsync(MetadataScope s, MetadataListRequest r, string t = "", string n = "", CancellationToken ct = default) => throw Missing(); public Task<Tag> CreateTagAsync(MetadataScope s, CreateTagRequest r, CancellationToken ct = default) => throw Missing(); public Task<TagBinding> CreateTagBindingAsync(MetadataScope s, CreateTagBindingRequest r, CancellationToken ct = default) => throw Missing(); public Task DeleteTagBindingAsync(MetadataScope s, string id, CancellationToken ct = default) => throw Missing(); public Task<MetadataPage<ResourceType>> ListResourceTypesAsync(MetadataScope s, MetadataListRequest r, CancellationToken ct = default) => throw Missing(); public Task<ResourceType> CreateResourceTypeAsync(MetadataScope s, CreateResourceTypeRequest r, CancellationToken ct = default) => throw Missing(); public Task<ResourceType> UpdateResourceTypeAsync(MetadataScope s, UpdateResourceTypeRequest r, CancellationToken ct = default) => throw Missing();
    }
    private sealed class Handler(Func<HttpRequestMessage, HttpResponseMessage> handler) : HttpMessageHandler { protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken ct) => Task.FromResult(handler(request)); }
}
