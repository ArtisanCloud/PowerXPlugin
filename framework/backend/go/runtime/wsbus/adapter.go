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
}

func NewAdapter(inner Publisher, defaultTenant string, logger *slog.Logger) Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		inner:         inner,
		defaultTenant: strings.TrimSpace(defaultTenant),
		logger:        runtimelogging.NewSlogAdapter(logger),
	}
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
		a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusFailed, ErrorCodeTenantRequired)
		return FailureResult(ErrorCodeTenantRequired, "tenant_uuid is required")
	}

	bearer := strings.TrimSpace(opts.BearerToken)
	if bearer == "" {
		bearer = strings.TrimSpace(middleware.BearerTokenFromContext(ctx))
	}

	result = a.inner.Publish(ctx, normalized, payload, PublishOptions{
		TenantUUID:  tenantUUID,
		TraceID:     traceID,
		BearerToken: bearer,
	})
	if !result.OK {
		reason := result.ErrorCode
		if missingContext {
			reason = runtimelogging.ReasonMissingContext
		}
		a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusFailed, reason)
		return result
	}
	reason := ""
	if missingContext {
		reason = runtimelogging.ReasonMissingContext
	}
	a.logPublish(normalized, tenantUUID, traceID, runtimelogging.StatusSucceeded, reason)
	return result
}

func (a *Adapter) logPublish(topic, tenantUUID, traceID, status, reason string) {
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
