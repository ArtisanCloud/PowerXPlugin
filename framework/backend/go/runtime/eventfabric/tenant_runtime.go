package eventfabric

import (
	"context"
	"net/http"
	"sync"
)

// TenantRuntimeConfig must be assembled from the target tenant's service
// credential, never from an inbound user token or System's topic_bindings.
type TenantRuntimeConfig struct {
	Connection                         Config
	TokenProvider                      TokenProvider
	HTTPClient                         *http.Client
	PluginID, TenantUUID               string
	PublishTopicKeys, ConsumeTopicKeys []string
}

// NewTenantHostRuntime resolves all requested directions before exposing a
// Runtime. A failed direction discards the entire assembly, without a fallback.
func NewTenantHostRuntime(ctx context.Context, options TenantRuntimeConfig) (*Runtime, error) {
	if ctx == nil || !identity(options.TenantUUID, options.PluginID) ||
		(len(options.PublishTopicKeys) == 0 && len(options.ConsumeTopicKeys) == 0) ||
		(len(options.PublishTopicKeys) > 0 && !validTopicKeys(options.PublishTopicKeys)) ||
		(len(options.ConsumeTopicKeys) > 0 && !validTopicKeys(options.ConsumeTopicKeys)) {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var client *Client
	var err error
	if options.TokenProvider != nil {
		client, err = NewClientWithTokenProvider(options.Connection, options.TokenProvider, options.HTTPClient)
	} else {
		client, err = NewClient(options.Connection, options.HTTPClient)
	}
	if err != nil {
		return nil, err
	}
	topics := map[string]string{}
	physical := map[string]string{}
	publish, consume := map[string]bool{}, map[string]bool{}
	for _, direction := range []struct {
		capability string
		keys       []string
	}{
		{PublishCapability, append([]string(nil), options.PublishTopicKeys...)},
		{ConsumeCapability, append([]string(nil), options.ConsumeTopicKeys...)},
	} {
		if len(direction.keys) == 0 {
			continue
		}
		in := ResolveTopicsRequest{TenantUUID: options.TenantUUID, PluginID: options.PluginID, CapabilityID: direction.capability, TopicKeys: direction.keys}
		if direction.capability == ConsumeCapability {
			in.SubscriberID = options.PluginID
		}
		resolved, err := client.ResolveTopics(ctx, in)
		if err != nil {
			return nil, err
		}
		for key, full := range resolved.TopicBindings {
			if (topics[key] != "" && topics[key] != full) || (physical[full] != "" && physical[full] != key) {
				return nil, failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
			}
			topics[key], physical[full] = full, key
			if direction.capability == PublishCapability {
				publish[key] = true
			} else {
				consume[full] = true
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runtime := newRuntime(client, options.PluginID, options.TenantUUID, topics)
	runtime.publishTopics, runtime.consumeTopics = publish, consume
	return runtime, nil
}

// TopicBindings returns an owned snapshot; callers cannot mutate live routing.
func (r *Runtime) TopicBindings() map[string]string {
	out := map[string]string{}
	if r != nil {
		for key, full := range r.topics {
			out[key] = full
		}
	}
	return out
}

// RuntimeSlot owns one active tenant context. Applications processing tenants
// concurrently should instead own one independently-created Runtime per tenant.
// A failed switch leaves the slot empty, never using the previous tenant.
type RuntimeSlot struct {
	switchMu sync.Mutex
	mu       sync.Mutex
	current  *Runtime
	pending  context.CancelFunc
	closed   bool
}

func (s *RuntimeSlot) Switch(ctx context.Context, options TenantRuntimeConfig) (*Runtime, error) {
	if s == nil || ctx == nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	s.switchMu.Lock()
	defer s.switchMu.Unlock()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	run, cancel := context.WithCancel(ctx)
	s.pending = cancel
	previous := s.current
	s.current = nil
	s.mu.Unlock()
	defer cancel()
	if previous != nil {
		_ = previous.Close()
	}
	selected, err := NewTenantHostRuntime(run, options)
	s.mu.Lock()
	s.pending = nil
	closed := s.closed
	if err == nil && !closed {
		s.current = selected
	}
	s.mu.Unlock()
	if closed {
		if selected != nil {
			_ = selected.Close()
		}
		return nil, failure(503, "EVENT_FABRIC_RUNTIME_CLOSED")
	}
	return selected, err
}

func (s *RuntimeSlot) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.closed = true
	if s.pending != nil {
		s.pending()
	}
	current := s.current
	s.current = nil
	s.mu.Unlock()
	if current != nil {
		_ = current.Close()
	}
	// Wait for an in-flight switch to release its old or partially built runtime.
	s.switchMu.Lock()
	s.switchMu.Unlock()
	return nil
}
