package capability

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
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

type fakeCatalogManager struct {
	entries []capabilities.CatalogEntry
	err     error
}

func (m *fakeCatalogManager) ListCapabilities(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]capabilities.CatalogEntry(nil), m.entries...), nil
}

func (m *fakeCatalogManager) ExportProtocols(ctx context.Context) ([]capabilities.ProtocolAsset, error) {
	return nil, nil
}

func (m *fakeCatalogManager) ExportCatalog(ctx context.Context) (*capabilities.CatalogSnapshot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &capabilities.CatalogSnapshot{Entries: append([]capabilities.CatalogEntry(nil), m.entries...)}, nil
}

func (m *fakeCatalogManager) RegisterWithHost(ctx context.Context, client capabilities.HostSyncClient) error {
	return nil
}

type fakeGatewayClient struct {
	enabled bool
	records []gateway.PlatformCapabilityCatalogRecord
	err     error
}

func (g *fakeGatewayClient) Enabled() bool { return g.enabled }

func (g *fakeGatewayClient) Invoke(ctx context.Context, params gateway.InvokeParams) (*gateway.InvokeResult, error) {
	return nil, nil
}

func (g *fakeGatewayClient) ListPlatformCapabilityCatalog(ctx context.Context, opts gateway.ListPlatformCapabilityCatalogOptions) ([]gateway.PlatformCapabilityCatalogRecord, error) {
	if g.err != nil {
		return nil, g.err
	}
	return append([]gateway.PlatformCapabilityCatalogRecord(nil), g.records...), nil
}

func (g *fakeGatewayClient) Close() error { return nil }

type fakeCapabilityRegistry struct {
	items []powerxcapability.Capability
	err   error
}

func (f *fakeCapabilityRegistry) List(context.Context, powerxcapability.ListInput) ([]powerxcapability.Capability, error) {
	return append([]powerxcapability.Capability(nil), f.items...), f.err
}
func (*fakeCapabilityRegistry) GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	return nil, nil
}
func (*fakeCapabilityRegistry) Resolve(context.Context, powerxcapability.ResolveInput) (*powerxcapability.ResolveResult, error) {
	return nil, nil
}
func (*fakeCapabilityRegistry) Invoke(context.Context, powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error) {
	return nil, nil
}
func (*fakeCapabilityRegistry) GetInvocation(context.Context, string) (*powerxcapability.Invocation, error) {
	return nil, nil
}

func TestCatalogServiceCoreXContractUsesAuthorizedFormalContract(t *testing.T) {
	svc := &CatalogService{capabilityRegistry: &fakeCapabilityRegistry{items: []powerxcapability.Capability{{
		CapabilityID: "com.corex.media.assets.read",
		Source:       "corex",
		Protocols: []powerxcapability.Protocol{{
			Channel: "rest", Method: "GET", Endpoint: "/api/v1/media/assets",
		}},
	}}}}

	entry, err := svc.CoreXContract(context.Background(), "com.corex.media.assets.read")
	if err != nil {
		t.Fatalf("CoreXContract error: %v", err)
	}
	rest, ok := entry.Protocols["rest"].([]map[string]interface{})
	if !ok || len(rest) != 1 {
		t.Fatalf("rest contract=%#v", entry.Protocols["rest"])
	}
	if rest[0]["method"] != "GET" || rest[0]["endpoint"] != "/api/v1/media/assets" {
		t.Fatalf("formal REST contract=%#v", rest[0])
	}
}

func TestCatalogServiceListSourceAllMergesCorexAndPlugin(t *testing.T) {
	svc := &CatalogService{
		manager: &fakeCatalogManager{
			entries: []capabilities.CatalogEntry{
				{ID: "com.powerx.plugins.base.template.list"},
				{ID: "com.shared.dup"},
			},
		},
		gateway: &fakeGatewayClient{
			enabled: true,
			records: []gateway.PlatformCapabilityCatalogRecord{
				{CapabilityID: "com.corex.media.assets.read"},
				{CapabilityID: "com.shared.dup"},
			},
		},
		descriptorCache: make(map[string]*descriptorMetadata),
	}

	entries, err := svc.List(context.Background(), ListOptions{Source: "all"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ids := map[string]int{}
	for _, entry := range entries {
		ids[entry.ID]++
	}

	if ids["com.corex.media.assets.read"] != 1 {
		t.Fatalf("expected corex entry in merged result, got %+v", ids)
	}
	if ids["com.powerx.plugins.base.template.list"] != 1 {
		t.Fatalf("expected plugin entry in merged result, got %+v", ids)
	}
	if ids["com.shared.dup"] != 1 {
		t.Fatalf("expected duplicate entry to be deduplicated, got %+v", ids)
	}
}

func TestCatalogServicePlatformEntryUsesSafeCatalogMetadata(t *testing.T) {
	svc := &CatalogService{}
	entries := svc.fromPlatformCatalogRecords([]gateway.PlatformCapabilityCatalogRecord{{
		CapabilityID: "com.corex.media.assets.read",
		Title:        "Media Assets Read",
		Description:  "Read media assets",
		Source:       "corex",
		Categories:   []string{"media"},
	}})
	if len(entries) != 1 {
		t.Fatalf("entries=%d, want 1", len(entries))
	}
	if entries[0].Title != "Media Assets Read" || entries[0].Description != "Read media assets" {
		t.Fatalf("safe catalog metadata was not preserved: %+v", entries[0])
	}
	if entries[0].Source != "corex" {
		t.Fatalf("source=%q", entries[0].Source)
	}
	if len(entries[0].Protocols) != 0 {
		t.Fatalf("catalog entry must not expose protocols: %+v", entries[0].Protocols)
	}
}

func TestCatalogServiceListSourceAnyEqualsAll(t *testing.T) {
	svc := &CatalogService{
		manager: &fakeCatalogManager{
			entries: []capabilities.CatalogEntry{{ID: "com.powerx.plugins.base.template.list"}},
		},
		gateway: &fakeGatewayClient{
			enabled: true,
			records: []gateway.PlatformCapabilityCatalogRecord{{CapabilityID: "com.corex.media.assets.read"}},
		},
		descriptorCache: make(map[string]*descriptorMetadata),
	}

	entries, err := svc.List(context.Background(), ListOptions{Source: "any"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected merged entries for source=any, got %d", len(entries))
	}
}

func TestCatalogServiceListSourceAllReturnsGatewayFailure(t *testing.T) {
	svc := &CatalogService{
		manager: &fakeCatalogManager{
			entries: []capabilities.CatalogEntry{{ID: "com.powerx.plugins.base.template.list"}},
		},
		gateway: &fakeGatewayClient{
			enabled: true,
			err:     errors.New("gateway down"),
		},
		descriptorCache: make(map[string]*descriptorMetadata),
	}

	_, err := svc.List(context.Background(), ListOptions{Source: "all"})
	if err == nil {
		t.Fatal("expected gateway failure for source=all")
	}
}
