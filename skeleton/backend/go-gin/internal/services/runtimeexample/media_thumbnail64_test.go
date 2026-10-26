package runtimeexample

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"image"
	"image/png"
	"net/http/httptest"
	"testing"
)

func TestLocalExplicitThumbnail64RequiresValidBytesAndDimensions(t *testing.T) {
	for _, width := range []int{64, 65} {
		var body bytes.Buffer
		if err := png.Encode(&body, image.NewNRGBA(image.Rect(0, 0, width, 36))); err != nil {
			t.Fatal(err)
		}
		data := body.Bytes()
		checksum := fmt.Sprintf("%x", sha256.Sum256(data))
		ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
		store := NewLocalMedia("https://plugin.invalid")
		asset, err := store.CreateAsset(ctx, dto.CreateAssetInput{Name: "fixture.png", MimeType: "image/png", SizeBytes: int64(len(data)), Checksum: checksum})
		if err != nil {
			t.Fatal(err)
		}
		variant, err := store.CreateVariant(ctx, asset.AssetUUID, dto.CreateVariantInput{VariantType: dto.VariantThumbnail64, MimeType: "image/png", SizeBytes: int64(len(data)), Checksum: checksum})
		if err != nil {
			t.Fatal(err)
		}
		ticket, err := store.PresignVariantUpload(ctx, asset.AssetUUID, variant.VariantUUID, dto.VariantTicketInput{})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(ticket.Method, ticket.URL, bytes.NewReader(data))
		for key, value := range ticket.Headers {
			request.Header.Set(key, value)
		}
		response := httptest.NewRecorder()
		store.ServeTransfer(response, request)
		if response.Code != 204 {
			t.Fatal("LOCAL_THUMBNAIL_TRANSFER_FAILED")
		}
		ready, err := store.CompleteVariantUpload(ctx, asset.AssetUUID, variant.VariantUUID, dto.CompleteUploadInput{Checksum: checksum})
		if width == 64 {
			if err != nil || ready.Status != "ready" || ready.MimeType != "image/png" || ready.SizeBytes != int64(len(data)) {
				t.Fatal("VALID_LOCAL_THUMBNAIL_REJECTED")
			}
		} else {
			if err == nil || ready != nil {
				t.Fatal("OVERSIZED_LOCAL_THUMBNAIL_ACCEPTED")
			}
		}
	}
}
