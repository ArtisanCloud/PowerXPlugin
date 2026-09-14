package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVariantTransferContract(t *testing.T) {
	const asset = "00000000-0000-4000-8000-000000000001"
	const variant = "00000000-0000-4000-8000-000000000002"
	checksum := strings.Repeat("a", 64)
	for _, action := range []string{"presign-upload", "complete-upload", "presign-download"} {
		for _, status := range []int{200, 400, 401, 403, 404, 409, 422, 503} {
			t.Run(fmt.Sprintf("%s/%d", action, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" || r.URL.Path != variantPath(asset, variant)+"/"+action || r.Header.Get("Authorization") != "Bearer test-sts" || r.Header.Get("X-Tenant-UUID") != "" {
						t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body == nil {
						t.Fatalf("body=%v err=%v", body, err)
					}
					if action == "complete-upload" {
						if len(body) != 1 || body["checksum"] != checksum {
							t.Errorf("body=%v", body)
						}
					} else if len(body) != 0 {
						t.Errorf("body=%v", body)
					}
					w.WriteHeader(status)
					if status != 200 {
						fmt.Fprint(w, `{"reason_code":"MEDIA_TEST_REJECTED"}`)
						return
					}
					if action == "complete-upload" {
						fmt.Fprintf(w, `{"data":{"asset_uuid":%q,"variant_uuid":%q,"status":"ready","completed_at":"2026-09-08T00:00:00Z"}}`, asset, variant)
					} else {
						method := "PUT"
						if action == "presign-download" {
							method = "GET"
						}
						fmt.Fprintf(w, `{"data":{"url":"/api/v1/media/transfers/test?ticket=opaque","method":%q,"headers":{"Content-Type":"image/png"},"expires_at":"2026-09-08T00:15:00Z"}}`, method)
					}
				}))
				defer server.Close()
				c, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "test-sts", nil }), server.Client())
				if err != nil {
					t.Fatal(err)
				}
				switch action {
				case "presign-upload":
					_, err = c.PresignVariantUpload(context.Background(), asset, variant, VariantTicketInput{})
				case "presign-download":
					_, err = c.PresignVariantDownload(context.Background(), asset, variant, VariantTicketInput{})
				case "complete-upload":
					_, err = c.CompleteVariantUpload(context.Background(), asset, variant, CompleteUploadInput{Checksum: checksum})
				}
				if status == 200 {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var target *HostHTTPError
				if !errors.As(err, &target) || target.StatusCode != status || target.ReasonCode != "MEDIA_TEST_REJECTED" {
					t.Fatalf("err=%v", err)
				}
			})
		}
	}
}

func TestVariantInvalidTTLAndCancelledToken(t *testing.T) {
	calls := 0
	c, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: "http://unused.invalid"}, HostTokenProviderFunc(func(ctx context.Context) (string, error) { calls++; return "", ctx.Err() }), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ttl := range []int64{-1, 59, 3601} {
		_, err := c.PresignVariantUpload(context.Background(), "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002", VariantTicketInput{ExpiresInSeconds: ttl})
		var target *HostHTTPError
		if !errors.As(err, &target) || target.StatusCode != 400 || calls != 0 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.PresignVariantDownload(ctx, "asset", "variant", VariantTicketInput{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestVariantCreateChecksumAndCompleteValidation(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	checksum := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["checksum"] != checksum {
			t.Errorf("body=%v", body)
		}
		fmt.Fprintf(w, `{"data":{"asset_uuid":%q,"variant_uuid":%q,"status":"pending_upload","completed_at":null}}`, id, id)
	}))
	defer server.Close()
	c, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.CreateVariant(context.Background(), id, CreateVariantInput{VariantType: "preview", Checksum: checksum, MimeType: "image/png", SizeBytes: 1})
	if err != nil || v.Status != "pending_upload" || v.CompletedAt != nil {
		t.Fatalf("variant=%v err=%v", v, err)
	}
	v, err = c.CompleteVariantUpload(context.Background(), id, id, CompleteUploadInput{Checksum: checksum})
	var target *HostHTTPError
	if v != nil || !errors.As(err, &target) || target.StatusCode != 502 {
		t.Fatalf("variant=%v err=%v", v, err)
	}
}

func TestVariantRejectsInvalidInputsBeforeCredentialAccess(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	calls := 0
	c, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: "http://unused.invalid"}, HostTokenProviderFunc(func(context.Context) (string, error) { calls++; return "sts", nil }), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "1", "../resource", "00000000-0000-0000-0000-000000000000", "00000000000040008000000000000001"} {
		for _, pair := range [][2]string{{value, id}, {id, value}} {
			_, err := c.PresignVariantUpload(context.Background(), pair[0], pair[1], VariantTicketInput{})
			var target *HostHTTPError
			if !errors.As(err, &target) || target.StatusCode != 400 {
				t.Fatalf("err=%v", err)
			}
		}
	}
	for _, checksum := range []string{"", "sha256:abc", strings.Repeat("z", 64), strings.Repeat("a", 62)} {
		_, err := c.CompleteVariantUpload(context.Background(), id, id, CompleteUploadInput{Checksum: checksum})
		var target *HostHTTPError
		if !errors.As(err, &target) || target.StatusCode != 400 {
			t.Fatalf("err=%v", err)
		}
	}
	if calls != 0 {
		t.Fatalf("credential_calls=%d", calls)
	}
}

func TestVariantTicketRejectsInvalidSuccessResponse(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, body := range []string{
		`{"data":{"url":"/ticket","method":"GET","expires_at":"2026-09-08T00:00:00Z"}}`,
		`{"data":{"url":"/ticket","method":"PUT","expires_at":"invalid"}}`,
		`{"data":null}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, HostTokenProviderFunc(func(context.Context) (string, error) { return "sts", nil }), server.Client())
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.PresignVariantUpload(context.Background(), id, id, VariantTicketInput{})
		server.Close()
		var target *HostHTTPError
		if !errors.As(err, &target) || target.StatusCode != 502 {
			t.Fatalf("err=%v", err)
		}
	}
}
