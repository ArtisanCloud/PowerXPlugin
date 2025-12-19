package capability

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"gopkg.in/yaml.v3"
)

// CatalogService exposes read-only operations for capability catalog snapshots.
type CatalogService struct {
	manager      capabilities.Manager
	typeCache    map[string]string
	typeCacheMux sync.RWMutex
}

// NewCatalogService builds a catalog service when a capabilities manager is available.
func NewCatalogService(deps *app.Deps) *CatalogService {
	if deps == nil {
		return nil
	}
	mgr := deps.CapabilitiesManager
	if mgr == nil {
		log := logger.WithField("component", "capability_catalog_service")
		mgr = capabilities.NewManager(deps.Config, log)
	}
	if mgr == nil {
		return nil
	}
	return &CatalogService{
		manager:   mgr,
		typeCache: make(map[string]string),
	}
}

// List returns the normalized capability entries from the catalog snapshot.
func (s *CatalogService) List(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	if s == nil || s.manager == nil {
		return nil, fmt.Errorf("capability catalog service not configured")
	}
	entries, err := s.manager.ListCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	return s.decorate(entries), nil
}

func (s *CatalogService) decorate(entries []capabilities.CatalogEntry) []capabilities.CatalogEntry {
	for i := range entries {
		entries[i].Module = deriveCapabilityModule(entries[i].ID)
		entries[i].Kind = s.lookupDescriptorKind(entries[i].Descriptor)
	}
	return entries
}

func deriveCapabilityModule(id string) string {
	parts := strings.Split(strings.TrimSpace(id), ".")
	if len(parts) <= 1 {
		return strings.TrimSpace(id)
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

func (s *CatalogService) lookupDescriptorKind(path string) string {
	if s == nil {
		return ""
	}
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return ""
	}
	normalized = filepath.Clean(normalized)

	s.typeCacheMux.RLock()
	if v, ok := s.typeCache[normalized]; ok {
		s.typeCacheMux.RUnlock()
		return v
	}
	s.typeCacheMux.RUnlock()

	data, err := os.ReadFile(normalized)
	if err != nil {
		logger.WithError(err).WithField("descriptor", normalized).Debug("failed to read capability descriptor for type inference")
		return ""
	}
	var meta struct {
		Type     string `yaml:"type"`
		Metadata struct {
			Protocols map[string]interface{} `yaml:"protocols"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		logger.WithError(err).WithField("descriptor", normalized).Debug("failed to parse capability descriptor for type inference")
		return ""
	}
	kind := classifyCapabilityKind(meta.Type, meta.Metadata.Protocols)

	s.typeCacheMux.Lock()
	s.typeCache[normalized] = kind
	s.typeCacheMux.Unlock()
	return kind
}

func classifyCapabilityKind(declared string, protocols map[string]interface{}) string {
	if normalized := normalizeKindName(declared); normalized != "" {
		return normalized
	}

	if hasWorkflowProtocol(protocols) {
		return "Workflow"
	}
	return "Capability"
}

func normalizeKindName(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "":
		return ""
	case "workflow":
		return "Workflow"
	case "tool":
		return "Tool"
	case "api", "capability":
		return "Capability"
	default:
		return strings.Title(value)
	}
}

func hasWorkflowProtocol(protocols map[string]interface{}) bool {
	if len(protocols) == 0 {
		return false
	}
	if v, ok := protocols["workflow"]; ok && v != nil {
		return true
	}
	return false
}
