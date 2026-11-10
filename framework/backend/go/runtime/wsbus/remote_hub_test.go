package wsbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const feedTenantA = "00000000-0000-4000-8000-000000000001"
const feedTenantB = "00000000-0000-4000-8000-000000000002"
const feedNamespace = "fixture.realtime.v1"

type fixtureFeed struct {
	mu         sync.Mutex
	rows       map[string][]RealtimeFrame
	wake       map[string]chan struct{}
	active     int
	calls      int
	publishErr error
	malicious  bool
	retry      int
	cursors    []string
}

func newFixtureFeed() *fixtureFeed {
	return &fixtureFeed{rows: map[string][]RealtimeFrame{}, wake: map[string]chan struct{}{}}
}
func (f *fixtureFeed) Cursor(_ context.Context, tenant, namespace string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := f.rows[tenant]
	if len(rows) == 0 {
		return "0-0", nil
	}
	return rows[len(rows)-1].Cursor, nil
}
func (f *fixtureFeed) PublishEvent(_ context.Context, tenant, namespace string, event RealtimeEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.publishErr != nil {
		return f.publishErr
	}
	cursor := fmt.Sprintf("%d-0", len(f.rows[tenant])+1)
	f.rows[tenant] = append(f.rows[tenant], RealtimeFrame{ContractVersion: RealtimeContract, Namespace: namespace, Cursor: cursor, Event: &event})
	if wake := f.wake[tenant]; wake != nil {
		close(wake)
	}
	f.wake[tenant] = make(chan struct{})
	return nil
}
func (f *fixtureFeed) SubscribeEvents(ctx context.Context, tenant, namespace, cursor string, consume func(RealtimeFrame) error) error {
	f.mu.Lock()
	f.active++
	f.cursors = append(f.cursors, cursor)
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.active--; f.mu.Unlock() }()
	for {
		f.mu.Lock()
		rows := append([]RealtimeFrame{}, f.rows[tenant]...)
		wake := f.wake[tenant]
		if wake == nil {
			wake = make(chan struct{})
			f.wake[tenant] = wake
		}
		malicious := f.malicious
		retry := f.retry
		if retry > 0 {
			f.retry--
		}
		f.mu.Unlock()
		if malicious {
			return consume(RealtimeFrame{ContractVersion: RealtimeContract, Namespace: namespace, Cursor: "9-0", Event: &RealtimeEvent{TenantUUID: feedTenantB, Topic: "fixture.topic", Payload: json.RawMessage(`{}`)}})
		}
		if retry > 0 {
			return ErrRealtimeUnavailable
		}
		for _, row := range rows {
			if compareRealtimeCursor(row.Cursor, cursor) > 0 {
				if err := consume(row); err != nil {
					return err
				}
				cursor = row.Cursor
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wake:
		}
	}
}
func fixtureEvent(tenant, topic, id string) RealtimeEvent {
	return RealtimeEvent{EventID: id, TenantUUID: tenant, Topic: topic, TraceID: "fixture-trace", Payload: json.RawMessage(`{"payload_version":1,"id":9007199254740993}`)}
}
func TestRemoteHubFanoutIsolationDeduplicationAndCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	service := newFixtureFeed()
	a, err := NewRemoteHub(ctx, service, feedNamespace)
	require.NoError(t, err)
	b, err := NewRemoteHub(ctx, service, feedNamespace)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, a.Close()); require.NoError(t, b.Close()) })
	first, second, cross := make(chan Event, 8), make(chan Event, 8), make(chan Event, 8)
	failed := make(chan error, 8)
	errorHandler := func(err error) { failed <- err }
	ua, err := a.SubscribeScoped(ctx, feedTenantA, "fixture.topic", func(ev Event) { first <- ev }, errorHandler)
	require.NoError(t, err)
	ub, err := b.SubscribeScoped(ctx, feedTenantA, "fixture.topic", func(ev Event) { second <- ev }, errorHandler)
	require.NoError(t, err)
	uc, err := b.SubscribeScoped(ctx, feedTenantB, "fixture.topic", func(ev Event) { cross <- ev }, errorHandler)
	require.NoError(t, err)
	id := uuid.NewString()
	require.NoError(t, service.PublishEvent(ctx, feedTenantA, feedNamespace, fixtureEvent(feedTenantA, "fixture.topic", id)))
	for _, ch := range []chan Event{first, second} {
		select {
		case event := <-ch:
			require.Equal(t, id, event.EventID)
			require.Equal(t, json.Number("9007199254740993"), event.Payload.(map[string]any)["id"])
			require.Equal(t, "fixture-trace", event.TraceID)
		case <-ctx.Done():
			t.Fatal("FEED_EVENT_TIMEOUT")
		}
	}
	require.Empty(t, cross)
	// 同一正式 event_id 即使以新的游标重复出现也只交付一次。
	require.NoError(t, service.PublishEvent(ctx, feedTenantA, feedNamespace, fixtureEvent(feedTenantA, "fixture.topic", id)))
	finalID := uuid.NewString()
	require.NoError(t, service.PublishEvent(ctx, feedTenantA, feedNamespace, fixtureEvent(feedTenantA, "fixture.topic", finalID)))
	for _, ch := range []chan Event{first, second} {
		select {
		case event := <-ch:
			require.Equal(t, finalID, event.EventID)
			require.Equal(t, "3-0", event.Cursor)
		case <-ctx.Done():
			t.Fatal("FEED_EVENT_TIMEOUT")
		}
	}
	ua()
	ub()
	uc()
	require.Eventually(t, func() bool { service.mu.Lock(); defer service.mu.Unlock(); return service.active == 0 }, time.Second, 5*time.Millisecond)
	require.Empty(t, first)
	require.Empty(t, second)
	require.Empty(t, cross)
	require.Empty(t, failed)
}
func TestRemoteHubPublishFailureCannotMirrorLocally(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := newFixtureFeed()
	service.publishErr = errors.New("FIXTURE_PUBLISH_DENIED")
	hub, err := NewRemoteHub(ctx, service, feedNamespace)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, hub.Close()) })
	received := make(chan Event, 1)
	un, err := hub.SubscribeScoped(ctx, feedTenantA, "fixture.topic", func(ev Event) { received <- ev }, func(error) {})
	require.NoError(t, err)
	defer un()
	err = hub.Publish(ctx, "fixture.topic", true, PublishOptions{TenantUUID: feedTenantA})
	require.Error(t, err)
	require.Empty(t, received)
	service.mu.Lock()
	require.Equal(t, 1, service.calls)
	service.mu.Unlock()
}
func TestRemoteHubInvalidTenantFrameStopsWithoutDelivery(t *testing.T) {
	service := newFixtureFeed()
	service.malicious = true
	hub, err := NewRemoteHub(context.Background(), service, feedNamespace)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, hub.Close()) })
	failed := make(chan error, 1)
	received := make(chan Event, 1)
	_, err = hub.SubscribeScoped(context.Background(), feedTenantA, "fixture.topic", func(ev Event) { received <- ev }, func(e error) { failed <- e })
	require.NoError(t, err)
	select {
	case err := <-failed:
		require.ErrorIs(t, err, ErrRealtimeInvalid)
	case <-time.After(time.Second):
		t.Fatal("FEED_FAILURE_TIMEOUT")
	}
	require.Empty(t, received)
	require.Eventually(t, func() bool { service.mu.Lock(); defer service.mu.Unlock(); return service.active == 0 }, time.Second, 5*time.Millisecond)
}
func TestRemoteHubBoundedReconnectAndCloseNotifySubscribers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	service := newFixtureFeed()
	service.retry = 2
	hub, err := NewRemoteHub(ctx, service, feedNamespace)
	require.NoError(t, err)
	received, failed := make(chan Event, 1), make(chan error, 1)
	_, err = hub.SubscribeScoped(ctx, feedTenantA, "fixture.topic", func(ev Event) { received <- ev }, func(e error) { failed <- e })
	require.NoError(t, err)
	require.NoError(t, service.PublishEvent(ctx, feedTenantA, feedNamespace, fixtureEvent(feedTenantA, "fixture.topic", uuid.NewString())))
	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("FEED_RECONNECT_TIMEOUT")
	}
	service.mu.Lock()
	require.Len(t, service.cursors, 3)
	require.Equal(t, []string{"0-0", "0-0", "0-0"}, service.cursors)
	service.mu.Unlock()
	require.NoError(t, hub.Close())
	select {
	case err := <-failed:
		require.ErrorIs(t, err, ErrRealtimeUnavailable)
	case <-time.After(time.Second):
		t.Fatal("FEED_CLOSE_TIMEOUT")
	}
	_, err = hub.SubscribeScoped(ctx, feedTenantA, "fixture.topic", func(Event) {}, nil)
	require.Error(t, err)
}
