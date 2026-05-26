package wsbus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
)

type Adapter struct {
	inner         Publisher
	defaultTenant string
	logger        runtimelogging.Logger
	hubBridge     LocalHub
	bridgeTopics  map[string]struct{}
}

func NewAdapter(inner Publisher, defaultTenant string, logger *slog.Logger, bridgeTopics ...string) Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	normalizedBridgeTopics := map[string]struct{}{}
	for _, topic := range bridgeTopics {
		normalized, result := NormalizeAndValidateTopic(topic)
		if !result.OK {
			continue
		}
		normalizedBridgeTopics[normalized] = struct{}{}
	}
	return &Adapter{
		inner:         inner,
		defaultTenant: strings.TrimSpace(defaultTenant),
		logger:        runtimelogging.NewSlogAdapter(logger),
		bridgeTopics:  normalizedBridgeTopics,
	}
}

func (a *Adapter) EnableHubBridge(hub LocalHub) {
	if a == nil {
		return
	}
	a.hubBridge = hub
}

func (a *Adapter) Publish(ctx context.Context, topic string, payload any, opts PublishOptions) PublishResult {
	if a == nil || a.inner == nil {
		return FailureResult(ErrorCodePublisherNotConfigured, "publisher is not configured")
	}
	normalized, result := NormalizeAndValidateTopic(topic)
	if !result.OK {
		return result
	}
	if result = ValidatePayload(payload); !result.OK {
		return result
	}

	traceID := strings.TrimSpace(opts.TraceID)
	missingContext := false
	if traceID == "" {
		traceID = strings.TrimSpace(middleware.RequestIDFromContext(ctx))
	}
	if traceID == "" {
		traceID = generateTraceID()
		missingContext = true
	}
	tenantUUID := strings.TrimSpace(opts.TenantUUID)
	if tenantUUID == "" {
		if t, ok := middleware.TenantUUIDFromContext(ctx); ok {
			tenantUUID = strings.TrimSpace(t)
		}
	}
	if tenantUUID == "" {
		tenantUUID = a.defaultTenant
	}
	if tenantUUID == "" {
		a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusFailed, ErrorCodeTenantRequired, PublishResult{})
		return FailureResult(ErrorCodeTenantRequired, "tenant_uuid is required")
	}

	bearer := strings.TrimSpace(opts.BearerToken)
	if bearer == "" {
		bearer = strings.TrimSpace(middleware.BearerTokenFromContext(ctx))
	}

	result = a.inner.Publish(ctx, normalized, payload, PublishOptions{
		TenantUUID:  tenantUUID,
		MemberUUID:  strings.TrimSpace(opts.MemberUUID),
		TraceID:     traceID,
		BearerToken: bearer,
	})
	if !result.OK {
		reason := result.ErrorCode
		if missingContext {
			reason = runtimelogging.ReasonMissingContext
		}
		a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusFailed, reason, result)
		return result
	}
	if a.hubBridge != nil {
		if _, ok := a.bridgeTopics[normalized]; ok {
			if err := a.hubBridge.Publish(ctx, normalized, payload, PublishOptions{
				TenantUUID:  tenantUUID,
				MemberUUID:  strings.TrimSpace(opts.MemberUUID),
				TraceID:     traceID,
				BearerToken: bearer,
			}); err != nil {
				a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusFailed, ErrorCodeLocalPublishFailed, result)
				return FailureResult(ErrorCodeLocalPublishFailed, err.Error())
			}
		}
	}
	reason := ""
	if missingContext {
		reason = runtimelogging.ReasonMissingContext
	}
	a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusSucceeded, reason, result)
	return result
}

func (a *Adapter) logPublish(topic, tenantUUID, traceID, status, reason string, result PublishResult) {
	if a == nil || a.logger == nil {
		return
	}
	fields := runtimelogging.NormalizeRuntimeFields(runtimelogging.Fields{
		runtimelogging.FieldTraceID:    strings.TrimSpace(traceID),
		runtimelogging.FieldTaskID:     strings.TrimSpace(traceID),
		runtimelogging.FieldTenantUUID: strings.TrimSpace(tenantUUID),
		runtimelogging.FieldTenantKey:  runtimelogging.TenantKeyFromUUID(tenantUUID),
		runtimelogging.FieldSubscriber: "wsbus.adapter",
		runtimelogging.FieldTopic:      strings.TrimSpace(topic),
		runtimelogging.FieldStatus:     status,
		runtimelogging.FieldReason:     reason,
		"outbound_url":                 strings.TrimSpace(result.OutboundURL),
		"http_status":                  result.HTTPStatus,
		"response_body":                strings.TrimSpace(result.ResponseBody),
		"upstream_error_code":          strings.TrimSpace(result.ErrorCode),
		"upstream_error_message":       strings.TrimSpace(result.ErrorMessage),
	})
	facade := runtimelogging.NewFacade(nil, a.logger)
	if status == runtimelogging.StatusFailed {
		facade.Warn("wsbus publish failed", runtimelogging.Entry{
			Fields: fields,
			Context: runtimelogging.Fields{
				"module":     "wsbus.adapter",
				"biz_scene":  "ws_publish",
				"biz_domain": "runtime",
			},
		})
		return
	}
	facade.Info("wsbus publish succeeded", runtimelogging.Entry{
		Fields: fields,
		Context: runtimelogging.Fields{
			"module":     "wsbus.adapter",
			"biz_scene":  "ws_publish",
			"biz_domain": "runtime",
		},
	})
}

func generateTraceID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return runtimelogging.FallbackUnknown
}
