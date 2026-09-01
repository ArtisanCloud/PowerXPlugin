package realtime

import (
	"context"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
)

// NewAuthorizedWSPublisher wraps a WebSocket publisher with the runtime
// events.yaml allowlist. Callers must explicitly choose the event type; there
// is no permissive default for undeclared topics.
func NewAuthorizedWSPublisher(inner fwwsbus.Publisher, descriptors []Descriptor, eventType string) fwwsbus.Publisher {
	return &authorizedWSPublisher{
		inner:       inner,
		descriptors: descriptors,
		eventType:   strings.TrimSpace(eventType),
	}
}

type authorizedWSPublisher struct {
	inner       fwwsbus.Publisher
	descriptors []Descriptor
	eventType   string
}

func (p *authorizedWSPublisher) Publish(ctx context.Context, topic string, payload any, opts fwwsbus.PublishOptions) fwwsbus.PublishResult {
	if p == nil || p.inner == nil {
		return fwwsbus.FailureResult(fwwsbus.ErrorCodePublisherNotConfigured, "publisher is not configured")
	}
	eventType := p.eventType
	if eventType == "" {
		return fwwsbus.FailureResult("REALTIME_EVENT_TYPE_REQUIRED", "realtime event type is required")
	}
	decision := Decide(p.descriptors, ActionPublish, topic, ProtocolWS, eventType, Scope{
		TenantUUID: strings.TrimSpace(opts.TenantUUID),
		MemberUUID: strings.TrimSpace(opts.MemberUUID),
		TraceID:    strings.TrimSpace(opts.TraceID),
	})
	if !decision.Allowed {
		return fwwsbus.FailureResult(decision.Reason, decision.Reason)
	}
	return p.inner.Publish(ctx, topic, payload, opts)
}
