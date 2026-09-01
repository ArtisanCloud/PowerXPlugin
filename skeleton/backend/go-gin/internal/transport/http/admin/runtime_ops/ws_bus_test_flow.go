package runtime_ops

import (
	"context"
	"net/http"
	"strings"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type wsBusTestFlowRequest struct {
	Topic      string `json:"topic"`
	TraceID    string `json:"trace_id"`
	ForceLocal bool   `json:"force_local"`
	MemberUUID string `json:"member_uuid"`
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

		tenantUUID, _ := resolveGatewayTenantUUID(c, deps, "")
		if strings.TrimSpace(tenantUUID) == "" {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeTenantRequired, "tenant required")
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
		if !allowWSBusPublish(c, deps, topic, tenantUUID, req.MemberUUID, traceID) {
			return
		}

		outboundBearer := ""
		hostCfg, useHost := resolveWSBusHostClientConfig(deps)
		if req.ForceLocal {
			useHost = false
		}
		if useHost {
			outboundBearer = resolveGatewayBearerToken(c, deps)
			logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)
		}

		grantResult := fwwsbus.PublishResult{OK: true}
		hostGrantOK := false
		hostPublishOK := false
		hostReachable := false
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

		if useHost {
			hostClient, err := fwwsbus.NewHostClient(hostCfg)
			if err != nil {
				contracts.ResponseError(c, http.StatusBadGateway, contracts.ErrCodeInternalError, "host ws bus client init failed")
				return
			}
			hostReachable = true
			grantResult = hostClient.RegisterTopics(context.Background(), topics, fwwsbus.PublishOptions{
				TenantUUID:  tenantUUID,
				TraceID:     traceID,
				BearerToken: outboundBearer,
			})
			if !grantResult.OK {
				contracts.ResponseError(c, http.StatusBadRequest, grantResult.ErrorCode, grantResult.ErrorMessage)
				return
			}
			hostGrantOK = true
			publisher = hostClient
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
			MemberUUID:  strings.TrimSpace(req.MemberUUID),
			TraceID:     traceID,
			BearerToken: outboundBearer,
		})
		if !publishResult.OK {
			contracts.ResponseError(c, http.StatusBadRequest, publishResult.ErrorCode, publishResult.ErrorMessage)
			return
		}
		if hostReachable {
			hostPublishOK = true
		}

		echoOK := !useHost
		echoSkipped := useHost

		flowMode := "local_only"
		if req.ForceLocal {
			flowMode = "local_forced"
		}
		if useHost {
			flowMode = "host_strict_ok"
			if hostReachable && hostGrantOK && hostPublishOK {
				flowMode = "host_strict_ok"
			}
		}

		contracts.ResponseSuccess(c, gin.H{
			"ok":              true,
			"topic":           topic,
			"tenant_uuid":     tenantUUID,
			"trace_id":        traceID,
			"grant_ok":        grantResult.OK,
			"publish_ok":      publishResult.OK,
			"echo_ok":         echoOK,
			"echo_skipped":    echoSkipped,
			"flow_mode":       flowMode,
			"host_reachable":  hostReachable,
			"host_grant_ok":   hostGrantOK,
			"host_publish_ok": hostPublishOK,
		})
	}
}
