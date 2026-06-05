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
		OwnerSubjectType: "ai_craft_asset",
		OwnerSubjectID:   "track/original/design.png",
		UploadChannel:    UploadChannelPresigned,
		Tags:             []string{"ai-craft"},
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
	require.Equal(t, "grpc", gw.calls[0].PreferredProtocol)
	require.Equal(t, "tenant-001", gw.calls[0].TenantUUID)
	createPayload, ok := gw.calls[0].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "presign_upload", createPayload["upload_channel"])
	require.Equal(t, "ai_craft_asset", createPayload["owner_subject_type"])

	require.Equal(t, CapabilityMediaAssetsManage, gw.calls[1].CapabilityID)
	require.Equal(t, "PresignMediaAsset", gw.calls[1].Action)
	presignPayload, ok := gw.calls[1].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "upload", presignPayload["action"])
	require.Equal(t, http.MethodPut, presignPayload["method"])

	require.Equal(t, CapabilityMediaAssetsRead, gw.calls[2].CapabilityID)
	require.Equal(t, "UpdateMediaAsset", gw.calls[2].Action)
	updatePayload, ok := gw.calls[2].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "under_review", updatePayload["business_status"])
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
	require.Equal(t, "download", payload["action"])
	require.Equal(t, http.MethodGet, payload["method"])
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
	require.Equal(t, "preview", createPayload["variant"])
	require.Equal(t, "media-asset-001", createPayload["uuid"])

	require.Equal(t, "PresignMediaAssetVariant", gw.calls[1].Action)
	presignPayload, ok := gw.calls[1].Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "preview", presignPayload["variant"])
	require.Equal(t, "upload", presignPayload["action"])
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
		payload, _ := req.Payload.(map[string]any)
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
		payload, _ := req.Payload.(map[string]any)
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
