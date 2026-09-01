package runtime_ops

import (
	"context"
	"net/http"
	"strings"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusPublishRequest struct {
	Topic      string `json:"topic"`
	Payload    any    `json:"payload"`
	TenantUUID string `json:"tenant_uuid"`
	MemberUUID string `json:"member_uuid"`
	TraceID    string `json:"trace_id"`
}

// WSBusPublishHandler provides a debug publish endpoint for standalone mode.
func WSBusPublishHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}
		var req wsBusPublishRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
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
		if !allowWSBusPublish(c, deps, req.Topic, tenantUUID, req.MemberUUID, traceID) {
			return
		}

		publisher := fwwsbus.Publisher(fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		))
		outboundBearer := ""
		hostCfg, useHost := resolveWSBusHostClientConfig(deps)
		if useHost {
			outboundBearer = resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)

			hostClient, err := fwwsbus.NewHostClient(hostCfg)
			if err != nil {
				contracts.ResponseError(c, http.StatusBadGateway, contracts.ErrCodeInternalError, "host ws bus client init failed")
				return
			}
			publisher = hostClient
		}
		result := publisher.Publish(context.Background(), req.Topic, req.Payload, fwwsbus.PublishOptions{
			TenantUUID:  tenantUUID,
			MemberUUID:  strings.TrimSpace(req.MemberUUID),
			TraceID:     traceID,
			BearerToken: outboundBearer,
		})

		if !result.OK {
			contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
			return
		}
		contracts.ResponseSuccess(c, gin.H{"ok": true})
	}
}

// allowWSBusPublish makes events.yaml the sole allowlist for management-plane
// WebSocket publishes. A missing descriptor is deliberately a hard failure: an
// operator cannot turn an arbitrary topic into a public realtime surface.
func allowWSBusPublish(c *gin.Context, deps *app.Deps, topic, tenantUUID, memberUUID, traceID string) bool {
	var descriptors []frameworkrealtime.Descriptor
	if deps != nil {
		descriptors = deps.RealtimeDescriptors
	}
	decision := frameworkrealtime.Decide(
		descriptors,
		frameworkrealtime.ActionPublish,
		strings.TrimSpace(topic),
		frameworkrealtime.ProtocolWS,
		"message",
		frameworkrealtime.Scope{
			TenantUUID: strings.TrimSpace(tenantUUID),
			MemberUUID: strings.TrimSpace(memberUUID),
			TraceID:    strings.TrimSpace(traceID),
		},
	)
	if decision.Allowed {
		return true
	}
	contracts.ResponseError(c, http.StatusForbidden, decision.Reason, decision.Reason)
	return false
}
