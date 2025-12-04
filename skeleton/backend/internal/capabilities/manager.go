package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/sirupsen/logrus"
)

const (
	defaultCatalogPath  = "capabilities/catalog.json"
	catalogEnvKey       = "POWERX_CAPABILITY_CATALOG"
	exposureDir         = "contracts/exposure"
	agentSDKDir         = "dist/agent-sdk"
	protocolOpenAPIType = "openapi"
	protocolProtoType   = "proto"
	protocolWorkflow    = "workflow"
	protocolMCP         = "mcp_manifest"
	protocolAgentStream = "agent_stream"
)

// Manager exposes high-level capability operations for runtime bootstrap and host sync.
type Manager interface {
	ListCapabilities(ctx context.Context) ([]CatalogEntry, error)
	ExportProtocols(ctx context.Context) ([]ProtocolAsset, error)
	RegisterWithHost(ctx context.Context, client HostSyncClient) error
}

// HostSyncClient represents the client used to register capability catalogs with PowerX.
type HostSyncClient interface {
	RegisterCatalog(ctx context.Context, catalog *CatalogSnapshot, assets []ProtocolAsset) error
}

// CatalogLoader resolves catalog snapshots from manifest outputs.
type CatalogLoader interface {
	LoadCatalog(ctx context.Context) (*CatalogSnapshot, error)
}

// AssetExporter enumerates protocol assets (OpenAPI, Proto, Workflow, MCP manifests, etc.).
type AssetExporter interface {
	Export(ctx context.Context, catalog *CatalogSnapshot) ([]ProtocolAsset, error)
}

// CatalogSnapshot is persisted JSON emitted by scripts/capabilities/catalog-parser.ts.
type CatalogSnapshot struct {
	PluginID        string         `json:"plugin_id"`
	ManifestVersion string         `json:"manifest_version"`
	GeneratedAt     string         `json:"generated_at"`
	Imports         []string       `json:"imports"`
	Entries         []CatalogEntry `json:"entries"`
}

// CatalogEntry describes a single capability definition.
type CatalogEntry struct {
	ID         string                 `json:"id"`
	Version    string                 `json:"version"`
	Descriptor string                 `json:"descriptor"`
	Schemas    map[string]string      `json:"schemas"`
	Protocols  map[string]interface{} `json:"protocols"`
	Tags       []string               `json:"tags"`
	Execution  ExecutionConfig        `json:"execution"`
	Checksum   string                 `json:"checksum"`
}

// ExecutionConfig controls sync/async semantics of a capability.
type ExecutionConfig struct {
	Mode           string `json:"mode"`
	CallbackURL    string `json:"callback_url,omitempty"`
	SSEChannel     string `json:"sse_channel,omitempty"`
	StatusEndpoint string `json:"status_endpoint,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// ProtocolAsset captures generated exposure artefacts.
type ProtocolAsset struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type manager struct {
	cfg           *config.Config
	logger        *logrus.Entry
	catalogLoader CatalogLoader
	assetExporter AssetExporter
}

// NewManager builds a filesystem-based manager using repo defaults.
func NewManager(cfg *config.Config, log *logrus.Entry) Manager {
	catalogPath := os.Getenv(catalogEnvKey)
	if strings.TrimSpace(catalogPath) == "" {
		catalogPath = defaultCatalogPath
	}

	loader := &fileSystemCatalogLoader{path: catalogPath}
	exporter := &fileSystemAssetExporter{
		roots: []assetRoot{
			{dir: exposureDir, protocolType: protocolOpenAPIType, matcher: func(path string) bool {
				return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
			}},
			{dir: filepath.Join(exposureDir, "proto"), protocolType: protocolProtoType, matcher: func(path string) bool {
				return strings.HasSuffix(path, ".proto")
			}},
			{dir: filepath.Join(exposureDir, "workflow"), protocolType: protocolWorkflow, matcher: nil},
			{dir: filepath.Join(exposureDir, "mcp-tools"), protocolType: protocolMCP, matcher: nil},
			{dir: filepath.Join(exposureDir, "agent-streams"), protocolType: protocolAgentStream, matcher: nil},
			{dir: agentSDKDir, protocolType: "sdk", matcher: nil},
		},
	}

	return &manager{
		cfg:           cfg,
		logger:        log,
		catalogLoader: loader,
		assetExporter: exporter,
	}
}

func (m *manager) ListCapabilities(ctx context.Context) ([]CatalogEntry, error) {
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if err := normalizeEntries(catalog.Entries); err != nil {
		return nil, err
	}
	return catalog.Entries, nil
}

func (m *manager) ExportProtocols(ctx context.Context) ([]ProtocolAsset, error) {
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return m.assetExporter.Export(ctx, catalog)
}

func (m *manager) RegisterWithHost(ctx context.Context, client HostSyncClient) error {
	if client == nil {
		return errors.New("host sync client is nil")
	}
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return err
	}
	if err := normalizeEntries(catalog.Entries); err != nil {
		return err
	}
	assets, err := m.assetExporter.Export(ctx, catalog)
	if err != nil {
		return err
	}
	return client.RegisterCatalog(ctx, catalog, assets)
}

// fileSystemCatalogLoader loads catalog snapshots from disk.
type fileSystemCatalogLoader struct {
	path string
}

func (l *fileSystemCatalogLoader) LoadCatalog(ctx context.Context) (*CatalogSnapshot, error) {
	p := strings.TrimSpace(l.path)
	if p == "" {
		p = defaultCatalogPath
	}
	if !filepath.IsAbs(p) {
		cwd, _ := os.Getwd()
		p = filepath.Join(cwd, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CatalogSnapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Entries: []CatalogEntry{}}, nil
		}
		return nil, fmt.Errorf("capabilities: failed to read catalog %s: %w", p, err)
	}
	var snapshot CatalogSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("capabilities: invalid catalog JSON %s: %w", p, err)
	}
	if strings.TrimSpace(snapshot.GeneratedAt) == "" {
		snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if snapshot.Entries == nil {
		snapshot.Entries = []CatalogEntry{}
	}
	return &snapshot, nil
}

// fileSystemAssetExporter walks known roots and emits protocol assets.
type fileSystemAssetExporter struct {
	roots []assetRoot
}

type assetRoot struct {
	dir          string
	protocolType string
	matcher      func(path string) bool
}

func (e *fileSystemAssetExporter) Export(ctx context.Context, catalog *CatalogSnapshot) ([]ProtocolAsset, error) {
	var assets []ProtocolAsset
	for _, root := range e.roots {
		if strings.TrimSpace(root.dir) == "" {
			continue
		}
		files, err := walkFiles(root.dir, root.matcher)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			assets = append(assets, ProtocolAsset{Type: root.protocolType, Path: f})
		}
	}
	return assets, nil
}

func walkFiles(dir string, matcher func(string) bool) ([]string, error) {
	var results []string
	root := filepath.Clean(dir)
	cwd, _ := os.Getwd()
	absRoot := root
	if !filepath.IsAbs(absRoot) && cwd != "" {
		absRoot = filepath.Join(cwd, root)
	}
	err := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := path
		if cwd != "" {
			if r, err := filepath.Rel(cwd, path); err == nil {
				rel = r
			}
		}
		rel = filepath.ToSlash(rel)
		if matcher != nil && !matcher(rel) {
			return nil
		}
		results = append(results, rel)
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("capabilities: walk %s failed: %w", dir, err)
	}
	return results, nil
}

// ValidateExecution enforces sync/async constraints and populates defaults.
func ValidateExecution(entries []CatalogEntry) error {
	return normalizeEntries(entries)
}

func normalizeEntries(entries []CatalogEntry) error {
	for i := range entries {
		mode := strings.TrimSpace(strings.ToLower(entries[i].Execution.Mode))
		if mode == "" {
			entries[i].Execution.Mode = "sync"
			continue
		}
		switch mode {
		case "sync":
			entries[i].Execution.Mode = "sync"
		case "async":
			if strings.TrimSpace(entries[i].Execution.CallbackURL) == "" && strings.TrimSpace(entries[i].Execution.SSEChannel) == "" {
				return fmt.Errorf("capabilities: async capability %s missing callback or SSE channel", entries[i].ID)
			}
			if strings.TrimSpace(entries[i].Execution.StatusEndpoint) == "" {
				return fmt.Errorf("capabilities: async capability %s missing status_endpoint", entries[i].ID)
			}
			entries[i].Execution.Mode = "async"
		default:
			return fmt.Errorf("capabilities: capability %s has invalid execution mode %q", entries[i].ID, entries[i].Execution.Mode)
		}
	}
	return nil
}

// EnsureManager warms up capability catalog during startup and logs the result.
func EnsureManager(ctx context.Context, mgr Manager, log *logrus.Entry) error {
	if mgr == nil {
		return nil
	}
	if _, err := mgr.ListCapabilities(ctx); err != nil {
		return err
	}
	if _, err := mgr.ExportProtocols(ctx); err != nil {
		return err
	}
	if log != nil {
		log.Debug("capabilities: catalog verified")
	}
	return nil
}

func managerLogger(component string) *logrus.Entry {
	return logger.WithField("component", component)
}
