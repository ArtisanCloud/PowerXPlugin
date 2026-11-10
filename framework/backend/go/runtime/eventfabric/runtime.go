package eventfabric

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
)

// Runtime is assembled once. There is deliberately no WSBus/local adapter or
// inbound-user credential lookup in this service event runtime.
type Runtime struct {
	Client         *Client
	plugin, tenant string
	topics         map[string]string
	publishTopics  map[string]bool
	consumeTopics  map[string]bool
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	closed         bool
	subscriptions  map[*ownedSubscription]struct{}
	workers        sync.WaitGroup
}

func (*Runtime) ReliableHost() bool { return true }

func NewHostRuntime(cfg Config, tokens TokenProvider, plugin, tenant string, topics map[string]string, h *http.Client) (*Runtime, error) {
	if !identity(tenant, plugin) || len(topics) == 0 {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	copyTopics := map[string]string{}
	physical := map[string]bool{}
	for business, full := range topics {
		if !valid(business, 255) || !physicalTopic(tenant, full) || len(full) <= len(tenant)+1 || physical[full] {
			return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
		}
		copyTopics[business] = full
		physical[full] = true
	}
	var client *Client
	var err error
	if tokens != nil {
		client, err = NewClientWithTokenProvider(cfg, tokens, h)
	} else {
		client, err = NewClient(cfg, h)
	}
	if err != nil {
		return nil, err
	}
	return newRuntime(client, plugin, tenant, copyTopics), nil
}

func newRuntime(client *Client, plugin, tenant string, topics map[string]string) *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{Client: client, plugin: plugin, tenant: tenant, topics: topics, ctx: ctx, cancel: cancel, subscriptions: map[*ownedSubscription]struct{}{}}
}
func (r *Runtime) active() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.closed
}
func (r *Runtime) Emit(ctx context.Context, ev event.Event) error {
	if ev.Reliability == nil {
		return failure(400, "EVENT_FABRIC_STABLE_ID_REQUIRED")
	}
	_, err := r.PublishEvent(ctx, event.ReliablePublish{Identity: *ev.Reliability, Event: ev})
	return err
}
func (r *Runtime) PublishEvent(ctx context.Context, in event.ReliablePublish) (*PublishReceipt, error) {
	if r == nil {
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.workers.Add(1)
	r.mu.Unlock()
	defer r.workers.Done()
	run, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	defer cancel()
	ev := in.Event
	full, ok := r.topics[string(ev.Topic)]
	if !ok || (r.publishTopics != nil && !r.publishTopics[string(ev.Topic)]) || ev.Meta.SourcePlugin != r.plugin || ev.Meta.TenantUUID != r.tenant || !identity(ev.Meta.TenantUUID, ev.Meta.SourcePlugin) || !physicalTopic(ev.Meta.TenantUUID, full) || !valid(ev.Meta.RequestID, 256) || !valid(ev.Meta.TraceID, 256) || ev.Meta.OccurredAt.IsZero() || !valid(ev.Meta.PayloadVersion, 32) || !json.Valid(ev.Payload) {
		return nil, failure(400, "EVENT_FABRIC_ENVELOPE_INVALID")
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return nil, failure(400, "EVENT_FABRIC_ENVELOPE_INVALID")
	}
	// The entire original envelope is opaque to Core. Its trace and business
	// topic are never rewritten using the physical transport topic or Core trace.
	return r.Client.Publish(run, PublishRequest{TenantUUID: ev.Meta.TenantUUID, PluginID: r.plugin, Event: Event{EventID: in.Identity.EventID, IdempotencyKey: in.Identity.IdempotencyKey, Topic: full, Version: ev.Meta.PayloadVersion, Payload: raw}})
}
func (r *Runtime) DecodeEvent(d Delivery) (event.Event, error) {
	var ev event.Event
	decoder := json.NewDecoder(bytes.NewReader(d.Payload))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&ev) != nil || decoder.Decode(new(any)) != io.EOF || r == nil || d.ContractVersion != Contract || d.SubscriberID != r.plugin || r.tenant != d.TenantUUID || r.topics[string(ev.Topic)] != d.Topic || (r.consumeTopics != nil && !r.consumeTopics[d.Topic]) || ev.Meta.TenantUUID != d.TenantUUID || ev.Meta.SourcePlugin != d.PublisherPluginID || ev.Meta.PayloadVersion != d.Version || ev.Meta.OccurredAt.IsZero() || !valid(ev.Meta.RequestID, 256) || !valid(ev.Meta.TraceID, 256) || !json.Valid(ev.Payload) {
		return event.Event{}, failure(502, "EVENT_FABRIC_ENVELOPE_INVALID")
	}
	ev.Reliability = &event.ReliableIdentity{EventID: d.EventID, IdempotencyKey: d.IdempotencyKey}
	return ev, nil
}

type ownedSubscription struct {
	Subscription
	owner *Runtime
	once  sync.Once
}

func (s *ownedSubscription) Close() error {
	var err error
	s.once.Do(func() {
		err = s.Subscription.Close()
		s.owner.mu.Lock()
		delete(s.owner.subscriptions, s)
		s.owner.mu.Unlock()
	})
	return err
}
func (r *Runtime) Subscribe(ctx context.Context, in SubscribeRequest) (Subscription, error) {
	if r == nil {
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.workers.Add(1)
	r.mu.Unlock()
	defer r.workers.Done()
	if in.PluginID != r.plugin || in.TenantUUID != r.tenant {
		return nil, failure(403, "EVENT_FABRIC_SUBSCRIBER_FORBIDDEN")
	}
	for _, topic := range in.Topics {
		if r.consumeTopics != nil && !r.consumeTopics[topic] {
			return nil, failure(403, "EVENT_FABRIC_TOPIC_FORBIDDEN")
		}
		found := false
		for _, full := range r.topics {
			if full == topic {
				found = true
			}
		}
		if !found {
			return nil, failure(403, "EVENT_FABRIC_TOPIC_FORBIDDEN")
		}
	}
	run, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(r.ctx, cancel)
	sub, err := r.Client.Subscribe(run, in)
	if err != nil {
		stop()
		cancel()
		return nil, err
	}
	owned := &ownedSubscription{Subscription: &cancelSubscription{Subscription: sub, cancel: cancel, stop: stop}, owner: r}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		_ = owned.Subscription.Close()
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.subscriptions[owned] = struct{}{}
	r.mu.Unlock()
	return owned, nil
}

type cancelSubscription struct {
	Subscription
	cancel context.CancelFunc
	stop   func() bool
}

func (s *cancelSubscription) Close() error { s.stop(); s.cancel(); return s.Subscription.Close() }
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	r.closed = true
	r.cancel()
	subs := make([]*ownedSubscription, 0, len(r.subscriptions))
	for sub := range r.subscriptions {
		subs = append(subs, sub)
	}
	r.mu.Unlock()
	for _, sub := range subs {
		_ = sub.Close()
	}
	r.workers.Wait()
	return nil
}

// RunConsumer invokes apply serially. A nil return from apply must mean that
// business effects AND the durable deduplication receipt committed together.
// It stops on uncertain ACK; a later delivery must go through business dedupe.
func (r *Runtime) RunConsumer(ctx context.Context, in SubscribeRequest, apply func(context.Context, event.Event) error) error {
	if apply == nil {
		return failure(400, "EVENT_FABRIC_HANDLER_REQUIRED")
	}
	if r == nil {
		return failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	r.workers.Add(1)
	r.mu.Unlock()
	defer r.workers.Done()
	run, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	defer cancel()
	ctx = run
	sub, err := r.Subscribe(ctx, in)
	if err != nil {
		return err
	}
	defer sub.Close()
	for {
		d, err := sub.Recv(ctx)
		if err != nil {
			return err
		}
		ev, err := r.DecodeEvent(d)
		if err != nil {
			return err
		}
		work, cancel := context.WithDeadline(ctx, d.AckDeadline)
		err = apply(work, ev)
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if _, settleErr := r.Client.Nack(ctx, d.Receipt(), "consumer_failed"); settleErr != nil {
				return settleErr
			}
			continue
		}
		if err = r.Client.Ack(ctx, d.Receipt()); err != nil {
			return err
		}
	}
}
