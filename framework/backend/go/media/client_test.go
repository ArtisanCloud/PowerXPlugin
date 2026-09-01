package media

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

func TestClientCreatePresignUpdateUsesMediaCapabilities(t *testing.T) {
	gw := &fakeGateway{}
	client := NewClient(gw, &http.Client{Transport: fakeHTTPTransport{}})

	asset, err := client.CreateAsset(context.Background(), CreateAssetInput{
		TenantUUID:       "tenant-001",
		Name:             "design.png",
		Driver:           "local",
		OwnerSubjectType: "ai_craft_asset",
		OwnerSubjectID:   "track/original/design.png",
		UploadChannel:    UploadChannelPresigned,
		Tags:             []string{"ai-craft"},
		ObjectKey:        "2a66f690-4ca6-5154-acb2-645171e4a87f",
		SizeBytes:        10,
		MimeType:         "image/png",
		ContentSHA256:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Metadata:         map[string]string{"source": "unit-test"},
	})
	require.NoError(t, err)
	require.Equal(t, "media-asset-001", asset.UUID)

	ticket, err := client.PresignAsset(context.Background(), PresignAssetInput{
		TenantUUID:       "tenant-001",
		UUID:             asset.UUID,
		Action:           PresignActionUpload,
		Method:           http.MethodPut,
		ExpiresInSeconds: 900,
		Headers:          map[string]string{"Content-Type": "image/png"},
	})
	require.NoError(t, err)
	require.Equal(t, http.MethodPut, ticket.Method)
	require.NoError(t, client.UploadBytes(context.Background(), ticket, bytes.NewReader([]byte("image-body")), "image/png"))

	status := BusinessStatusUnderReview
	_, err = client.UpdateAsset(context.Background(), UpdateAssetInput{
		TenantUUID:     "tenant-001",
		UUID:           asset.UUID,
		BusinessStatus: &status,
	})
	require.NoError(t, err)

	require.Len(t, gw.calls, 3)
	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[0].CapabilityID)
	require.Equal(t, "CreateMediaAsset", gw.calls[0].Action)
	require.Equal(t, "rest", gw.calls[0].PreferredProtocol)
	require.Equal(t, "tenant-001", gw.calls[0].TenantUUID)
	createPayload, ok := gw.calls[0].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, createPayload["method"])
	require.Equal(t, "/api/v1/media/assets", createPayload["endpoint"])
	createBody, ok := createPayload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "local", createBody["driver"])
	require.Equal(t, "presign_upload", createBody["uploadMethod"])
	require.Equal(t, "ai_craft_asset", createBody["ownerSubjectType"])
	require.Equal(t, "track/original/design.png", createBody["ownerSubjectId"])
	require.Equal(t, "2a66f690-4ca6-5154-acb2-645171e4a87f", createBody["objectKey"])
	require.Equal(t, int64(10), createBody["sizeBytes"])
	require.Equal(t, "image/png", createBody["mimeType"])
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", createBody["contentSha256"])
	metadata, ok := createBody["metadata"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "unit-test", metadata["source"])
	require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", metadata["content_sha256"])

	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[1].CapabilityID)
	require.Equal(t, "PresignMediaAsset", gw.calls[1].Action)
	presignPayload, ok := gw.calls[1].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, presignPayload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001/presign", presignPayload["endpoint"])
	presignBody, ok := presignPayload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "upload", presignBody["action"])
	require.Equal(t, http.MethodPut, presignBody["method"])
	require.Equal(t, uint32(900), presignBody["expiresInSeconds"])

	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[2].CapabilityID)
	require.Equal(t, "UpdateMediaAsset", gw.calls[2].Action)
	updatePayload, ok := gw.calls[2].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPatch, updatePayload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001", updatePayload["endpoint"])
	updateBody, ok := updatePayload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "under_review", updateBody["business_status"])
}

func TestClientDownloadUsesPresignTicket(t *testing.T) {
	gw := &fakeGateway{}
	client := NewClient(gw, &http.Client{Transport: fakeHTTPTransport{}})

	ticket, err := client.PresignAsset(context.Background(), PresignAssetInput{
		TenantUUID:       "tenant-001",
		UUID:             "media-asset-001",
		Action:           PresignActionDownload,
		Method:           http.MethodGet,
		ExpiresInSeconds: 300,
	})
	require.NoError(t, err)
	body, err := client.DownloadBytes(context.Background(), ticket)
	require.NoError(t, err)
	require.Equal(t, []byte("download-body"), body)

	require.Len(t, gw.calls, 1)
	require.Equal(t, "PresignMediaAsset", gw.calls[0].Action)
	payload, ok := gw.calls[0].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, payload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001/presign", payload["endpoint"])
	presignBody, ok := payload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "download", presignBody["action"])
	require.Equal(t, http.MethodGet, presignBody["method"])
}

func TestClientMediaAssetVariantUsesManageCapability(t *testing.T) {
	gw := &fakeGateway{}
	client := NewClient(gw, &http.Client{Transport: fakeHTTPTransport{}})

	variant, err := client.CreateAssetVariant(context.Background(), CreateAssetVariantInput{
		TenantUUID: "tenant-001",
		UUID:       "media-asset-001",
		Variant:    "preview",
		Name:       "design.preview.jpg",
		MimeType:   "image/jpeg",
	})
	require.NoError(t, err)
	require.Equal(t, "media-asset-001", variant.AssetUUID)
	require.Equal(t, "preview", variant.Variant)

	ticket, err := client.PresignAssetVariant(context.Background(), PresignAssetVariantInput{
		TenantUUID:       "tenant-001",
		UUID:             "media-asset-001",
		Variant:          "preview",
		Action:           PresignActionUpload,
		Method:           http.MethodPut,
		ExpiresInSeconds: 900,
	})
	require.NoError(t, err)
	require.Equal(t, http.MethodPut, ticket.Method)

	require.Len(t, gw.calls, 2)
	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[0].CapabilityID)
	require.Equal(t, "CreateMediaAssetVariant", gw.calls[0].Action)
	createPayload, ok := gw.calls[0].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, createPayload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001/variants/preview", createPayload["endpoint"])
	createBody, ok := createPayload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "preview", createBody["variant"])
	require.Equal(t, "media-asset-001", createBody["uuid"])

	require.Equal(t, "PresignMediaAssetVariant", gw.calls[1].Action)
	presignPayload, ok := gw.calls[1].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodPost, presignPayload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001/variants/preview/presign", presignPayload["endpoint"])
	presignBody, ok := presignPayload["body"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "preview", presignBody["variant"])
	require.Equal(t, "upload", presignBody["action"])
}

func TestClientListAndDeleteAssetsUseTypedCatalogContract(t *testing.T) {
	gw := &fakeGateway{}
	client := NewClient(gw, &http.Client{Transport: fakeHTTPTransport{}})

	assets, err := client.ListAssets(context.Background(), ListAssetsInput{
		TenantUUID:       "tenant-001",
		Page:             2,
		PageSize:         50,
		Keyword:          "design",
		OwnerSubjectType: "ai_craft_track",
		OwnerSubjectID:   "track-001",
		Tags:             []string{"ai-craft"},
		BusinessStatuses: []BusinessStatus{BusinessStatusPublished},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), assets.Total)
	require.Equal(t, "media-asset-001", assets.Items[0].UUID)

	require.NoError(t, client.DeleteAsset(context.Background(), DeleteAssetInput{
		TenantUUID: "tenant-001",
		UUID:       "media-asset-001",
	}))

	require.Len(t, gw.calls, 2)
	require.Equal(t, CapabilityMediaAssetsRead, gw.calls[0].CapabilityID)
	require.Equal(t, "ListMediaAssets", gw.calls[0].Action)
	listPayload, ok := gw.calls[0].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodGet, listPayload["method"])
	require.Equal(t, "/api/v1/media/assets", listPayload["endpoint"])
	query, ok := listPayload["query"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ai_craft_track", query["ownerSubjectType"])
	require.Equal(t, []string{"published"}, query["businessStatus"])

	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[1].CapabilityID)
	require.Equal(t, "DeleteMediaAsset", gw.calls[1].Action)
	deletePayload, ok := gw.calls[1].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, http.MethodDelete, deletePayload["method"])
	require.Equal(t, "/api/v1/media/assets/media-asset-001", deletePayload["endpoint"])
}

type fakeGateway struct {
	calls []gateway.InvokeRequest
}

func (f *fakeGateway) Invoke(_ context.Context, req gateway.InvokeRequest) (*gateway.Response, error) {
	f.calls = append(f.calls, req)
	switch req.Action {
	case "CreateMediaAsset", "GetMediaAsset", "UpdateMediaAsset":
		return &gateway.Response{Data: map[string]any{
			"uuid":             "media-asset-001",
			"tenant_uuid":      "tenant-001",
			"name":             "design.png",
			"driver":           "local",
			"object_key":       "media-asset-001",
			"mime_type":        "image/png",
			"business_status":  "under_review",
			"owner_subject_id": "track/original/design.png",
		}}, nil
	case "ListMediaAssets":
		return &gateway.Response{Data: map[string]any{
			"items": []map[string]any{{
				"uuid":             "media-asset-001",
				"tenant_uuid":      "tenant-001",
				"name":             "design.png",
				"driver":           "local",
				"objectKey":        "media-asset-001",
				"mimeType":         "image/png",
				"businessStatus":   "published",
				"ownerSubjectType": "ai_craft_track",
			}},
			"total":    1,
			"page":     2,
			"pageSize": 50,
		}}, nil
	case "DeleteMediaAsset":
		return &gateway.Response{Data: map[string]any{"deleted": true}}, nil
	case "CreateMediaAssetVariant":
		return &gateway.Response{Data: map[string]any{
			"uuid":        "media-variant-001",
			"tenant_uuid": "tenant-001",
			"asset_uuid":  "media-asset-001",
			"variant":     "preview",
			"name":        "design.preview.jpg",
			"driver":      "local",
			"object_key":  "media-asset-001/preview",
			"mime_type":   "image/jpeg",
		}}, nil
	case "PresignMediaAsset":
		payload := requestBody(req)
		method := http.MethodGet
		if payload["action"] == "upload" {
			method = http.MethodPut
		}
		return &gateway.Response{Data: map[string]any{
			"url":                "https://media.example/object",
			"method":             method,
			"expires_in_seconds": 300,
			"headers":            map[string]string{"X-Test": "1"},
			"object_key":         "media-asset-001",
		}}, nil
	case "PresignMediaAssetVariant":
		payload := requestBody(req)
		method := http.MethodGet
		if payload["action"] == "upload" {
			method = http.MethodPut
		}
		return &gateway.Response{Data: map[string]any{
			"url":                "https://media.example/object-preview",
			"method":             method,
			"expires_in_seconds": 300,
			"headers":            map[string]string{"X-Test": "1"},
			"object_key":         "media-asset-001/preview",
		}}, nil
	default:
		return &gateway.Response{Data: map[string]any{}}, nil
	}
}

func requestBody(req gateway.InvokeRequest) map[string]any {
	payload, _ := req.Payload.(map[string]any)
	body, _ := payload["body"].(map[string]any)
	return body
}

type fakeHTTPTransport struct{}

func (fakeHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPut {
		_, _ = io.ReadAll(req.Body)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     http.Header{},
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Request:    req,
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte("download-body"))),
		Request:    req,
	}, nil
}
