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
	manager            capabilities.Manager
	descriptorCache    map[string]*descriptorMetadata
	descriptorCacheMux sync.RWMutex
}

type descriptorMetadata struct {
	Kind      string
	Protocols map[string]interface{}
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
		manager:         mgr,
		descriptorCache: make(map[string]*descriptorMetadata),
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
		meta := s.lookupDescriptorMeta(entries[i].Descriptor)
		if meta != nil {
			if meta.Kind != "" {
				entries[i].Kind = meta.Kind
			}
			if len(entries[i].Protocols) == 0 && len(meta.Protocols) > 0 {
				entries[i].Protocols = cloneProtocolMap(meta.Protocols)
			}
		}
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

func (s *CatalogService) lookupDescriptorMeta(path string) *descriptorMetadata {
	if s == nil {
		return nil
	}
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return nil
	}
	normalized = filepath.Clean(normalized)

	s.descriptorCacheMux.RLock()
	if v, ok := s.descriptorCache[normalized]; ok {
		s.descriptorCacheMux.RUnlock()
		return v
	}
	s.descriptorCacheMux.RUnlock()

	data, err := os.ReadFile(normalized)
	if err != nil {
		logger.WithError(err).WithField("descriptor", normalized).Debug("failed to read capability descriptor for metadata inference")
		return nil
	}
	var manifest struct {
		Type     string `yaml:"type"`
		Metadata struct {
			Protocols map[string]interface{} `yaml:"protocols"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		logger.WithError(err).WithField("descriptor", normalized).Debug("failed to parse capability descriptor for metadata inference")
		return nil
	}

	meta := &descriptorMetadata{
		Kind:      classifyCapabilityKind(manifest.Type, manifest.Metadata.Protocols),
		Protocols: manifest.Metadata.Protocols,
	}

	s.descriptorCacheMux.Lock()
	s.descriptorCache[normalized] = meta
	s.descriptorCacheMux.Unlock()
	return meta
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

func cloneProtocolMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
