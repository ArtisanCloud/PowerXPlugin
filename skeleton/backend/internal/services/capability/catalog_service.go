package capability

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

// CatalogService exposes read-only operations for capability catalog snapshots.
type CatalogService struct {
	manager capabilities.Manager
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
	return &CatalogService{manager: mgr}
}

// List returns the normalized capability entries from the catalog snapshot.
func (s *CatalogService) List(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	if s == nil || s.manager == nil {
		return nil, fmt.Errorf("capability catalog service not configured")
	}
	return s.manager.ListCapabilities(ctx)
}
