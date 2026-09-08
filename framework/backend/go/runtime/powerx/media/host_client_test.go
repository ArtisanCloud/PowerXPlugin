package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostClientCreatesUUIDAssetWithSTS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/media/assets" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer service-sts" {
			t.Fatalf("Authorization = %q", got)
		}
		var in CreateAssetInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		if in.OwnerSubjectUUID != "member-uuid" || in.Name != "brief.pdf" {
			t.Fatalf("input = %#v", in)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"asset_uuid":"asset-uuid","name":"brief.pdf","mime_type":"application/pdf","size_bytes":3,"status":"pending_upload"}}`))
	}))
	defer server.Close()

	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	asset, err := client.CreateAsset(context.Background(), CreateAssetInput{Name: "brief.pdf", MimeType: "application/pdf", SizeBytes: 3, Checksum: "sha256:abc", OwnerSubjectUUID: "member-uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if asset.AssetUUID != "asset-uuid" || asset.Status != "pending_upload" {
		t.Fatalf("asset = %#v", asset)
	}
}

func TestHostClientPreservesCoreReasonCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"MEDIA_FORBIDDEN"}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "service-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetAsset(context.Background(), "asset-uuid")
	var hostErr *HostHTTPError
	if !errors.As(err, &hostErr) || hostErr.StatusCode != http.StatusForbidden || hostErr.ReasonCode != "MEDIA_FORBIDDEN" {
		t.Fatalf("error = %#v", err)
	}
}
