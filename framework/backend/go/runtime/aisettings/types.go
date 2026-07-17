package aisettings

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const CapabilityAISettingsAdminRead = "com.corex.ai.settings.admin_read"

type GatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
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
