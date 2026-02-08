package wsbus

import (
	"context"
	"log/slog"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
)

type Adapter struct {
	inner         Publisher
	defaultTenant string
	logger        *slog.Logger
}

func NewAdapter(inner Publisher, defaultTenant string, logger *slog.Logger) Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		inner:         inner,
		defaultTenant: strings.TrimSpace(defaultTenant),
		logger:        logger,
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
		return FailureResult(ErrorCodeTenantRequired, "tenant_uuid is required")
	}

	traceID := strings.TrimSpace(opts.TraceID)
	if traceID == "" {
		traceID = strings.TrimSpace(middleware.RequestIDFromContext(ctx))
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
	if !result.OK && a.logger != nil {
		a.logger.Warn("wsbus publish failed",
			slog.String("topic", normalized),
			slog.String("tenant", tenantUUID),
			slog.String("error_code", result.ErrorCode),
			slog.String("error_message", result.ErrorMessage),
		)
	}
	return result
}
