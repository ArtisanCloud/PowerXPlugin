package eventfabric

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/stretchr/testify/require"
)

const otherTenant = "77777777-7777-4777-8777-777777777777"

func readTopicsRequest(t *testing.T, r *http.Request) (topicInvocation, ResolveTopicsRequest, string) {
	t.Helper()
	var wire topicInvocation
	require.NoError(t, json.NewDecoder(r.Body).Decode(&wire))
	var in struct {
		Tenant     string   `json:"tenant_uuid"`
		Plugin     string   `json:"plugin_id"`
		Keys       []string `json:"topic_keys"`
		Operation  string   `json:"operation"`
		Subscriber string   `json:"subscriber_id"`
	}
	raw, _ := json.Marshal(wire.Payload.Body)
	require.NoError(t, json.Unmarshal(raw, &in))
	return wire, ResolveTopicsRequest{TenantUUID: in.Tenant, PluginID: in.Plugin, SubscriberID: in.Subscriber, CapabilityID: wire.Capability, TopicKeys: in.Keys}, in.Operation
}

func tenantOptions(s *httptest.Server, f fixture, tenant, key string) TenantRuntimeConfig {
	return TenantRuntimeConfig{Connection: Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: key}, HTTPClient: s.Client(), TenantUUID: tenant, PluginID: f.Publisher, PublishTopicKeys: []string{f.Business}}
}

func publishFixture(f fixture, ev event.Event) event.ReliablePublish {
	return event.ReliablePublish{Identity: event.ReliableIdentity{EventID: f.Delivery.EventID, IdempotencyKey: f.Delivery.IdempotencyKey}, Event: ev}
}

func TestTenantRuntimesResolveIndependentCredentialsAndRejectCrossTenantUse(t *testing.T) {
	f := loadFixture(t)
	var mu sync.Mutex
	seen := map[string]string{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, in, operation := readTopicsRequest(t, r)
		if operation == "resolve_topics" {
			expected := "ApiKey tenant-a"
			if in.TenantUUID == otherTenant {
				expected = "ApiKey tenant-b"
			}
			require.Equal(t, expected, r.Header.Get("Authorization"))
			mu.Lock()
			seen[in.TenantUUID] = expected
			mu.Unlock()
			wire(w, ResolvedTopics{Contract, in.TenantUUID, in.PluginID, map[string]string{f.Business: in.TenantUUID + ".powerx.fixture.changed"}}, Protocol, false)
			return
		}
		require.Equal(t, "publish", operation)
		require.NotContains(t, request.Payload.Body, "topic_keys")
		wire(w, f.PublishResult, Protocol, false)
	}))
	defer s.Close()
	a, err := NewTenantHostRuntime(context.Background(), tenantOptions(s, f, f.Tenant, "tenant-a"))
	require.NoError(t, err)
	defer a.Close()
	b, err := NewTenantHostRuntime(context.Background(), tenantOptions(s, f, otherTenant, "tenant-b"))
	require.NoError(t, err)
	defer b.Close()
	require.NotEqual(t, a.TopicBindings()[f.Business], b.TopicBindings()[f.Business])
	require.Len(t, seen, 2)
	copy := a.TopicBindings()
	copy[f.Business] = b.TopicBindings()[f.Business]
	require.NotEqual(t, copy, a.TopicBindings())
	_, err = a.PublishEvent(context.Background(), publishFixture(f, f.Envelope))
	require.NoError(t, err)
	_, err = b.PublishEvent(context.Background(), publishFixture(f, f.Envelope))
	require.Error(t, err)
	in := f.subscription()
	in.TenantUUID = otherTenant
	in.PluginID = f.Publisher
	in.SubscriberID = f.Publisher
	_, err = a.Subscribe(context.Background(), in)
	require.Error(t, err)
}

func TestTenantRuntimeDirectionsAndMappingConflicts(t *testing.T) {
	f := loadFixture(t)
	for _, mode := range []string{"normal", "same-key-conflict", "duplicate-physical"} {
		t.Run(mode, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, in, _ := readTopicsRequest(t, r)
				bindings := map[string]string{}
				for _, key := range in.TopicKeys {
					full := f.Tenant + ".powerx.fixture." + key
					if mode == "same-key-conflict" && in.CapabilityID == ConsumeCapability {
						full += ".other"
					}
					if mode == "duplicate-physical" {
						full = f.Physical
					}
					bindings[key] = full
				}
				wire(w, ResolvedTopics{Contract, f.Tenant, f.Publisher, bindings}, Protocol, false)
			}))
			defer s.Close()
			opts := tenantOptions(s, f, f.Tenant, "fixture")
			opts.PublishTopicKeys = []string{"outbound"}
			opts.ConsumeTopicKeys = []string{"inbound"}
			if mode == "same-key-conflict" {
				opts.ConsumeTopicKeys = []string{"outbound"}
			}
			r, err := NewTenantHostRuntime(context.Background(), opts)
			if mode != "normal" {
				require.Error(t, err)
				require.Nil(t, r)
				return
			}
			require.NoError(t, err)
			defer r.Close()
			ev := f.Envelope
			ev.Topic = "inbound"
			ev.Meta.SourcePlugin = f.Publisher
			_, err = r.PublishEvent(context.Background(), publishFixture(f, ev))
			require.Error(t, err)
			in := f.subscription()
			in.PluginID = f.Publisher
			in.SubscriberID = f.Publisher
			in.Topics = []string{r.TopicBindings()["outbound"]}
			_, err = r.Subscribe(context.Background(), in)
			require.Error(t, err)
		})
	}
}

func TestRuntimeSlotClosesPreviousSubscriptionAndHasNoFailedSwitchFallback(t *testing.T) {
	f := loadFixture(t)
	f.Subscriber = f.Publisher
	closed := make(chan struct{})
	var closeOnce sync.Once
	var denied atomic.Bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tenant/invocations/stream" {
			_, _ = io.Copy(io.Discard, r.Body)
			ready(w, f)
			<-r.Context().Done()
			closeOnce.Do(func() { close(closed) })
			return
		}
		_, in, _ := readTopicsRequest(t, r)
		if in.TenantUUID == otherTenant {
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Error("OLD_SUBSCRIPTION_NOT_CLOSED")
			}
		}
		if denied.Load() {
			w.WriteHeader(409)
			_ = json.NewEncoder(w).Encode(map[string]string{"error_code": "EVENT_FABRIC_INSTALL_PREPARATION_REQUIRED"})
			return
		}
		wire(w, ResolvedTopics{Contract, in.TenantUUID, in.PluginID, map[string]string{f.Business: in.TenantUUID + ".powerx.fixture.changed"}}, Protocol, false)
	}))
	defer s.Close()
	var slot RuntimeSlot
	defer slot.Close()
	first := tenantOptions(s, f, f.Tenant, "tenant-a")
	first.PublishTopicKeys = nil
	first.ConsumeTopicKeys = []string{f.Business}
	a, err := slot.Switch(context.Background(), first)
	require.NoError(t, err)
	in := f.subscription()
	in.Topics = []string{a.TopicBindings()[f.Business]}
	in.ExpectedPublishers = map[string][]string{in.Topics[0]: {f.Publisher}}
	sub, err := a.Subscribe(context.Background(), in)
	require.NoError(t, err)
	b, err := slot.Switch(context.Background(), tenantOptions(s, f, otherTenant, "tenant-b"))
	require.NoError(t, err)
	require.False(t, a.active())
	require.True(t, b.active())
	_, err = sub.Recv(context.Background())
	require.Error(t, err)
	denied.Store(true)
	next, err := slot.Switch(context.Background(), first)
	require.Error(t, err)
	require.Nil(t, next)
	require.False(t, b.active())
	require.Nil(t, slot.current)
	require.NoError(t, slot.Close())
	_, err = slot.Switch(context.Background(), first)
	require.Error(t, err)
}

func TestRuntimeSlotCloseCancelsPendingResolve(t *testing.T) {
	f := loadFixture(t)
	started := make(chan struct{})
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer s.Close()
	defer close(release)
	var slot RuntimeSlot
	done := make(chan error, 1)
	go func() {
		_, err := slot.Switch(context.Background(), tenantOptions(s, f, f.Tenant, "fixture"))
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("RESOLVE_NOT_STARTED")
	}
	require.NoError(t, slot.Close())
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("RESOLVE_NOT_CANCELLED")
	}
}

func TestStaticRuntimeRequiresExplicitCredentialTenant(t *testing.T) {
	f := loadFixture(t)
	_, err := NewHostRuntime(Config{BaseURL: "http://127.0.0.1", AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Publisher, "", map[string]string{f.Business: f.Physical}, nil)
	require.Error(t, err)
	_, err = NewHostRuntime(Config{BaseURL: "http://127.0.0.1", AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Publisher, otherTenant, map[string]string{f.Business: f.Physical}, nil)
	require.Error(t, err)
}
