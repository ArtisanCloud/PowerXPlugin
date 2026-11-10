package wsbus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

// ScopedSubscriber 在真实用户／客户完成授权后才启动指定租户的上行订阅。
type ScopedSubscriber interface {
	SubscribeScoped(context.Context, string, string, func(Event), func(error)) (func(), error)
}

func SubscribeScoped(ctx context.Context, hub LocalHub, tenant, topic string, handler func(Event), failed func(error)) (func(), error) {
	if scoped, ok := hub.(ScopedSubscriber); ok {
		return scoped.SubscribeScoped(ctx, tenant, topic, handler, failed)
	}
	if local, ok := hub.(interface {
		Subscribe(string, func(Event)) func()
	}); ok {
		return local.Subscribe(topic, handler), nil
	}
	return nil, ErrRealtimeUnavailable
}

type remoteSubscription struct {
	topic   string
	handler func(Event)
	failed  func(error)
	active  atomic.Bool
}
type remoteFeed struct {
	seen           map[string]bool
	seenOrder      []string
	tenant, cursor string
	ready          chan struct{}
	err            error
	cancel         context.CancelFunc
	subscriptions  map[uint64]*remoteSubscription
}
type RemoteHub struct {
	service   RealtimeService
	namespace string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	feeds     map[string]*remoteFeed
	next      uint64
	closed    bool
	workers   sync.WaitGroup
}

func NewRemoteHub(ctx context.Context, service RealtimeService, namespace string) (*RemoteHub, error) {
	if service == nil || namespace == "" || len(namespace) > 128 {
		return nil, ErrRealtimeInvalid
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &RemoteHub{service: service, namespace: namespace, ctx: ctx, cancel: cancel, feeds: map[string]*remoteFeed{}}, nil
}

// Publish 只向选定的 Host 发送；不向进程内分发镜像，也不改变失败的传输。
func (h *RemoteHub) Publish(ctx context.Context, topic string, payload any, opts PublishOptions) error {
	if h == nil {
		return ErrRealtimeUnavailable
	}
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if closed {
		return ErrRealtimeUnavailable
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ErrRealtimeInvalid
	}
	return h.service.PublishEvent(ctx, opts.TenantUUID, h.namespace, RealtimeEvent{EventID: uuid.NewString(), TenantUUID: opts.TenantUUID, Topic: topic, MemberUUID: opts.MemberUUID, TraceID: opts.TraceID, Payload: raw})
}
func (h *RemoteHub) SubscribeScoped(ctx context.Context, tenant, topic string, handler func(Event), failed func(error)) (func(), error) {
	if h == nil || handler == nil || !streamTopic.MatchString(topic) || validateRealtimeScope(tenant, h.namespace) != nil {
		return nil, ErrRealtimeInvalid
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrRealtimeUnavailable
	}
	feed := h.feeds[tenant]
	created := feed == nil
	if created {
		if len(h.feeds) >= 32 {
			h.mu.Unlock()
			return nil, errors.New("REALTIME_TENANT_LIMIT")
		}
		feed = &remoteFeed{tenant: tenant, ready: make(chan struct{}), subscriptions: map[uint64]*remoteSubscription{}, seen: map[string]bool{}}
		h.feeds[tenant] = feed
	}
	h.mu.Unlock()
	if created {
		lookup, cancel := context.WithCancel(ctx)
		stop := context.AfterFunc(h.ctx, cancel)
		cursor, err := h.service.Cursor(lookup, tenant, h.namespace)
		stop()
		cancel()
		h.mu.Lock()
		feed.cursor, feed.err = cursor, err
		if err != nil {
			delete(h.feeds, tenant)
		}
		close(feed.ready)
		h.mu.Unlock()
	}
	select {
	case <-h.ctx.Done():
		return nil, ErrRealtimeUnavailable
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-feed.ready:
	}
	h.mu.Lock()
	if feed.err != nil {
		err := feed.err
		h.mu.Unlock()
		return nil, err
	}
	if h.closed || h.feeds[tenant] != feed {
		h.mu.Unlock()
		return nil, ErrRealtimeUnavailable
	}
	if len(feed.subscriptions) >= 512 {
		h.mu.Unlock()
		return nil, errors.New("REALTIME_SUBSCRIPTION_LIMIT")
	}
	h.next++
	id := h.next
	sub := &remoteSubscription{topic: topic, handler: handler, failed: failed}
	sub.active.Store(true)
	feed.subscriptions[id] = sub
	if feed.cancel == nil {
		run, cancel := context.WithCancel(h.ctx)
		feed.cancel = cancel
		h.workers.Add(1)
		go h.run(run, feed)
	}
	h.mu.Unlock()
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			sub.active.Store(false)
			h.mu.Lock()
			delete(feed.subscriptions, id)
			if len(feed.subscriptions) == 0 && h.feeds[tenant] == feed {
				delete(h.feeds, tenant)
				feed.cancel()
			}
			h.mu.Unlock()
		})
	}
	stop := context.AfterFunc(ctx, unsubscribe)
	return func() { stop(); unsubscribe() }, nil
}
func (h *RemoteHub) run(ctx context.Context, feed *remoteFeed) {
	defer h.workers.Done()
	failures := 0
	for ctx.Err() == nil {
		err := h.service.SubscribeEvents(ctx, feed.tenant, h.namespace, feed.cursor, func(frame RealtimeFrame) error {
			if frame.ContractVersion != RealtimeContract || frame.Namespace != h.namespace || frame.Event == nil || frame.Event.TenantUUID != feed.tenant || !streamCursor.MatchString(frame.Cursor) {
				return ErrRealtimeInvalid
			}
			// 重复／乱序只按正式单流游标判断，不能用业务时间猜测顺序。
			if compareRealtimeCursor(frame.Cursor, feed.cursor) <= 0 {
				return nil
			}
			if feed.seen[frame.Event.EventID] {
				feed.cursor = frame.Cursor
				return nil
			}
			var payload any
			decoder := json.NewDecoder(strings.NewReader(string(frame.Event.Payload)))
			decoder.UseNumber()
			if decoder.Decode(&payload) != nil {
				return ErrRealtimeInvalid
			}
			h.mu.Lock()
			subscriptions := make([]*remoteSubscription, 0, len(feed.subscriptions))
			for _, sub := range feed.subscriptions {
				if sub.topic == frame.Event.Topic {
					subscriptions = append(subscriptions, sub)
				}
			}
			h.mu.Unlock()
			event := Event{Topic: frame.Event.Topic, Payload: payload, TenantUUID: feed.tenant, MemberUUID: frame.Event.MemberUUID, TraceID: frame.Event.TraceID, EventID: frame.Event.EventID, Cursor: frame.Cursor}
			for _, sub := range subscriptions {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if sub.active.Load() {
					sub.handler(event)
				}
			}
			feed.seen[frame.Event.EventID] = true
			feed.seenOrder = append(feed.seenOrder, frame.Event.EventID)
			if len(feed.seenOrder) > 2048 {
				delete(feed.seen, feed.seenOrder[0])
				feed.seenOrder = feed.seenOrder[1:]
			}
			feed.cursor = frame.Cursor
			return nil
		})
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			failures = 0
			continue
		}
		var denial *gateway.InvocationError
		fatal := errors.Is(err, ErrRealtimeForbidden) || errors.Is(err, ErrRealtimeInvalid) || errors.Is(err, ErrRealtimeCursorExpired) || (errors.As(err, &denial) && (denial.StatusCode == 401 || denial.StatusCode == 403 || denial.StatusCode == 400))
		failures++
		if fatal || failures > 3 {
			h.fail(feed, err)
			return
		}
		timer := time.NewTimer(time.Duration(1<<(failures-1)) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (h *RemoteHub) fail(feed *remoteFeed, err error) {
	h.mu.Lock()
	if h.feeds[feed.tenant] == feed {
		delete(h.feeds, feed.tenant)
	}
	subs := feed.subscriptions
	feed.subscriptions = map[uint64]*remoteSubscription{}
	feed.cancel()
	h.mu.Unlock()
	for _, sub := range subs {
		if sub.active.Swap(false) && sub.failed != nil {
			sub.failed(err)
		}
	}
}
func (h *RemoteHub) Close() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		h.workers.Wait()
		return nil
	}
	h.closed = true
	h.cancel()
	subs := []*remoteSubscription{}
	for _, feed := range h.feeds {
		if feed.cancel != nil {
			feed.cancel()
		}
		for _, sub := range feed.subscriptions {
			subs = append(subs, sub)
		}
	}
	h.feeds = map[string]*remoteFeed{}
	h.mu.Unlock()
	for _, sub := range subs {
		if sub.active.Swap(false) && sub.failed != nil {
			sub.failed(ErrRealtimeUnavailable)
		}
	}
	h.workers.Wait()
	return nil
}

func compareRealtimeCursor(a, b string) int {
	aa, bb := strings.SplitN(a, "-", 2), strings.SplitN(b, "-", 2)
	for i := 0; i < 2; i++ {
		if len(aa[i]) < len(bb[i]) {
			return -1
		}
		if len(aa[i]) > len(bb[i]) {
			return 1
		}
		if aa[i] < bb[i] {
			return -1
		}
		if aa[i] > bb[i] {
			return 1
		}
	}
	return 0
}
