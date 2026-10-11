package media

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStandaloneMediaAPIKeyIsNotBearer(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "ApiKey fixture-key" {
			t.Error("wrong authentication scheme")
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"reason_code":"fixture.denied"}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "fixture-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateAsset(context.Background(), CreateAssetInput{Name: "fixture"}); err == nil {
		t.Fatal("upstream denial was hidden")
	}
	if calls != 1 {
		t.Fatalf("unexpected retries: %d", calls)
	}
	if _, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "", nil); err == nil {
		t.Fatal("missing API key accepted")
	}
	if _, err := NewHostClientWithTokenProvider(HostClientConfig{BaseURL: server.URL}, nil, nil); err == nil {
		t.Fatal("missing STS accepted")
	}
}

func TestMediaPresignSendsJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("JSON content type required")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "{}" {
			t.Errorf("presign body = %q, err = %v", body, err)
		}
		_, _ = w.Write([]byte(`{"data":{"url":"https://storage.invalid/file","method":"PUT"}}`))
	}))
	defer server.Close()
	client, err := NewHostClientWithAPIKey(HostClientConfig{BaseURL: server.URL}, "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PresignUpload(context.Background(), "fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.PresignDownload(context.Background(), "fixture"); err != nil {
		t.Fatal(err)
	}
}
