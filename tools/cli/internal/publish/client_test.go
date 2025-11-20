package publish

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientSubmitSuccess(t *testing.T) {
	tmp := t.TempDir()
	pkg := writeTempFile(t, tmp, "plugin.tar.gz", "binary-data")
	meta := writeTempFile(t, tmp, "metadata.json", `{"version":"0.1.0"}`)
	manifest := writeTempFile(t, tmp, "manifest.json", `{"name":"demo"}`)
	rbac := writeTempFile(t, tmp, "rbac.json", `{"rules":[]}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/plugins/releases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("missing auth header, got %s", got)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("pluginId") != "com.test.demo" {
			t.Fatalf("pluginId mismatch: %s", r.FormValue("pluginId"))
		}
		if r.FormValue("channel") != "beta" {
			t.Fatalf("channel mismatch: %s", r.FormValue("channel"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"publishId": "PUB-123",
				"reviewUrl": "https://powerx/publish/PUB-123",
				"status":    "pending",
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(Options{
		BaseURL:  server.URL,
		APIToken: "test-token",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	resp, err := client.Submit(context.Background(), &SubmitRequest{
		PluginID:     "com.test.demo",
		Version:      "0.1.0",
		Channel:      "beta",
		Notes:        "demo",
		PackagePath:  pkg,
		MetadataPath: meta,
		ManifestPath: manifest,
		RBACPath:     rbac,
		CLIVersion:   "test",
	})
	if err != nil {
		t.Fatalf("Submit error: %v", err)
	}
	if resp.PublishID != "PUB-123" {
		t.Fatalf("publish id mismatch: %s", resp.PublishID)
	}
	if resp.Status != "pending" {
		t.Fatalf("status mismatch: %s", resp.Status)
	}
	if resp.ReviewURL == "" {
		t.Fatal("expected review url")
	}
}

func TestClientSubmitEnvelopeError(t *testing.T) {
	tmp := t.TempDir()
	pkg := writeTempFile(t, tmp, "plugin.tar.gz", "binary-data")
	meta := writeTempFile(t, tmp, "metadata.json", "{}")
	manifest := writeTempFile(t, tmp, "manifest.json", "{}")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    409,
			"message": "duplicate",
		})
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.Submit(context.Background(), &SubmitRequest{
		PluginID:     "com.test.demo",
		Version:      "0.1.0",
		Channel:      "dev",
		PackagePath:  pkg,
		MetadataPath: meta,
		ManifestPath: manifest,
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestClientSubmitHTTPError(t *testing.T) {
	tmp := t.TempDir()
	pkg := writeTempFile(t, tmp, "plugin.tar.gz", "binary-data")
	meta := writeTempFile(t, tmp, "metadata.json", "{}")
	manifest := writeTempFile(t, tmp, "manifest.json", "{}")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.Submit(context.Background(), &SubmitRequest{
		PluginID:     "com.test.demo",
		Version:      "0.1.0",
		Channel:      "dev",
		PackagePath:  pkg,
		MetadataPath: meta,
		ManifestPath: manifest,
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected HTTP error, got %v", err)
	}
}

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
