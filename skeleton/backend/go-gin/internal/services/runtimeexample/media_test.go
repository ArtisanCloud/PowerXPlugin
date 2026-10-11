package runtimeexample

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocalMediaTransferLifecycle(t *testing.T) {
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	cross := authx.ContextWithTenantUUID(context.Background(), other)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	store := NewLocalMedia(server.URL)
	mux.HandleFunc(MediaTransferPath, store.ServeTransfer)
	body := []byte("media-bytes")
	checksum := fmt.Sprintf("%x", sha256.Sum256(body))
	asset, err := store.CreateAsset(ctx, dto.CreateAssetInput{Name: "test", MimeType: "application/octet-stream", SizeBytes: int64(len(body)), Checksum: checksum})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.GetAsset(cross, asset.AssetUUID); err == nil {
		t.Fatal("cross tenant")
	}
	if _, err = store.GetAsset(context.Background(), asset.AssetUUID); err == nil {
		t.Fatal("missing identity")
	}
	if _, err = store.CompleteUpload(ctx, asset.AssetUUID, dto.CompleteUploadInput{Checksum: checksum}); err == nil {
		t.Fatal("incomplete accepted")
	}
	transfer := func(ticket *dto.TransferTicket, b []byte, want int) []byte {
		t.Helper()
		req, _ := http.NewRequest(ticket.Method, ticket.URL, bytes.NewReader(b))
		for k, v := range ticket.Headers {
			req.Header.Set(k, v)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("status %d want %d", resp.StatusCode, want)
		}
		return out
	}
	upload, err := store.PresignUpload(ctx, asset.AssetUUID)
	if err != nil {
		t.Fatal(err)
	}
	transfer(upload, []byte("wrong"), 409)
	transfer(upload, body, 204)
	transfer(upload, body, 404)
	ready, err := store.CompleteUpload(ctx, asset.AssetUUID, dto.CompleteUploadInput{Checksum: checksum})
	if err != nil || ready.Status != "ready" {
		t.Fatal(ready, err)
	}
	download, err := store.PresignDownload(ctx, asset.AssetUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(transfer(download, nil, 200), body) {
		t.Fatal("bytes differ")
	}
	variant, err := store.CreateVariant(ctx, asset.AssetUUID, dto.CreateVariantInput{VariantType: "preview", MimeType: "application/octet-stream", SizeBytes: int64(len(body)), Checksum: checksum})
	if err != nil {
		t.Fatal(err)
	}
	vt, err := store.PresignVariantUpload(ctx, asset.AssetUUID, variant.VariantUUID, dto.VariantTicketInput{})
	if err != nil {
		t.Fatal(err)
	}
	transfer(vt, body, 204)
	v, err := store.CompleteVariantUpload(ctx, asset.AssetUUID, variant.VariantUUID, dto.CompleteUploadInput{Checksum: checksum})
	if err != nil || v.CompletedAt == nil || v.Status != "ready" {
		t.Fatal(v, err)
	}
	vd, err := store.PresignVariantDownload(ctx, asset.AssetUUID, variant.VariantUUID, dto.VariantTicketInput{})
	if err != nil {
		t.Fatal(err)
	}
	transfer(vd, nil, 200)
	if _, err = store.PresignVariantDownload(ctx, other, variant.VariantUUID, dto.VariantTicketInput{}); err == nil {
		t.Fatal("wrong asset")
	}
	if err = store.DeleteAsset(ctx, asset.AssetUUID); err != nil {
		t.Fatal(err)
	}
	transfer(download, nil, 404)
	transfer(vd, nil, 404)
}
func TestLocalMediaTicketExpiry(t *testing.T) {
	s := NewLocalMedia("http://localhost")
	now := time.Now()
	s.now = func() time.Time { return now }
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	sum := fmt.Sprintf("%x", sha256.Sum256(nil))
	a, err := s.CreateAsset(ctx, dto.CreateAssetInput{Name: "empty", MimeType: "text/plain", Checksum: sum})
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := s.PresignUpload(ctx, a.AssetUUID)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	req := httptest.NewRequest("PUT", ticket.URL, nil)
	req.Header.Set("X-Local-Media-Ticket", ticket.Headers["X-Local-Media-Ticket"])
	w := httptest.NewRecorder()
	s.ServeTransfer(w, req)
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
