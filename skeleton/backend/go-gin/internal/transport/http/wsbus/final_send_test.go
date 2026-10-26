package wsbus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	contracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	realtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	wsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	app "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gorilla/websocket"
)

type finalSendRevalidator struct {
	identity contracts.DelegatedIdentity
	change   func(context.Context, *contracts.WSIdentity) error
	calls    atomic.Int64
}

func (r *finalSendRevalidator) RevalidateWSIdentity(ctx context.Context, _ contracts.WSIdentityQuery) (*contracts.WSIdentity, error) {
	r.calls.Add(1)
	identity := r.identity
	identity.Schema = contracts.WSIdentityResultSchema
	identity.ProofExpiresAt = identity.UserSessionExpiresAt
	result := &contracts.WSIdentity{DelegatedIdentity: identity}
	if err := r.change(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Exercise the real send lock and WriteJSON path. The subscription succeeds
// before a queued event's final identity decision becomes unavailable.
func TestFinalSendRevalidatesAfterQueueAndReleasesConnection(t *testing.T) {
	for _, scenario := range []string{"allowed", "revoked", "unavailable", "cancelled", "expired", "late_success", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			now := time.Now().UTC()
			expiry := now.Add(time.Minute)
			member := "33333333-3333-4333-8333-333333333333"
			identity := contracts.DelegatedIdentity{Schema: contracts.IdentityResultSchema, Mode: "delegated", PluginID: app.PluginID, TenantUUID: "11111111-1111-4111-8111-111111111111", UserUUID: "22222222-2222-4222-8222-222222222222", MemberUUID: &member, Roles: []string{}, Permissions: []string{"topic:read"}, PolicyVersion: "p1", PermsHash: "hash", SourceMethod: "GET", SourcePath: "/ws", Transport: "ws_upgrade", SourceRequestID: "source", SourceTraceID: "source", TraceID: "resolve", ProofExpiresAt: expiry, UserSessionExpiresAt: expiry, WSRevalidationHandle: "pxwsi_v1_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", WSRevalidationExpiresAt: &expiry}
			lifecycle := realtime.NewIdentityLifecycle()
			defer lifecycle.Close()
			hub := wsbus.NewMemoryHub()
			topic := "_topic.final.member_" + member
			ready := make(chan *wsConn, 1)
			closed := make(chan struct{})
			var rejected atomic.Bool
			var cancel context.CancelFunc
			var lease *realtime.IdentityLease
			revalidator := &finalSendRevalidator{identity: identity}
			revalidator.change = func(ctx context.Context, current *contracts.WSIdentity) error {
				if !rejected.Load() {
					return nil
				}
				switch scenario {
				case "revoked":
					current.Permissions = []string{}
				case "unavailable":
					return iamerrors.New(iamerrors.CodeContextUnavailable, iamerrors.CodeContextUnavailable)
				case "cancelled":
					cancel()
				case "expired":
					<-lease.Done()
				case "late_success":
					lifecycle.Revoke(realtime.IdentityRevocation{MemberUUID: member})
				case "timeout":
					return context.DeadlineExceeded
				}
				return nil
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				conn, err := upgrader.Upgrade(w, request, nil)
				if err != nil {
					t.Error(err)
					return
				}
				ctx, stop := context.WithCancel(context.Background())
				cancel = stop
				defer stop()
				ttl := time.Minute
				if scenario == "expired" {
					ttl = 200 * time.Millisecond
				}
				lease, err = lifecycle.Open(realtime.IdentityLeaseOptions{Revalidator: revalidator, Identity: &identity, PluginID: app.PluginID, TTL: ttl, ObservedRevision: lifecycle.Revision(), Close: func(string) { stop() }, AuthorizeTopic: func(v *contracts.DelegatedIdentity, actual string) bool {
					return actual == topic && v.HasPermission("topic:read")
				}})
				if err != nil {
					t.Error(err)
					_ = conn.Close()
					return
				}
				defer lease.Close(realtime.WSIdentityRevoked)
				ws := &wsConn{ctx: ctx, lease: lease, conn: conn, subs: map[string]func(){}, tenantUUID: identity.TenantUUID, memberUUID: member}
				defer close(closed)
				defer ws.close()
				descriptors := []realtime.Descriptor{{Key: topic, Protocols: []realtime.Protocol{realtime.ProtocolWS}, Actions: []realtime.Action{realtime.ActionSubscribe}, Scope: realtime.ScopeMember, EventTypes: []string{"message"}}}
				if len(ws.subscribe(hub, []string{topic}, descriptors)) != 1 {
					t.Error("IAM_WS_TOPIC_FORBIDDEN")
					return
				}
				ready <- ws
				<-ctx.Done()
			}))
			socket, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				server.Close()
				t.Fatal(err)
			}
			ws := <-ready
			t.Cleanup(func() { cancel(); _ = socket.Close(); server.Close() })
			ws.sendMu.Lock()
			started, completed := make(chan struct{}), make(chan struct{})
			go func() {
				defer close(completed)
				close(started)
				_ = hub.Publish(context.Background(), topic, map[string]any{"event_uuid": "event-1"}, wsbus.PublishOptions{TenantUUID: identity.TenantUUID, MemberUUID: member})
			}()
			<-started
			rejected.Store(true)
			ws.sendMu.Unlock()
			select {
			case <-completed:
			case <-time.After(2 * time.Second):
				t.Fatal("WS_SEND_TIMEOUT")
			}
			_ = socket.SetReadDeadline(time.Now().Add(time.Second))
			var message wsResponse
			err = socket.ReadJSON(&message)
			if scenario == "allowed" {
				if err != nil || message.Type != "event" {
					t.Fatal(err, message)
				}
				cancel()
			} else if err == nil {
				t.Fatal(message)
			}
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Fatal("WS_CLOSE_TIMEOUT")
			}
			if revalidator.calls.Load() != 2 || hub.Subscribers(topic) != 0 {
				t.Fatal(revalidator.calls.Load(), hub.Subscribers(topic))
			}
		})
	}
}
