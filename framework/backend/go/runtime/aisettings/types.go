package aisettings

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const (
	// CapabilityAISettingsAdminRead remains the contract used by the legacy
	// summary/profile methods.
	CapabilityAISettingsAdminRead = "com.corex.ai.settings.admin_read"
	// Catalog capabilities are formal Core business capabilities with explicit
	// API Key grants. They must not use route-derived generated capability IDs.
	CapabilityAISettingsCatalogProviders = "com.corex.ai.catalog.providers.read"
	CapabilityAISettingsCatalogModels    = "com.corex.ai.catalog.models.read"
)

type GatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
}

// Service is implemented by Core's client or a plugin-owned local settings
// provider. Bind the selected provider once during startup.
type Service interface {
	Summary(context.Context, string) (Summary, error)
	ProviderProfiles(context.Context, string) ([]ProviderProfile, error)
	ModelProfiles(context.Context, string) ([]ModelProfile, error)
	Routing(context.Context, string) (RoutingConfig, error)
	Health(context.Context, string) (HealthStatus, error)
}

// CatalogService is available only from the delegated PowerX client. Local
// mode intentionally uses the plugin-owned catalog configured at startup.
type CatalogService interface {
	CatalogProviders(context.Context, string, string) ([]CatalogProvider, error)
	CatalogModels(context.Context, string, string, string, string) ([]string, error)
}

type Config struct {
	Invoker    GatewayInvoker
	TenantUUID string
}

type Client struct {
	invoker    GatewayInvoker
	tenantUUID string
}

type Summary map[string]any
type ProviderProfile map[string]any
type ModelProfile map[string]any
type RoutingConfig map[string]any
type HealthStatus map[string]any
type CatalogProvider map[string]any
