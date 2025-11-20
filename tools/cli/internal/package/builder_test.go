package pkg

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "plugin.yaml"), `
id: com.example.demo
version: 0.1.0
backend:
  entry: backend/cmd/plugin
`)

	contracts := filepath.Join(tmp, "docs", "contracts")
	mkDir(t, contracts)
	writeFile(t, filepath.Join(contracts, "manifest.json"), `{"name":"Demo","version":"0.1.0"}`)
	writeFile(t, filepath.Join(contracts, "rbac.json"), `{"rules":[]}`)

	webAdmin := filepath.Join(tmp, "web-admin")
	distDir := filepath.Join(webAdmin, "dist")
	mkDir(t, distDir)
	writeFile(t, filepath.Join(webAdmin, "package.json"), `{"name":"demo-web","scripts":{"build":"nuxt build"}}`)
	writeFile(t, filepath.Join(distDir, "index.html"), "<html>demo</html>")

	backendDir := filepath.Join(tmp, "backend")
	mkDir(t, backendDir)
	writeFile(t, filepath.Join(backendDir, "go.mod"), "module example.com/demo\n")

	fixed := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	builder := NewBuilder(
		WithClock(func() time.Time { return fixed }),
		WithFrontendBuild(func(ctx context.Context, opts *Options) error {
			// dist is pre-populated for tests
			return nil
		}),
		WithBackendBuild(func(ctx context.Context, opts *Options, output string) error {
			mkDir(t, filepath.Dir(output))
			return os.WriteFile(output, []byte("binary"), 0o755)
		}),
	)

	result, err := builder.Build(context.Background(), &Options{
		EntryPath:  tmp,
		CLIVersion: "test-cli",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result == nil {
		t.Fatalf("Build() result is nil")
	}
	if _, err := os.Stat(result.PackagePath); err != nil {
		t.Fatalf("package.tar.gz missing: %v", err)
	}
	if _, err := os.Stat(result.MetadataPath); err != nil {
		t.Fatalf("metadata.json missing: %v", err)
	}
	if result.DistHash == "" {
		t.Fatalf("expected non-empty dist hash")
	}
	if len(result.Artifacts) == 0 {
		t.Fatalf("artifacts should not be empty")
	}

	meta := Metadata{}
	data, err := os.ReadFile(result.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if want := "0.1.0"; meta.Version != want {
		t.Fatalf("metadata version = %s, want %s", meta.Version, want)
	}
	if meta.CLIVersion != "test-cli" {
		t.Fatalf("metadata cliVersion = %s", meta.CLIVersion)
	}
	if meta.DistHash != result.DistHash {
		t.Fatalf("metadata dist hash mismatch: %s vs %s", meta.DistHash, result.DistHash)
	}
	if meta.Channel != "dev" {
		t.Fatalf("expected default channel 'dev', got %s", meta.Channel)
	}
	if meta.Signature != nil {
		t.Fatalf("expected nil signature placeholder")
	}
}

func TestBuilderMissingManifest(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "plugin.yaml"), "id: com.example\nversion: 1.0.0\n")

	builder := NewBuilder(
		WithFrontendBuild(func(ctx context.Context, opts *Options) error { return nil }),
		WithBackendBuild(func(ctx context.Context, opts *Options, output string) error { return nil }),
	)

	_, err := builder.Build(context.Background(), &Options{EntryPath: tmp})
	if err == nil {
		t.Fatalf("expected error when manifest.json missing")
	}
	if !strings.Contains(err.Error(), "manifest.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuilderMissingFrontendDist(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "plugin.yaml"), "id: com.example.demo\nversion: 1.0.0\n")
	mkDir(t, filepath.Join(tmp, "docs", "contracts"))
	writeFile(t, filepath.Join(tmp, "docs", "contracts", "manifest.json"), `{"name":"Demo"}`)
	writeFile(t, filepath.Join(tmp, "docs", "contracts", "rbac.json"), `{"rules":[]}`)
	mkDir(t, filepath.Join(tmp, "web-admin"))
	writeFile(t, filepath.Join(tmp, "web-admin", "package.json"), `{"name":"demo"}`)
	mkDir(t, filepath.Join(tmp, "backend"))
	writeFile(t, filepath.Join(tmp, "backend", "go.mod"), "module example.com/demo\n")

	builder := NewBuilder(
		WithFrontendBuild(func(ctx context.Context, opts *Options) error { return nil }),
		WithBackendBuild(func(ctx context.Context, opts *Options, output string) error { return nil }),
	)

	_, err := builder.Build(context.Background(), &Options{EntryPath: tmp})
	if err == nil {
		t.Fatal("expected error when dist folder missing")
	}
	if !strings.Contains(err.Error(), "frontend build output not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mkDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
