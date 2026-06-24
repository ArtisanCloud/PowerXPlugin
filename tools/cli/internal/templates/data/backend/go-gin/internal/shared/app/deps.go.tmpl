package app

import (
	"context"
	"os"
	"strings"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/grpc/client"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/admin_console"
	capmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/capability"
	opsmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/operations"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/marketplace"
	"gorm.io/gorm"

	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
)

type DelegatedAuthProxy interface {
	Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*iamservice.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
}

// Deps bundles shared infrastructure dependencies for handlers and services.
type Deps struct {
	DB                  *gorm.DB
	Ctx                 context.Context
	PowerXClient        *client.PowerXServiceClient
	CapabilityGateway   gatewayClient
	Config              *config.Config
	CapabilitiesManager capabilities.Manager
	CapabilityMetrics   *capmetrics.Metrics
	TaxProviderClient   *marketplacesvc.TaxProviderClient
	MarketplaceBilling  marketplacesvc.BillingClient
	LicenseAuthority    marketplacesvc.LicenseAuthority
	LicenseCache        marketplacesvc.LicenseCache
	OperationsMetrics   *opsmetrics.Metrics
	AdminConsoleMetrics *adminmetrics.Metrics
	EventEmitter        fweventbridge.Emitter
	WSBusHub            fwwsbus.LocalHub
	IAMMode             iamservice.IAMMode
	IAMModeSource       string
	AuthProxy           DelegatedAuthProxy
	IAMDirectory        iamservice.IAMDirectory
	IAMRegistry         *fwiamadapters.Registry
	IAMDirectoryService fwiamcontracts.DirectoryService
	IAMAuthzService     fwiamcontracts.AuthzService
	IAMContextService   fwiamcontracts.IdentityContextService
}

type gatewayClient interface {
	Enabled() bool
	Invoke(ctx context.Context, params gateway.InvokeParams) (*gateway.InvokeResult, error)
	ListPlatformCapabilities(ctx context.Context, opts gateway.ListPlatformCapabilitiesOptions) ([]gateway.PlatformCapabilityRecord, error)
	ResolveGatewayTenantUUID(ctx context.Context) (string, error)
	ListAgents(ctx context.Context, env string) ([]gateway.AgentRecord, error)
	GetAgent(ctx context.Context, agentUUID string) (*gateway.AgentRecord, error)
	SyncPluginSkill(ctx context.Context, params gateway.PluginSkillSyncParams) (*gateway.PluginSkillSyncResult, error)
	SyncPluginAgent(ctx context.Context, params gateway.PluginAgentSyncParams) (*gateway.PluginAgentSyncResult, error)
	RegisterCatalog(ctx context.Context, catalog *capabilities.CatalogSnapshot, assets []capabilities.ProtocolAsset) error
	CreateAgentSession(ctx context.Context, params gateway.AgentSessionParams) (*gateway.AgentSessionRecord, error)
	ListAgentSessions(ctx context.Context, opts gateway.AgentSessionListOptions) ([]gateway.AgentSessionRecord, error)
	ListAgentSessionMessages(ctx context.Context, opts gateway.AgentSessionMessageListOptions) ([]gateway.AgentSessionMessageRecord, error)
	DeleteAgentSession(ctx context.Context, opts gateway.AgentSessionMutationOptions) error
	ArchiveAgentSession(ctx context.Context, opts gateway.AgentSessionMutationOptions) error
	StreamAgentSSE(ctx context.Context, params gateway.AgentStreamParams) (*gateway.AgentStream, error)
	Close() error
}

// RuntimeDefaults returns the configured runtime ops defaults (if any).
func (d *Deps) RuntimeDefaults() *config.RuntimeOpsDefaults {
	if d == nil || d.Config == nil {
		return nil
	}
	return d.Config.RuntimeOps
}

// RuntimeLogger provides a structured logger enriched with runtime metadata.
func (d *Deps) RuntimeLogger(ctx context.Context, component string, extra logger.Fields) *logger.Entry {
	if extra == nil {
		extra = logger.Fields{}
	}
	if _, ok := extra[runtimelogging.FieldSubscriber]; !ok {
		extra[runtimelogging.FieldSubscriber] = component
	}
	if ctx == nil && d != nil {
		ctx = d.Ctx
	}

	var tenantID string
	if tid, ok := authx.TenantUUIDFromContext(ctx); ok && tid != "" {
		tenantID = tid
	}

	traceID := ""
	if ctx != nil {
		if v := ctx.Value("request_id"); v != nil {
			if s, ok := v.(string); ok {
				traceID = s
			}
		}
	}
	baseFields := runtimelogging.Fields{
		runtimelogging.FieldPluginID:   PluginID,
		runtimelogging.FieldTenantUUID: tenantID,
		runtimelogging.FieldTraceID:    traceID,
		runtimelogging.FieldComponent:  component,
		"request_id":                   traceID,
	}
	normalized := runtimelogging.NormalizeContextFields(baseFields, runtimelogging.Fields(extra))
	if trimAny(normalized["biz_scene"]) == "" && strings.TrimSpace(component) != "" {
		normalized["biz_scene"] = strings.TrimSpace(component)
	}
	if trimAny(normalized["biz_domain"]) == "" {
		normalized["biz_domain"] = inferBizDomain(component)
	}
	labels := buildRuntimeLabels(component, normalized)
	normalized["labels"] = labels
	for k, v := range labels {
		if _, exists := normalized[k]; !exists {
			normalized[k] = v
		}
	}
	extra = logger.Fields(normalized)

	return logger.WithRuntimeFields(PluginID, tenantID, traceID, component, extra)
}

func buildRuntimeLabels(component string, fields runtimelogging.Fields) map[string]string {
	labels := map[string]string{
		"system":   firstNonEmpty(strings.TrimSpace(os.Getenv("POWERX_LOG_SYSTEM")), "powerx"),
		"service":  firstNonEmpty(strings.TrimSpace(os.Getenv("POWERX_LOG_SERVICE")), "backend"),
		"env":      firstNonEmpty(strings.TrimSpace(os.Getenv("POWERX_ENV")), strings.TrimSpace(os.Getenv("POWERX_SERVER_MODE")), "dev"),
		"instance": firstNonEmpty(strings.TrimSpace(os.Getenv("HOSTNAME")), "local"),
	}
	if module := firstNonEmpty(trimAny(fields["module"]), strings.TrimSpace(component)); module != "" {
		labels["module"] = module
	}
	return labels
}

func trimAny(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func inferBizDomain(component string) string {
	c := strings.ToLower(strings.TrimSpace(component))
	switch {
	case c == "":
		return "backend"
	case strings.Contains(c, "runtime") || strings.Contains(c, "mcp") || strings.Contains(c, "wsbus"):
		return "runtime_ops"
	case strings.Contains(c, "capability"):
		return "capability"
	case strings.Contains(c, "integration"):
		return "integration"
	case strings.Contains(c, "marketplace"):
		return "marketplace"
	case strings.Contains(c, "iam") || strings.Contains(c, "auth"):
		return "iam"
	case strings.Contains(c, "security") || strings.Contains(c, "consent") || strings.Contains(c, "toolgrant"):
		return "security"
	case strings.Contains(c, "operations") || strings.Contains(c, "incident") || strings.Contains(c, "sla"):
		return "operations"
	case strings.Contains(c, "agent"):
		return "agent"
	default:
		return "backend"
	}
}

func (d *Deps) LocalIAMEnabled() bool {
	return d != nil && d.IAMMode == iamservice.IAMModeLocal && d.IAMDirectory != nil
}

func (d *Deps) DelegatedIAMEnabled() bool {
	return d != nil && d.IAMMode == iamservice.IAMModeDelegated && d.AuthProxy != nil
}

func (d *Deps) LocalDirectory() iamservice.IAMDirectory {
	if d.LocalIAMEnabled() {
		return d.IAMDirectory
	}
	return nil
}

func (d *Deps) DelegatedProxy() DelegatedAuthProxy {
	if d.DelegatedIAMEnabled() {
		return d.AuthProxy
	}
	return nil
}
