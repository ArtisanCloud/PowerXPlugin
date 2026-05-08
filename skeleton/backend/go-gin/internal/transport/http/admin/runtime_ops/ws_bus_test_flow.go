package runtime_ops

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusTestFlowRequest struct {
	Topic      string `json:"topic"`
	TenantUUID string `json:"tenant_uuid"`
	TraceID    string `json:"trace_id"`
	Payload    any    `json:"payload"`
}

// WSBusTestFlowHandler runs grant -> publish in one unified backend entry.
func WSBusTestFlowHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}

		var req wsBusTestFlowRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}

		tenantUUID, tenantMismatch := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}
		if strings.TrimSpace(tenantUUID) == "" && os.Getenv("POWERX_PROXY") != "1" {
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
			return
		}

		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			contracts.ResponseBadRequest(c, "topic is required")
			return
		}
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}
		if traceID == "" {
			traceID = "ws-bus-flow-" + time.Now().UTC().Format("20060102T150405.000Z")
		}

		outboundBearer := ""
		if os.Getenv("POWERX_PROXY") == "1" && deps.Config != nil && deps.Config.Gateway != nil {
			outboundBearer = resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)
		}

		grantResult := fwwsbus.PublishResult{OK: true}
		topics, expand := fwwsbus.ExpandTopicsForRegister([]string{topic})
		if !expand.OK {
			contracts.ResponseError(c, http.StatusBadRequest, expand.ErrorCode, expand.ErrorMessage)
			return
		}

		publisher := fwwsbus.Publisher(fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		))

		if os.Getenv("POWERX_PROXY") == "1" && deps.Config != nil && deps.Config.Gateway != nil {
			hostClient, err := fwwsbus.NewHostClient(fwwsbus.HostClientConfig{
				BaseURL:    strings.TrimSpace(deps.Config.Gateway.BaseURL),
				APIPrefix:  strings.TrimSpace(deps.Config.Gateway.APIPrefix),
				Token:      strings.TrimSpace(deps.Config.Gateway.ToolToken),
				TenantUUID: "",
				UserAgent:  strings.TrimSpace(deps.Config.Gateway.UserAgent),
				Timeout:    deps.Config.Gateway.Timeout,
			})
			if err == nil {
				grantResult = hostClient.RegisterTopics(context.Background(), topics, fwwsbus.PublishOptions{
					TenantUUID:  tenantUUID,
					TraceID:     traceID,
					BearerToken: outboundBearer,
				})
				if !grantResult.OK {
					contracts.ResponseError(c, http.StatusBadRequest, grantResult.ErrorCode, grantResult.ErrorMessage)
					return
				}
				publisher = hostClient
			}
		}

		payload := req.Payload
		if payload == nil {
			payload = gin.H{
				"type":        "framework.wsbus.test",
				"title":       "WS Bus Flow Test",
				"message":     "grant->publish flow from framework lab",
				"tenant_uuid": tenantUUID,
				"trace_id":    traceID,
				"created_at":  time.Now().UTC().Format(time.RFC3339Nano),
			}
		}

		publishResult := publisher.Publish(context.Background(), topic, payload, fwwsbus.PublishOptions{
			TenantUUID:  tenantUUID,
			TraceID:     traceID,
			BearerToken: outboundBearer,
		})
		if !publishResult.OK {
			contracts.ResponseError(c, http.StatusBadRequest, publishResult.ErrorCode, publishResult.ErrorMessage)
			return
		}

		// Ensure UI probe can always observe an event frame on current plugin WS session.
		// In proxy mode this acts as local echo while host publish still executes above.
		echoResult := fwwsbus.NewLocalPublisher(deps.WSBusHub, nil).Publish(
			context.Background(),
			topic,
			payload,
			fwwsbus.PublishOptions{
				TenantUUID: tenantUUID,
				TraceID:    traceID,
			},
		)
		if !echoResult.OK {
			contracts.ResponseError(c, http.StatusBadRequest, echoResult.ErrorCode, echoResult.ErrorMessage)
			return
		}

		flowMode := "local_only"
		if os.Getenv("POWERX_PROXY") == "1" {
			flowMode = "host_plus_local_echo"
		}

		contracts.ResponseSuccess(c, gin.H{
			"ok":          true,
			"topic":       topic,
			"tenant_uuid": tenantUUID,
			"trace_id":    traceID,
			"grant_ok":    grantResult.OK,
			"publish_ok":  publishResult.OK,
			"echo_ok":     echoResult.OK,
			"flow_mode":   flowMode,
		})
	}
}
