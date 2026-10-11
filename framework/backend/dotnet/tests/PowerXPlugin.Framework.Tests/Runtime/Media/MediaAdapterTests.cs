using System.Net;
using System.Text;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Media;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Media;

public sealed class MediaAdapterTests : IDisposable
{
    private readonly string _root = Path.Combine(Path.GetTempPath(), "powerx-media-tests-" + Guid.NewGuid().ToString("N"));

    [Fact]
    public async Task Local_adapter_stores_and_reads_without_host_transport()
    {
        var adapter = new LocalMediaAdapter(new LocalMediaAdapterOptions { LocalRoot = _root });
        await using var input = new MemoryStream(Encoding.UTF8.GetBytes("contract"));
        var stored = await adapter.StoreAssetBytesAsync(new CreateMediaAssetRequest { TenantUuid = "tenant-a", Name = "contract.txt", Folder = "crm/contracts" }, input, "text/plain", 8);

        Assert.Equal("local", adapter.Capabilities().ProviderMode);
        Assert.Equal("local", stored.StorageProvider);
        await using var output = await adapter.OpenReadAsync(stored.MediaAssetUuid, stored.StorageKey);
        using var reader = new StreamReader(output);
        Assert.Equal("contract", await reader.ReadToEndAsync());
    }

    [Fact]
    public async Task Delegated_client_uses_formal_media_contract_and_service_credential()
    {
        var handler = new StubHandler(async request =>
        {
            Assert.Equal(HttpMethod.Post, request.Method);
            Assert.Equal("/api/v1/tenant/media/assets", request.RequestUri!.AbsolutePath);
            Assert.Equal("Bearer framework-sts", request.Headers.Authorization!.ToString());
            var body = await request.Content!.ReadAsStringAsync();
            Assert.Contains("\"checksum\":\"sha256:abc\"", body);
            return Json("{\"data\":{\"asset_uuid\":\"asset-1\",\"name\":\"contract.pdf\",\"mime_type\":\"application/pdf\",\"size_bytes\":3}}");
        });
        var client = new PowerXMediaClient(
            new PowerXMediaClientOptions { BaseUrl = "https://core.example" },
            new StaticServiceCredentialProvider(new ServiceCredential("framework-sts")),
            new HttpClient(handler));

        var asset = await client.CreateAssetAsync(new CreateMediaAssetRequest { Name = "contract.pdf", MimeType = "application/pdf", SizeBytes = 3, ContentSha256 = "abc" });

        Assert.Equal("asset-1", asset.Uuid);
    }

    public void Dispose()
    {
        if (Directory.Exists(_root)) Directory.Delete(_root, recursive: true);
    }

    private static HttpResponseMessage Json(string body) => new(HttpStatusCode.OK) { Content = new StringContent(body, Encoding.UTF8, "application/json") };

    private sealed class StubHandler(Func<HttpRequestMessage, Task<HttpResponseMessage>> send) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => send(request);
    }
}
