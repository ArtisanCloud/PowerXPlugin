package runtime_ops

import (
	"context"
	"net/http"
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusGrantRequest struct {
	Topics     []string `json:"topics"`
	TenantUUID string   `json:"tenant_uuid"`
	TraceID    string   `json:"trace_id"`
}

// WSBusGrantHandler provides a debug grant endpoint for standalone/host.
func WSBusGrantHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}
		var req wsBusGrantRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}
		topics, result := fwwsbus.ExpandTopicsForRegister(req.Topics)
		if !result.OK {
			contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
			return
		}

		tenantUUID, tenantMismatch := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}

		if os.Getenv("POWERX_PROXY") == "1" && deps.Config != nil && deps.Config.Gateway != nil {
			outboundBearer := resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)

				hostClient, err := fwwsbus.NewHostClient(fwwsbus.HostClientConfig{
					BaseURL:    strings.TrimSpace(deps.Config.Gateway.BaseURL),
					APIPrefix:  strings.TrimSpace(deps.Config.Gateway.APIPrefix),
					Token:      strings.TrimSpace(deps.Config.Gateway.ToolToken),
					TenantUUID: "",
					UserAgent:  strings.TrimSpace(deps.Config.Gateway.UserAgent),
					Timeout:    deps.Config.Gateway.Timeout,
				})
			if err == nil {
				result = hostClient.RegisterTopics(context.Background(), topics, fwwsbus.PublishOptions{
					TenantUUID:  tenantUUID,
					TraceID:     traceID,
					BearerToken: outboundBearer,
				})
				if !result.OK {
					contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
					return
				}
				contracts.ResponseSuccess(c, gin.H{"ok": true, "topics": topics})
				return
			}
		}

		// standalone: no-op grant, just return expanded topics
		contracts.ResponseSuccess(c, gin.H{"ok": true, "topics": topics})
	}
}
