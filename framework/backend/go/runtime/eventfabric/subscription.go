package eventfabric

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

var renew = errors.New("EVENT_FABRIC_RECONNECT")

type subscription struct {
	cancel context.CancelFunc
	done   chan struct{}
	ready  chan struct{}
	items  chan Delivery
	err    error
	once   sync.Once
}

func (s *subscription) Close() error { s.cancel(); <-s.done; return nil }
func (s *subscription) Recv(ctx context.Context) (Delivery, error) {
	if err := ctx.Err(); err != nil {
		return Delivery{}, err
	}
	select {
	case <-ctx.Done():
		return Delivery{}, ctx.Err()
	case <-s.done:
		return Delivery{}, s.err
	case d := <-s.items:
		select {
		case <-s.done:
			return Delivery{}, s.err
		default:
		}
		if !time.Now().Before(d.AckDeadline) {
			return Delivery{}, failure(409, "EVENT_FABRIC_RECEIPT_STALE")
		}
		return d, nil
	}
}
func validateSubscription(in SubscribeRequest) bool {
	if !identity(in.TenantUUID, in.PluginID) || in.SubscriberID != in.PluginID || len(in.Topics) == 0 || len(in.Topics) > 100 || in.BatchSize < 1 || in.BatchSize > 100 || len(in.SupportedVersions) == 0 || len(in.SupportedVersions) > 32 || in.ReconnectLimit < 0 || in.ReconnectLimit > 20 || in.ReconnectBackoff < 0 || in.ReconnectBackoff > 5*time.Second {
		return false
	}
	if in.CompatibilityMode != "strict" && in.CompatibilityMode != "backward" && in.CompatibilityMode != "any" {
		return false
	}
	seen := map[string]bool{}
	for _, topic := range in.Topics {
		if !physicalTopic(in.TenantUUID, topic) || seen[topic] || len(in.ExpectedPublishers[topic]) == 0 {
			return false
		}
		seen[topic] = true
		for _, p := range in.ExpectedPublishers[topic] {
			if !valid(p, 128) {
				return false
			}
		}
	}
	for _, v := range in.SupportedVersions {
		if !valid(v, 32) {
			return false
		}
	}
	return true
}
func (c *Client) Subscribe(ctx context.Context, in SubscribeRequest) (Subscription, error) {
	if c == nil || c.gateway == nil {
		return nil, failure(503, "EVENT_FABRIC_UNAVAILABLE")
	}
	if !validateSubscription(in) {
		return nil, failure(400, "EVENT_FABRIC_INVALID_ARGUMENT")
	}
	// Own all mutable policy so a caller cannot change identity under a live stream.
	in.Topics = append([]string(nil), in.Topics...)
	in.SupportedVersions = append([]string(nil), in.SupportedVersions...)
	publishers := map[string][]string{}
	for topic, p := range in.ExpectedPublishers {
		publishers[topic] = append([]string(nil), p...)
	}
	in.ExpectedPublishers = publishers
	run, cancel := context.WithCancel(ctx)
	s := &subscription{cancel: cancel, done: make(chan struct{}), ready: make(chan struct{}), items: make(chan Delivery)}
	go func() { defer close(s.done); defer cancel(); s.err = c.consume(run, in, s) }()
	select {
	case <-ctx.Done():
		_ = s.Close()
		return nil, ctx.Err()
	case <-s.done:
		select {
		case <-s.ready:
			return s, nil
		default:
			return nil, s.err
		}
	case <-s.ready:
		return s, nil
	}
}
func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
func validateDelivery(in SubscribeRequest, d Delivery) bool {
	if d.ContractVersion != Contract || d.TenantUUID != in.TenantUUID || d.SubscriberID != in.SubscriberID || !canonicalUUID(d.DeliveryID) || !valid(d.EventID, 128) || !valid(d.IdempotencyKey, 128) || !contains(in.Topics, d.Topic) || !contains(in.ExpectedPublishers[d.Topic], d.PublisherPluginID) || !valid(d.PublishedBy, 512) || !valid(d.TraceID, 256) || !valid(d.Version, 32) || !json.Valid(d.Payload) || len(d.Payload) > 1<<20 || d.Attempt < 1 || d.MaxAttempts < d.Attempt || d.AckDeadline.IsZero() || !time.Now().Before(d.AckDeadline) {
		return false
	}
	return in.CompatibilityMode != "strict" || contains(in.SupportedVersions, d.Version)
}
func streamError(code, trace, request string) *HTTPError {
	status := 502
	switch code {
	case "EVENT_FABRIC_CONTRACT_REQUIRED", "EVENT_FABRIC_INVALID_ARGUMENT", "EVENT_FABRIC_RESERVED_ATTRIBUTE", "EVENT_FABRIC_STREAM_REQUIRED":
		status = 400
	case "EVENT_FABRIC_SERVICE_REQUIRED":
		status = 401
	case "EVENT_FABRIC_TENANT_FORBIDDEN", "EVENT_FABRIC_OWNER_FORBIDDEN", "EVENT_FABRIC_SUBSCRIBER_FORBIDDEN", "EVENT_FABRIC_TOPIC_FORBIDDEN", "EVENT_FABRIC_DELIVERY_FORBIDDEN", "EVENT_FABRIC_OPERATION_FORBIDDEN", "EVENT_FABRIC_GRANT_REQUIRED":
		status = 403
	case "EVENT_FABRIC_AUTHORIZATION_UNAVAILABLE", "EVENT_FABRIC_UNAVAILABLE", "EVENT_FABRIC_SCHEMA_NOT_READY":
		status = 503
	case "EVENT_FABRIC_VERSION_INCOMPATIBLE", "EVENT_FABRIC_RECEIPT_STALE", "EVENT_FABRIC_IDEMPOTENCY_CONFLICT":
		status = 409
	}
	if !valid(code, 128) || !strings.HasPrefix(code, "EVENT_FABRIC_") {
		code = "EVENT_FABRIC_RESPONSE_INVALID"
		status = 502
	}
	err := failure(status, code)
	err.TraceID = trace
	err.RequestID = request
	return err
}
func recoverable(err error) bool {
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	var up *HTTPError
	if !errors.As(err, &up) {
		return false
	}
	switch up.ReasonCode {
	case "EVENT_FABRIC_SCHEMA_NOT_READY", "EVENT_FABRIC_RESPONSE_INVALID", "CAPABILITY_STREAM_RESPONSE_INVALID", "CAPABILITY_STREAM_EVENT_TOO_LARGE":
		return false
	}
	return up.StatusCode >= 500 && (up.ReasonCode == "EVENT_FABRIC_UNAVAILABLE" || up.ReasonCode == "EVENT_FABRIC_AUTHORIZATION_UNAVAILABLE" || strings.HasPrefix(up.ReasonCode, "CAPABILITY_"))
}
func (c *Client) consume(ctx context.Context, in SubscribeRequest, s *subscription) error {
	limit := in.ReconnectLimit
	if limit == 0 {
		limit = 6
	}
	backoff := in.ReconnectBackoff
	if backoff == 0 {
		backoff = 250 * time.Millisecond
	}
	body := map[string]any{"contract_version": Contract, "operation": "subscribe", "tenant_uuid": in.TenantUUID, "plugin_id": in.PluginID, "subscriber_id": in.SubscriberID, "topics": in.Topics, "batch_size": in.BatchSize, "compatibility_mode": in.CompatibilityMode, "supported_versions": in.SupportedVersions}
	failures := 0
	for {
		ready := false
		err := c.gateway.InvokeStream(ctx, capability.StreamInput{CapabilityID: ConsumeCapability, PreferredProtocol: Protocol, TenantUUID: in.TenantUUID, Body: body}, func(frame capability.StreamEvent) error {
			switch frame.Event {
			case "ready":
				var r struct {
					Contract   string `json:"contract_version"`
					Tenant     string `json:"tenant_uuid"`
					Subscriber string `json:"subscriber_id"`
				}
				if ready || json.Unmarshal([]byte(frame.Data), &r) != nil || r.Contract != Contract || r.Tenant != in.TenantUUID || r.Subscriber != in.SubscriberID {
					return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
				}
				ready = true
				s.once.Do(func() { close(s.ready) })
				return nil
			case "delivery":
				var d Delivery
				if !ready || json.Unmarshal([]byte(frame.Data), &d) != nil || !validateDelivery(in, d) {
					return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case s.items <- d:
					failures = 0
					return nil
				}
			case "reconnect":
				var r struct {
					Contract string `json:"contract_version"`
				}
				if !ready || json.Unmarshal([]byte(frame.Data), &r) != nil || r.Contract != Contract {
					return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
				}
				return renew
			case "error":
				var r struct {
					Code string `json:"code"`
				}
				if json.Unmarshal([]byte(frame.Data), &r) != nil {
					return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
				}
				return streamError(r.Code, frame.TraceID, frame.RequestID)
			default:
				return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
			}
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, renew) {
			failures = 0
		} else {
			if !recoverable(err) {
				return err
			}
			failures++
			if failures > limit {
				if err != nil {
					return err
				}
				return failure(503, "EVENT_FABRIC_RECONNECT_EXHAUSTED")
			}
		}
		delay := backoff
		for n := 1; n < failures && delay < 5*time.Second; n++ {
			delay *= 2
		}
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
