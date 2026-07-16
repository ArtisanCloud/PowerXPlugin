package middleware

import (
	"strings"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	obsintegration "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/integration"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

var delegatedGatewayRequired = []string{
	"PX_GATEWAY_BASE_URL",
	"POWERX_STS_CLIENT_ID",
	"POWERX_STS_CLIENT_SECRET",
	"POWERX_GRPC_UPSTREAM_ADDRESS",
	"POWERX_GRPC_UPSTREAM_TENANT_UUID",
	"PX_GATEWAY_AUTH_SCHEME=bearer",
}

// RequireCapabilityGateway ensures integration handlers expose consistent gateway error semantics.
func RequireCapabilityGateway(c *gin.Context, deps *app.Deps) bool {
	if c == nil {
		return false
	}
	if deps == nil {
		obsintegration.RecordPluginGatewayInvokeFailure("SERVICE_UNAVAILABLE")
		contracts.ResponseErrorWithDetails(c, 503, "SERVICE_UNAVAILABLE", "capability gateway unavailable", gin.H{
			"required":      delegatedGatewayRequired,
			"present":       []string{},
			"provider_mode": "unknown",
		})
		return false
	}

	if shouldEnforceDelegatedContract(deps) {
		if cfgErr := capgateway.ValidateDelegatedConfig(deps.Config); cfgErr != nil {
			obsintegration.RecordPluginGatewayInvokeFailure(cfgErr.Code)
			contracts.ResponseErrorWithDetails(c, 503, cfgErr.Code, cfgErr.Message, gin.H{
				"required":      cfgErr.Required,
				"present":       cfgErr.Present,
				"provider_mode": cfgErr.ProviderMode,
			})
			return false
		}
	}

	if deps.CapabilityGateway == nil || !deps.CapabilityGateway.Enabled() {
		obsintegration.RecordPluginGatewayInvokeFailure("SERVICE_UNAVAILABLE")
		contracts.ResponseErrorWithDetails(c, 503, "SERVICE_UNAVAILABLE", "capability gateway unavailable", gin.H{
			"required":      delegatedGatewayRequired,
			"present":       detectGatewayPresent(deps),
			"provider_mode": detectGatewayProviderMode(deps),
		})
		return false
	}
	return true
}

func shouldEnforceDelegatedContract(deps *app.Deps) bool {
	if deps == nil {
		return false
	}
	if deps.ProviderMode == fwprovider.ModeDelegated || deps.IAMAdapterMode == iamservice.IAMAdapterModeDelegated {
		return true
	}
	return strings.EqualFold(detectGatewayProviderMode(deps), "delegated")
}

func detectGatewayProviderMode(deps *app.Deps) string {
	if deps == nil {
		return "unknown"
	}
	if deps.ProviderMode == fwprovider.ModeDelegated || deps.IAMAdapterMode == iamservice.IAMAdapterModeDelegated {
		return "delegated"
	}
	if deps.ProviderMode == fwprovider.ModeLocal || deps.IAMAdapterMode == iamservice.IAMAdapterModeLocal {
		return "local"
	}
	if deps.Config == nil || deps.Config.Context == nil {
		return "unknown"
	}
	mode := strings.ToLower(strings.TrimSpace(deps.Config.Context.ProviderMode))
	if mode == "" {
		return "unknown"
	}
	return mode
}

func detectGatewayPresent(deps *app.Deps) []string {
	present := make([]string, 0, 3)
	if deps == nil || deps.Config == nil || deps.Config.Gateway == nil {
		return present
	}
	gcfg := deps.Config.Gateway
	if strings.TrimSpace(gcfg.BaseURL) != "" {
		present = append(present, "PX_GATEWAY_BASE_URL")
	}
	if strings.EqualFold(strings.TrimSpace(gcfg.AuthScheme), "bearer") {
		present = append(present, "PX_GATEWAY_AUTH_SCHEME=bearer")
	}
	if deps.Config.GRPCUpstream != nil {
		if strings.TrimSpace(deps.Config.GRPCUpstream.STSClientID) != "" {
			present = append(present, "POWERX_STS_CLIENT_ID")
		}
		if strings.TrimSpace(deps.Config.GRPCUpstream.STSClientSecret) != "" {
			present = append(present, "POWERX_STS_CLIENT_SECRET")
		}
		if strings.TrimSpace(deps.Config.GRPCUpstream.Address) != "" {
			present = append(present, "POWERX_GRPC_UPSTREAM_ADDRESS")
		}
		if strings.TrimSpace(deps.Config.GRPCUpstream.TenantUUID) != "" {
			present = append(present, "POWERX_GRPC_UPSTREAM_TENANT_UUID")
		}
	}
	return present
}
