package runtime_ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

type notificationTestRequest struct {
	TenantUUID string `json:"tenant_uuid"`
	MemberUUID string `json:"member_uuid"`
	Topic      string `json:"topic"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	TraceID    string `json:"trace_id"`
}

func NotificationTestHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			contracts.ResponseServiceUnavailable(c, "ws bus is not configured", nil)
			return
		}

		var req notificationTestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}

		resolvedTenantUUID, tenantMismatch := admincommon.ResolveTenantUUIDStrict(c, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}
		tenantUUID := strings.TrimSpace(resolvedTenantUUID)
		if tenantUUID == "" {
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
			return
		}

		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			topic = "plugin.notify.tenant." + tenantUUID
		}

		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "WS Test Notification"
		}
		message := strings.TrimSpace(req.Message)
		if message == "" {
			message = "websocket probe event"
		}
		traceID := strings.TrimSpace(req.TraceID)
		if traceID == "" {
			traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
		}
		if traceID == "" {
			traceID = "ws-notify-" + time.Now().UTC().Format("20060102T150405.000Z")
		}

		payload := gin.H{
			"type":        "notification.test",
			"title":       title,
			"message":     message,
			"tenant_uuid": tenantUUID,
			"member_uuid": strings.TrimSpace(req.MemberUUID),
			"trace_id":    traceID,
			"created_at":  time.Now().UTC().Format(time.RFC3339Nano),
		}

		hostCfg, useHost := resolveWSBusHostClientConfig(deps)
		hostPublishOK := false
		hostReachable := false
		powerxProxy := strings.TrimSpace(os.Getenv("POWERX_PROXY"))
		if useHost {
			hostReachable = true
			logGatewayAuthSelection(c, deps, "", tenantUUID)
			if err := sendHostNotification(c.Request.Context(), hostCfg, req, title, message); err != nil {
				contracts.ResponseError(c, http.StatusBadGateway, contracts.ErrCodeInternalError, err.Error())
				return
			}
			hostPublishOK = true
		} else {
			publisher := fwwsbus.Publisher(fwwsbus.NewAdapter(
				fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
				"",
				nil,
			))
			result := publisher.Publish(context.Background(), topic, payload, fwwsbus.PublishOptions{
				TenantUUID:  tenantUUID,
				MemberUUID:  strings.TrimSpace(req.MemberUUID),
				TraceID:     traceID,
				BearerToken: "",
			})
			if !result.OK {
				contracts.ResponseError(c, http.StatusBadRequest, result.ErrorCode, result.ErrorMessage)
				return
			}
		}

		flowMode := "local_only"
		effectiveTarget := "local"
		if useHost {
			flowMode = "host_strict_ok"
			effectiveTarget = "host"
			if hostReachable && hostPublishOK {
				flowMode = "host_strict_ok"
			}
		}

		contracts.ResponseSuccess(c, gin.H{
			"ok":               true,
			"topic":            topic,
			"tenant_uuid":      tenantUUID,
			"member_uuid":      strings.TrimSpace(req.MemberUUID),
			"trace_id":         traceID,
			"flow_mode":        flowMode,
			"effective_target": effectiveTarget,
			"powerx_proxy":     powerxProxy,
			"provider_mode":    strings.TrimSpace(deps.ProviderMode.String()),
			"host_reachable":   hostReachable,
			"host_publish_ok":  hostPublishOK,
		})
	}
}

func sendHostNotification(ctx context.Context, cfg fwwsbus.HostClientConfig, req notificationTestRequest, title, message string) error {
	endpoint, err := url.JoinPath(strings.TrimRight(cfg.BaseURL, "/"), strings.Trim(strings.TrimSpace(cfg.APIPrefix), "/"), "notifications/test")
	if err != nil {
		return fmt.Errorf("build host notification endpoint failed: %w", err)
	}
	body := map[string]any{
		"member_uuid":  strings.TrimSpace(req.MemberUUID),
		"title":        title,
		"content":      message,
		"type":         "info",
		"category":     "system",
		"is_important": false,
		"metadata": map[string]any{
			"source":   "plugin-framework-runtime",
			"trace_id": strings.TrimSpace(req.TraceID),
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal host notification request failed: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create host notification request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if ua := strings.TrimSpace(cfg.UserAgent); ua != "" {
		httpReq.Header.Set("User-Agent", ua)
	}
	switch strings.ToLower(strings.TrimSpace(cfg.AuthScheme)) {
	case "apikey", "api_key", "api-key":
		if apiKey := strings.TrimSpace(cfg.APIKey); apiKey != "" {
			httpReq.Header.Set("Authorization", "ApiKey "+apiKey)
		}
	default:
		token := strings.TrimSpace(cfg.Token)
		if token == "" && cfg.TokenProvider != nil {
			issued, err := cfg.TokenProvider(ctx)
			if err != nil {
				return fmt.Errorf("host notification STS token exchange failed: %w", err)
			}
			token = strings.TrimSpace(issued)
		}
		if token == "" {
			return fmt.Errorf("host notification STS token is empty")
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: cfg.Timeout}
	if client.Timeout <= 0 {
		client.Timeout = 10 * time.Second
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("host notification request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("host notification request failed: status=%d endpoint=%s body=%s", resp.StatusCode, endpoint, strings.TrimSpace(string(respBody)))
	}
	return nil
}
