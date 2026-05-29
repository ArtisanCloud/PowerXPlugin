package runtime_ops

import (
	"context"
	"errors"
	"os"
	"strings"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

// resolveGatewayBearerToken 统一 ws-bus 出站 token 选择规则：
// - proxy 模式：不透传入站 Bearer，host/delegated 由 HostClient TokenProvider 交换 STS；
// - local/standalone 模式：默认不透传；仅在 delegated 且非 proxy 时透传入站 Bearer。
func resolveGatewayBearerToken(c *gin.Context, deps *app.Deps) string {
	if c == nil || deps == nil {
		return ""
	}
	if os.Getenv("POWERX_PROXY") == "1" {
		return ""
	}
	if deps.Config != nil && deps.Config.Gateway != nil {
		authScheme := strings.ToLower(strings.TrimSpace(deps.Config.Gateway.AuthScheme))
		apiKey := strings.TrimSpace(deps.Config.Gateway.APIKey)
		if authScheme == "apikey" || authScheme == "api_key" || authScheme == "api-key" {
			return ""
		}
		if authScheme == "" && apiKey != "" {
			return ""
		}
	}
	if deps.IAMMode != iamservice.IAMModeDelegated {
		return ""
	}
	if raw, ok := middleware.GetRawBearerToken(c); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

// resolveGatewayTenantUUID 统一 ws-bus 出站 tenant 选择规则：
// 1) proxy 模式：tenant 由宿主按凭证解析，插件侧不透传 tenant_uuid；
// 2) 非 proxy 模式：入站请求 token/上下文租户优先；
// 3) 若请求体 tenant_uuid 与入站租户不一致，返回 mismatch=true；
// 4) 不再使用 gateway 配置或工具 token 推导 tenant，避免插件侧越权注入。
func resolveGatewayTenantUUID(c *gin.Context, deps *app.Deps, requested string) (tenantUUID string, mismatch bool) {
	resolvedTenantUUID, tenantMismatch := admincommon.ResolveTenantUUIDStrict(c, requested)
	if tenantMismatch {
		return "", true
	}
	tenantUUID = strings.TrimSpace(resolvedTenantUUID)
	if tenantUUID != "" {
		return tenantUUID, false
	}
	return "", false
}

// logGatewayAuthSelection 输出 ws-bus 出站鉴权选择，便于联调观察 token 来源。
func logGatewayAuthSelection(c *gin.Context, deps *app.Deps, outboundBearer string, tenantUUID string) {
	if deps == nil || deps.Config == nil || deps.Config.Logging == nil || !deps.Config.Logging.DebugMode {
		return
	}

	inboundBearerPresent := false
	inboundBearerPrefix := ""
	if c != nil {
		if raw, ok := middleware.GetRawBearerToken(c); ok {
			raw = strings.TrimSpace(raw)
			if raw != "" {
				inboundBearerPresent = true
				inboundBearerPrefix = tokenPrefix(raw)
			}
		}
	}

	apiKey := ""
	authScheme := ""
	if deps.Config.Gateway != nil {
		apiKey = strings.TrimSpace(deps.Config.Gateway.APIKey)
		authScheme = strings.TrimSpace(deps.Config.Gateway.AuthScheme)
	}

	outboundSource := "sts_token_provider"
	if strings.EqualFold(authScheme, "apikey") || strings.EqualFold(authScheme, "api_key") || strings.EqualFold(authScheme, "api-key") {
		outboundSource = "PX_GATEWAY_API_KEY"
	}
	if strings.TrimSpace(outboundBearer) != "" {
		outboundSource = "request_bearer_passthrough"
	}

	taskID := strings.TrimSpace(inboundBearerPrefix)
	if taskID == "" {
		taskID = runtimelogging.FallbackUnknown
	}

	extraFields := logger.Fields{
		runtimelogging.FieldTaskID:      taskID,
		runtimelogging.FieldTopic:       "runtime_ops.ws_bus.gateway_auth",
		runtimelogging.FieldStatus:      runtimelogging.StatusProcessing,
		runtimelogging.FieldTenantUUID:  strings.TrimSpace(tenantUUID),
		runtimelogging.FieldGatewayAuth: authScheme,
		runtimelogging.FieldTokenSource: outboundSource,
		"biz_scene":                     "wsbus_gateway_auth",
		"biz_domain":                    "runtime_ops",
		"iam_mode":                      deps.IAMMode,
		"inbound_bearer_present":        inboundBearerPresent,
		"inbound_bearer_prefix":         inboundBearerPrefix,
		"outbound_bearer_prefix":        tokenPrefix(outboundBearer),
		"px_gateway_api_key_set":        apiKey != "",
		"resolved_gateway_tenant":       strings.TrimSpace(tenantUUID),
	}
	deps.RuntimeLogger(c.Request.Context(), "ws_bus_gateway_auth", extraFields).Info("WS bus gateway auth resolved")
}

func tokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 16 {
		return token
	}
	return token[:16] + "..."
}

// resolveWSBusHostClientConfig 统一 WSBus HostClient 模式与凭证决策：
// 1) host(POWERX_PROXY=1 + IAM delegated): bearer + STS TokenProvider；
// 2) local+proxy(POWERX_PROXY=1 + IAM local): 按 PX_GATEWAY_AUTH_SCHEME 选择 bearer/apikey；
// 3) local(POWERX_PROXY!=1): 不走 host client。
func resolveWSBusHostClientConfig(deps *app.Deps) (cfg fwwsbus.HostClientConfig, useHost bool) {
	if deps == nil || deps.Config == nil || deps.Config.Gateway == nil || strings.TrimSpace(os.Getenv("POWERX_PROXY")) != "1" {
		return fwwsbus.HostClientConfig{}, false
	}
	gw := deps.Config.Gateway
	cfg = fwwsbus.HostClientConfig{
		BaseURL:    strings.TrimSpace(gw.BaseURL),
		APIPrefix:  strings.TrimSpace(gw.APIPrefix),
		TenantUUID: "",
		UserAgent:  strings.TrimSpace(gw.UserAgent),
		Timeout:    gw.Timeout,
	}

	if deps.IAMMode == iamservice.IAMModeDelegated {
		cfg.AuthScheme = "bearer"
		cfg.TokenProvider = newPowerXSTSTokenProvider(deps)
		return cfg, true
	}

	// local+proxy 模式：按网关配置分流
	cfg.AuthScheme = strings.ToLower(strings.TrimSpace(gw.AuthScheme))
	cfg.APIKey = strings.TrimSpace(gw.APIKey)
	return cfg, true
}

func newPowerXSTSTokenProvider(deps *app.Deps) fwwsbus.TokenProvider {
	return func(ctx context.Context) (string, error) {
		if deps == nil || deps.PowerXClient == nil {
			return "", errors.New("powerx STS client is not configured")
		}
		token := strings.TrimSpace(deps.PowerXClient.GetToken())
		if token != "" && token != "sts" {
			return token, nil
		}
		token, _, err := deps.PowerXClient.ExchangeSTS(ctx)
		return strings.TrimSpace(token), err
	}
}
