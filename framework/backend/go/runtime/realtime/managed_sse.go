package realtime

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ssebus"
)

// PublishManagedSSE applies descriptor authorization before publishing a
// framework-managed SSE envelope. Raw upstream SSE must use ProxySSEStream
// instead, because its event/data semantics are deliberately preserved.
func PublishManagedSSE(ctx context.Context, broker *ssebus.Broker, descriptors []Descriptor, channel string, eventType string, payload any, scope Scope) (PermissionDecision, error) {
	decision := Decide(descriptors, ActionPublish, channel, ProtocolSSE, eventType, scope)
	if !decision.Allowed {
		return decision, fmt.Errorf("%s", decision.Reason)
	}
	if err := broker.Publish(ctx, channel, eventType, NewSSEEnvelope(channel, eventType, payload, scope), ssebus.PublishOptions{
		TenantUUID: scope.TenantUUID,
		MemberUUID: scope.MemberUUID,
		TraceID:    scope.TraceID,
	}); err != nil {
		return decision, err
	}
	return decision, nil
}

// SubscribeManagedSSE applies descriptor authorization before allocating a
// subscriber. A denied request gets a closed channel and a reason-bearing
// decision, so callers never accidentally stream an unauthorized channel.
func SubscribeManagedSSE(broker *ssebus.Broker, descriptors []Descriptor, channel string, eventType string, scope Scope) (<-chan ssebus.Event, func(), PermissionDecision) {
	decision := Decide(descriptors, ActionSubscribe, channel, ProtocolSSE, eventType, scope)
	if !decision.Allowed || broker == nil {
		closed := make(chan ssebus.Event)
		close(closed)
		return closed, func() {}, decision
	}
	events, cancel := broker.Subscribe(channel)
	return events, cancel, decision
}
