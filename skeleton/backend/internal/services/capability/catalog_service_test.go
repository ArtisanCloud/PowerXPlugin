package capability

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

func TestCatalogServiceLookupDescriptorMeta(t *testing.T) {
	svc := &CatalogService{
		descriptorCache: make(map[string]*descriptorMetadata),
	}
	meta := svc.lookupDescriptorMeta("contracts/capabilities/com.powerx.plugins.base.template.list.yaml")
	if meta == nil {
		t.Fatalf("expected metadata from descriptor, got nil")
	}
	if meta.Protocols == nil || len(meta.Protocols) == 0 {
		t.Fatalf("expected protocols to be populated, got %+v", meta.Protocols)
	}
	if meta.Kind == "" {
		t.Fatalf("expected kind to be inferred, got empty")
	}
}

func TestListLocalCatalogDecoratesProtocols(t *testing.T) {
	root := findRepoRoot(t)
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	manager := capabilities.NewManager(nil, logger.WithField("component", "test"))
	svc := &CatalogService{
		manager:         manager,
		descriptorCache: make(map[string]*descriptorMetadata),
	}

	entries, err := svc.listLocalCatalog(context.Background())
	if err != nil {
		t.Fatalf("listLocalCatalog error: %v", err)
	}
	var target *capabilities.CatalogEntry
	for i := range entries {
		if entries[i].ID == "com.powerx.plugins.base.template.list" {
			target = &entries[i]
			break
		}
	}
	if target == nil {
		t.Fatalf("target capability not found in catalog")
	}
	rest, ok := target.Protocols["rest"].(map[string]interface{})
	if !ok || len(rest) == 0 {
		t.Fatalf("expected rest protocol to be populated, got %+v", target.Protocols["rest"])
	}
	if rest["path"] == "" && rest["endpoint"] == "" {
		t.Fatalf("rest protocol missing path/endpoint: %+v", rest)
	}
}

func TestResolveDescriptorPathFromNestedDir(t *testing.T) {
	const relPath = "contracts/capabilities/com.powerx.plugins.base.template.list.yaml"

	rootDir := findRepoRoot(t)
	absExpected := filepath.Join(rootDir, relPath)
	if _, err := os.Stat(absExpected); err != nil {
		t.Fatalf("expected descriptor at %s: %v", absExpected, err)
	}

	nested := filepath.Join(rootDir, "skeleton", "backend")
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("expected nested dir %s: %v", nested, err)
	}

	if err := os.Chdir(nested); err != nil {
		t.Fatalf("chdir nested: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(rootDir)
	})

	resolved := resolveDescriptorPath(relPath)
	if resolved == "" {
		t.Fatalf("resolveDescriptorPath returned empty result")
	}
	resolved = filepath.Clean(resolved)
	if !strings.HasSuffix(resolved, relPath) {
		t.Fatalf("resolved path %s does not end with %s", resolved, relPath)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root from %s", dir)
		}
		dir = parent
	}
}
