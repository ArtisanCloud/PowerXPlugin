package eventfabric

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/stretchr/testify/require"
)

func TestRuntimePreservesEnvelopeAndRequiresOutboxIdentity(t *testing.T) {
	f := loadFixture(t)
	var captured Event
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Payload struct {
				Body struct {
					Event Event `json:"event"`
				} `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		captured = body.Payload.Body.Event
		wire(w, f.PublishResult, Protocol, false)
	}))
	defer s.Close()
	r, err := NewHostRuntime(Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Publisher, f.Tenant, map[string]string{f.Business: f.Physical}, s.Client())
	require.NoError(t, err)
	defer r.Close()
	err = r.Emit(context.Background(), f.Envelope)
	require.Error(t, err)
	require.Empty(t, captured.EventID)
	ev := f.Envelope
	ev.Reliability = &event.ReliableIdentity{EventID: f.Delivery.EventID, IdempotencyKey: f.Delivery.IdempotencyKey}
	require.NoError(t, r.Emit(context.Background(), ev))
	require.Equal(t, "1", captured.Version)
	require.Equal(t, f.Physical, captured.Topic)
	var inner event.Event
	require.NoError(t, json.Unmarshal(captured.Payload, &inner))
	require.Equal(t, f.Envelope.Meta, inner.Meta)
	require.Equal(t, f.Envelope.Topic, inner.Topic)
	require.JSONEq(t, string(f.Envelope.Payload), string(inner.Payload))
	receiver, err := NewHostRuntime(Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Subscriber, f.Tenant, map[string]string{f.Business: f.Physical}, s.Client())
	require.NoError(t, err)
	defer receiver.Close()
	decoded, err := receiver.DecodeEvent(f.Delivery)
	require.NoError(t, err)
	require.Equal(t, f.Envelope.Meta, decoded.Meta)
	require.Equal(t, f.Envelope.Topic, decoded.Topic)
	require.Equal(t, f.Delivery.EventID, decoded.Reliability.EventID)
	require.NotEqual(t, f.Delivery.TraceID, decoded.Meta.TraceID)
	bad := f.Delivery
	bad.PublisherPluginID = "foreign"
	_, err = receiver.DecodeEvent(bad)
	require.Error(t, err)
	require.NoError(t, r.Close())
	require.Error(t, r.Emit(context.Background(), ev))
}
func TestConsumerCommitsThenAckAndDeduplicatesRedelivery(t *testing.T) {
	f := loadFixture(t)
	var streamCount, acks, nacks, business atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tenant/invocations/stream" {
			ready(w, f)
			d := f.Delivery
			if streamCount.Add(1) > 1 {
				d.DeliveryID = "66666666-6666-4666-8666-666666666666"
				d.Attempt = 2
			}
			frame(w, "delivery", d)
			<-r.Context().Done()
			return
		}
		var request struct {
			Payload struct {
				Body struct {
					Operation  string `json:"operation"`
					DeliveryID string `json:"delivery_id"`
				} `json:"body"`
			} `json:"payload"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		if request.Payload.Body.Operation == "nack" {
			nacks.Add(1)
		}
		acks.Add(1)
		require.EqualValues(t, 1, business.Load())
		if acks.Load() == 1 {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"error_code":"EVENT_FABRIC_UNAVAILABLE"}`))
			return
		}
		out := f.AckResult
		out.DeliveryID = request.Payload.Body.DeliveryID
		wire(w, out, Protocol, false)
	}))
	defer s.Close()
	r, err := NewHostRuntime(Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Subscriber, f.Tenant, map[string]string{f.Business: f.Physical}, s.Client())
	require.NoError(t, err)
	defer r.Close()
	committed := false
	apply := func(_ context.Context, ev event.Event) error {
		require.Equal(t, f.Delivery.EventID, ev.Reliability.EventID)
		if !committed {
			committed = true
			business.Add(1)
		}
		return nil
	}
	err = r.RunConsumer(context.Background(), f.subscription(), apply)
	var uncertain *SettlementError
	require.ErrorAs(t, err, &uncertain)
	require.True(t, uncertain.Uncertain)
	require.True(t, committed)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.RunConsumer(ctx, f.subscription(), apply) }()
	require.Eventually(t, func() bool { return acks.Load() == 2 }, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EVENT_FABRIC_WORKER_LEAK")
	}
	require.EqualValues(t, 1, business.Load())
	require.Zero(t, nacks.Load())
}
func TestRuntimeCloseStopsOwnedSubscriptions(t *testing.T) {
	f := loadFixture(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ready(w, f); <-r.Context().Done() }))
	defer s.Close()
	r, err := NewHostRuntime(Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Subscriber, f.Tenant, map[string]string{f.Business: f.Physical}, s.Client())
	require.NoError(t, err)
	sub, err := r.Subscribe(context.Background(), f.subscription())
	require.NoError(t, err)
	require.NoError(t, r.Close())
	_, err = sub.Recv(context.Background())
	require.Error(t, err)
	_, err = r.Subscribe(context.Background(), f.subscription())
	require.Error(t, err)
}

func TestRuntimeCloseCancelsAndWaitsForBusinessWorker(t *testing.T) {
	f := loadFixture(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready(w, f)
		frame(w, "delivery", f.Delivery)
		<-r.Context().Done()
	}))
	defer s.Close()
	r, err := NewHostRuntime(Config{BaseURL: s.URL, AuthScheme: "apikey", APIKey: "fixture"}, nil, f.Subscriber, f.Tenant, map[string]string{f.Business: f.Physical}, s.Client())
	require.NoError(t, err)
	started, done := make(chan struct{}), make(chan error, 1)
	go func() {
		done <- r.RunConsumer(context.Background(), f.subscription(), func(ctx context.Context, _ event.Event) error { close(started); <-ctx.Done(); return ctx.Err() })
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("EVENT_FABRIC_WORKER_NOT_STARTED")
	}
	require.NoError(t, r.Close())
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("EVENT_FABRIC_WORKER_LEAK")
	}
}
