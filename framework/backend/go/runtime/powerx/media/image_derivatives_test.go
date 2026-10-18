package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const derivativeAssetUUID = "00000000-0000-4000-8000-000000000001"

func TestHostAssetDerivativeSnapshotOperations(t *testing.T) {
	fixture, err := os.ReadFile("testdata/host_asset_derivatives.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, auth := range []string{"sts", "api_key"} {
		for _, operation := range []string{"create", "complete", "get"} {
			t.Run(auth+"/"+operation, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					method, path := http.MethodPost, "/api/v1/tenant/media/assets"
					if operation == "complete" {
						path += "/" + derivativeAssetUUID + "/complete-upload"
					} else if operation == "get" {
						method, path = http.MethodGet, path+"/"+derivativeAssetUUID
					}
					credential := "Bearer test-sts"
					if auth == "api_key" {
						credential = "ApiKey test-key"
					}
					if r.Method != method || r.URL.Path != path || r.Header.Get("Authorization") != credential || r.Header.Get("X-Tenant-UUID") != "" {
						t.Errorf("request=%s %s credential_boundary=%t", r.Method, r.URL.Path, r.Header.Get("Authorization") == credential)
					}
					if operation != "get" {
						var input map[string]any
						if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
							t.Error(err)
						}
						if input["checksum"] != "sha256:"+strings.Repeat("a", 64) || input["tenant_uuid"] != nil || input["derivatives"] != nil {
							t.Errorf("input=%v", input)
						}
					}
					_, _ = w.Write(fixture)
				}))
				defer server.Close()
				var client *HostClient
				var err error
				if auth == "api_key" {
					client, err = NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "test-key", server.Client())
				} else {
					client, err = NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "test-sts", nil }), server.Client())
				}
				if err != nil {
					t.Fatal(err)
				}
				var asset *HostAsset
				switch operation {
				case "create":
					asset, err = client.CreateAsset(context.Background(), CreateAssetInput{Name: "fixture.png", MimeType: "image/png", SizeBytes: 4096, Checksum: "sha256:" + strings.Repeat("a", 64)})
				case "complete":
					asset, err = client.CompleteUpload(context.Background(), derivativeAssetUUID, CompleteUploadInput{Checksum: "sha256:" + strings.Repeat("a", 64)})
				case "get":
					asset, err = client.GetAsset(context.Background(), derivativeAssetUUID)
				}
				if err != nil {
					t.Fatal(err)
				}
				if calls != 1 || asset.AssetUUID != derivativeAssetUUID || asset.Status != "ready" || asset.MimeType != "image/png" || asset.SizeBytes != 4096 {
					t.Fatalf("asset=%+v calls=%d", asset, calls)
				}
				d := asset.Derivatives
				if d == nil || d.AssetUUID != asset.AssetUUID || d.UploadState != "ready" || d.Status != "processing" || d.Recipe != "standard-image-v1" || d.JobUUID != "00000000-0000-4000-8000-000000000002" || d.Attempts != 1 || d.ErrorCode != "" || d.Retryable {
					t.Fatalf("derivatives=%+v", d)
				}
				if len(d.Items) != 2 || d.Items[0].Variant != "preview" || d.Items[0].Status != "ready" || d.Items[0].VariantUUID != "00000000-0000-4000-8000-000000000003" || d.Items[1].Variant != "thumbnail" || d.Items[1].Status != "processing" || d.Items[1].VariantUUID != "" {
					t.Fatalf("items=%+v", d.Items)
				}
			})
		}
	}
}

func TestHostAssetDerivativeOptionalAndFailureStates(t *testing.T) {
	for _, tc := range []struct {
		name, derivative string
	}{
		{"absent", ""},
		{"null", `,"derivatives":null`},
		{"queued", `,"derivatives":{"status":"queued","upload_state":"ready","attempts":0,"retryable":false,"items":[]}`},
		{"failed", `,"derivatives":{"status":"failed","upload_state":"ready","error_code":"MEDIA_IMAGE_DECODE_FAILED","attempts":3,"retryable":true,"items":[]}`},
		{"unsupported", `,"derivatives":{"status":"unsupported","upload_state":"ready","items":[]}`},
		{"unavailable", `,"derivatives":{"status":"unavailable","upload_state":"ready","error_code":"MEDIA_UPSTREAM_DEPENDENCY","items":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"data":{"asset_uuid":%q,"status":"ready"%s}}`, derivativeAssetUUID, tc.derivative)
			}))
			defer server.Close()
			client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "test-key", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			asset, err := client.GetAsset(context.Background(), derivativeAssetUUID)
			if err != nil || asset.Status != "ready" {
				t.Fatalf("asset=%+v err=%v", asset, err)
			}
			encoded, err := json.Marshal(asset)
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "absent" || tc.name == "null" {
				if asset.Derivatives != nil || bytes.Contains(encoded, []byte(`"derivatives"`)) {
					t.Fatalf("asset=%s", encoded)
				}
				return
			}
			if asset.Derivatives == nil || asset.Derivatives.Status != tc.name {
				t.Fatalf("derivatives=%+v", asset.Derivatives)
			}
			if tc.name == "failed" && (asset.Derivatives.ErrorCode != "MEDIA_IMAGE_DECODE_FAILED" || asset.Derivatives.Attempts != 3 || !asset.Derivatives.Retryable) {
				t.Fatalf("derivatives=%+v", asset.Derivatives)
			}
		})
	}
}

func TestHostAssetRejectsMalformedDerivatives(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"data":{"asset_uuid":%q,"status":"ready","derivatives":"ready"}}`, derivativeAssetUUID)
	}))
	defer server.Close()
	client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "test-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetAsset(context.Background(), derivativeAssetUUID)
	var hostErr *HostHTTPError
	if !errors.As(err, &hostErr) || hostErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("err=%v", err)
	}
}

func TestVariantReusePreservesReturnedMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/media/assets/"+derivativeAssetUUID+"/variants" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		fmt.Fprintf(w, `{"data":{"asset_uuid":%q,"variant_uuid":"00000000-0000-4000-8000-000000000003","mime_type":"image/png","size_bytes":512,"status":"ready","completed_at":"2026-10-18T00:00:00Z"}}`, derivativeAssetUUID)
	}))
	defer server.Close()
	client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "test-key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	variant, err := client.CreateVariant(context.Background(), derivativeAssetUUID, CreateVariantInput{VariantType: "preview", MimeType: "image/jpeg", SizeBytes: 1024, Checksum: strings.Repeat("a", 64)})
	if err != nil || variant.MimeType != "image/png" || variant.SizeBytes != 512 || variant.Status != "ready" || variant.CompletedAt == nil {
		t.Fatalf("variant=%+v err=%v", variant, err)
	}
}
