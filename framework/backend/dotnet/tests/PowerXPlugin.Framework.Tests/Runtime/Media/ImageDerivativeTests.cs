using System.Net;
using System.Text;
using System.Text.Json;
using System.Text.Json.Nodes;
using PowerXPlugin.Framework.Runtime.Common;
using PowerXPlugin.Framework.Runtime.Media;
using Xunit;

namespace PowerXPlugin.Framework.Tests.Runtime.Media;

public sealed class ImageDerivativeTests
{
    private const string AssetUuid = "00000000-0000-4000-8000-000000000001";
    private static string Fixture() => File.ReadAllText(Path.Combine(AppContext.BaseDirectory, "Fixtures", "host_asset_derivatives.json"));

    [Theory]
    [InlineData("create", "Bearer")]
    [InlineData("complete", "Bearer")]
    [InlineData("get", "Bearer")]
    [InlineData("create", "ApiKey")]
    [InlineData("complete", "ApiKey")]
    [InlineData("get", "ApiKey")]
    public async Task Host_operations_preserve_derivative_snapshot_without_extra_requests(string operation, string scheme)
    {
        var calls = 0;
        var handler = new StubHandler(async request =>
        {
            calls++;
            var path = "/api/v1/tenant/media/assets";
            var method = HttpMethod.Post;
            if (operation == "complete") path += $"/{AssetUuid}/complete-upload";
            if (operation == "get") { path += $"/{AssetUuid}"; method = HttpMethod.Get; }
            Assert.Equal(method, request.Method);
            Assert.Equal(path, request.RequestUri!.AbsolutePath);
            Assert.Equal(scheme + " test-service", request.Headers.Authorization!.ToString());
            Assert.False(request.Headers.Contains("X-Tenant-UUID"));
            if (operation == "get") Assert.Null(request.Content);
            else
            {
                using var body = JsonDocument.Parse(await request.Content!.ReadAsStringAsync());
                Assert.Equal("sha256:abc", body.RootElement.GetProperty("checksum").GetString());
                Assert.False(body.RootElement.TryGetProperty("derivatives", out _));
                Assert.False(body.RootElement.TryGetProperty("tenant_uuid", out _));
            }
            return Json(Fixture());
        });
        var client = Client(handler, scheme);
        var asset = operation switch
        {
            "create" => await client.CreateAssetAsync(new CreateMediaAssetRequest { Name = "fixture.png", MimeType = "image/png", SizeBytes = 4096, ContentSha256 = "abc" }),
            "complete" => await client.CompleteUploadAsync(AssetUuid, "abc"),
            _ => await client.GetAssetAsync(AssetUuid)
        };

        Assert.Equal(1, calls);
        Assert.Equal(AssetUuid, asset.Uuid);
        Assert.Equal("ready", asset.Status);
        Assert.Equal("image/png", asset.MimeType);
        Assert.Equal(4096, asset.SizeBytes);
        var derivatives = Assert.IsType<ImageDerivatives>(asset.Derivatives);
        Assert.Equal(asset.Uuid, derivatives.AssetUuid);
        Assert.Equal("ready", derivatives.UploadState);
        Assert.Equal("processing", derivatives.Status);
        Assert.Equal("standard-image-v1", derivatives.Recipe);
        Assert.Equal("00000000-0000-4000-8000-000000000002", derivatives.JobUuid);
        Assert.Equal(1, derivatives.Attempts);
        Assert.Null(derivatives.ErrorCode);
        Assert.False(derivatives.Retryable);
        Assert.Collection(derivatives.Items,
            preview => { Assert.Equal("preview", preview.Variant); Assert.Equal("ready", preview.Status); Assert.Equal("00000000-0000-4000-8000-000000000003", preview.VariantUuid); },
            thumbnail => { Assert.Equal("thumbnail", thumbnail.Variant); Assert.Equal("processing", thumbnail.Status); Assert.Null(thumbnail.VariantUuid); });

        var serialized = JsonSerializer.SerializeToElement(asset, new JsonSerializerOptions(JsonSerializerDefaults.Web));
        var wire = serialized.GetProperty("derivatives");
        Assert.Equal(AssetUuid, wire.GetProperty("asset_uuid").GetString());
        Assert.Equal(derivatives.Items[0].VariantUuid, wire.GetProperty("items")[0].GetProperty("variant_uuid").GetString());
        Assert.False(wire.GetProperty("items")[1].TryGetProperty("variant_uuid", out _));
    }

    [Theory]
    [InlineData("absent")]
    [InlineData("null")]
    [InlineData("queued")]
    [InlineData("ready")]
    [InlineData("failed")]
    [InlineData("unsupported")]
    [InlineData("unavailable")]
    public async Task Optional_and_processing_states_do_not_change_upload_status(string state)
    {
        var response = JsonNode.Parse(Fixture())!;
        var data = response["data"]!.AsObject();
        if (state == "absent") data.Remove("derivatives");
        else if (state == "null") data["derivatives"] = null;
        else
        {
            var derivatives = data["derivatives"]!;
            derivatives["status"] = state;
            derivatives["attempts"] = state == "failed" ? 3 : 0;
            derivatives["retryable"] = state == "failed";
            if (state == "failed") derivatives["error_code"] = "MEDIA_IMAGE_DECODE_FAILED";
            if (state == "ready")
            {
                derivatives["items"]![1]!["status"] = "ready";
                derivatives["items"]![1]!["variant_uuid"] = "00000000-0000-4000-8000-000000000004";
            }
            else derivatives["items"] = new JsonArray();
        }
        var client = Client(new StubHandler(_ => Task.FromResult(Json(response.ToJsonString()))));
        var asset = await client.GetAssetAsync(AssetUuid);

        Assert.Equal("ready", asset.Status);
        if (state is "absent" or "null")
        {
            Assert.Null(asset.Derivatives);
            var serialized = JsonSerializer.SerializeToElement(asset, new JsonSerializerOptions(JsonSerializerDefaults.Web));
            Assert.False(serialized.TryGetProperty("derivatives", out _));
            return;
        }
        Assert.Equal(state, asset.Derivatives!.Status);
        if (state == "failed")
        {
            Assert.Equal("MEDIA_IMAGE_DECODE_FAILED", asset.Derivatives.ErrorCode);
            Assert.Equal(3, asset.Derivatives.Attempts);
            Assert.True(asset.Derivatives.Retryable);
        }
    }

    [Fact]
    public async Task Malformed_derivatives_fail_instead_of_becoming_ready()
    {
        var client = Client(new StubHandler(_ => Task.FromResult(Json($"{{\"data\":{{\"asset_uuid\":\"{AssetUuid}\",\"status\":\"ready\",\"derivatives\":\"ready\"}}}}"))));
        await Assert.ThrowsAsync<JsonException>(() => client.GetAssetAsync(AssetUuid));
    }

    private static PowerXMediaClient Client(HttpMessageHandler handler, string scheme = "Bearer") => new(
        new PowerXMediaClientOptions { BaseUrl = "https://core.example" },
        new StaticServiceCredentialProvider(new ServiceCredential("test-service", scheme)), new HttpClient(handler));

    private static HttpResponseMessage Json(string body) => new(HttpStatusCode.OK) { Content = new StringContent(body, Encoding.UTF8, "application/json") };

    private sealed class StubHandler(Func<HttpRequestMessage, Task<HttpResponseMessage>> send) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => send(request);
    }
}
