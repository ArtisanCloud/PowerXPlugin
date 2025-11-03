package app

import (
	"context"
	"strconv"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/config"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/grpc/client"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/logger"
	authx "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/middleware"
	adminmetrics "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/observability/admin_console"
	opsmetrics "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/observability/operations"
	marketplacesvc "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/marketplace"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Deps bundles shared infrastructure dependencies for handlers and services.
type Deps struct {
	DB                  *gorm.DB
	Ctx                 context.Context
	PowerXClient        *client.PowerXServiceClient
	Config              *config.Config
	TaxProviderClient   *marketplacesvc.TaxProviderClient
	MarketplaceBilling  marketplacesvc.BillingClient
	LicenseAuthority    marketplacesvc.LicenseAuthority
	LicenseCache        marketplacesvc.LicenseCache
	OperationsMetrics   *opsmetrics.Metrics
	AdminConsoleMetrics *adminmetrics.Metrics
}

// RuntimeDefaults returns the configured runtime ops defaults (if any).
func (d *Deps) RuntimeDefaults() *config.RuntimeOpsDefaults {
	if d == nil || d.Config == nil {
		return nil
	}
	return d.Config.RuntimeOps
}

// RuntimeLogger provides a structured logger enriched with runtime metadata.
func (d *Deps) RuntimeLogger(ctx context.Context, component string, extra logger.Fields) *logrus.Entry {
	if extra == nil {
		extra = logger.Fields{}
	}
	if ctx == nil && d != nil {
		ctx = d.Ctx
	}

	var tenantID string
	if tid, ok := authx.TenantIDFromContext(ctx); ok && tid > 0 {
		tenantID = strconv.FormatUint(tid, 10)
	}

	traceID := ""
	if ctx != nil {
		if v := ctx.Value("request_id"); v != nil {
			if s, ok := v.(string); ok {
				traceID = s
			}
		}
	}

	return logger.WithRuntimeFields(PluginID, tenantID, traceID, component, extra)
}
