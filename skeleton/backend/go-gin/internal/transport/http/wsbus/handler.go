package wsbus

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiamdelegated "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	fwiamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	fwidentity "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type wsRequest struct {
	Type   string   `json:"type"`
	Topics []string `json:"topics"`
}

type wsResponse struct {
	Type    string `json:"type"`
	Topic   string `json:"topic,omitempty"`
	Message string `json:"message,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

type wsSubscriber interface {
	Subscribe(topic string, handler func(fwwsbus.Event)) func()
}

type wsConn struct {
	ctx        context.Context
	lease      *frameworkrealtime.IdentityLease
	conn       *websocket.Conn
	sendMu     sync.Mutex
	subs       map[string]func()
	tenantUUID string
	memberUUID string
	sendHook   func(wsResponse)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func RegisterRoutes(r *gin.Engine, deps *app.Deps, jwtCfg middleware.JWTAuthConfig, prefixes ...string) {
	if r == nil {
		return
	}
	prefix := "/api/v1"
	if len(prefixes) > 0 {
		prefix = prefixes[0]
	}
	if deps != nil && deps.IAMAdapterMode.String() == "delegated" {
		r.GET(path.Join(prefix, "ws"), Handler(deps, jwtCfg, prefix))
		return
	}
	r.GET(path.Join("/api", "ws"), Handler(deps, jwtCfg))
}

func Handler(deps *app.Deps, jwtCfg middleware.JWTAuthConfig, apiPrefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.WSBusHub == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ws bus is not configured"})
			return
		}
		subscriber, ok := deps.WSBusHub.(wsSubscriber)
		if !ok {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ws bus does not support subscriptions"})
			return
		}

		tenantUUID, memberUUID, ok := resolveIdentity(c, jwtCfg)
		if !ok && !jwtCfg.Optional {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ws := &wsConn{subs: make(map[string]func()), tenantUUID: tenantUUID, memberUUID: memberUUID}
		stopCtx, stop := context.WithCancel(c.Request.Context())
		ws.ctx = stopCtx
		defer stop()
		var localExpiry time.Time
		if deps.IAMAdapterMode.String() == "delegated" {
			if deps.WSIdentityLifecycle == nil {
				fwidentity.RespondIdentityError(c, fwiamerrors.New(fwiamerrors.CodeAdapterNotBound, fwiamerrors.CodeAdapterNotBound))
				return
			}
			// Core's authenticated business Bearer is authoritative only for this
			// initial assertion. The user context is then replaced by the Core proof.
			raw := c.GetHeader("Authorization")
			tc, _, valid := middleware.ParseFromHeaders(func(name string) string {
				if strings.EqualFold(name, "Authorization") {
					return raw
				}
				return c.GetHeader(name)
			}, jwtCfg)
			if !valid {
				fwidentity.RespondIdentityError(c, fwiamerrors.New(fwiamerrors.CodeUnauthorized, fwiamerrors.CodeUnauthorized))
				return
			}
			middleware.SetTenantContext(c, tc)
			prefix := "/api/v1"
			if len(apiPrefixes) > 0 {
				prefix = apiPrefixes[0]
			}
			revision := deps.WSIdentityLifecycle.Revision()
			identity, err := fwidentity.ResolveRequestIdentity(c, httpmw.DelegatedIdentityOptions(deps, prefix), "ws_upgrade")
			if err != nil {
				fwidentity.RespondIdentityError(c, err)
				return
			}
			revalidator, err := deps.IAMRegistry.WSIdentityRevalidator()
			if err != nil {
				fwidentity.RespondIdentityError(c, err)
				return
			}
			ws.tenantUUID = identity.TenantUUID
			ws.memberUUID = ""
			if identity.MemberUUID != nil {
				ws.memberUUID = *identity.MemberUUID
			}
			lease, err := deps.WSIdentityLifecycle.Open(frameworkrealtime.IdentityLeaseOptions{Revalidator: revalidator, Identity: identity, PluginID: app.PluginID, TTL: time.Minute, ObservedRevision: revision, Close: func(string) { stop() },
				AuthorizeTopic: func(v *fwiamcontracts.DelegatedIdentity, topic string) bool {
					member := ""
					if v.MemberUUID != nil {
						member = *v.MemberUUID
					}
					decision := frameworkrealtime.Decide(deps.RealtimeDescriptors, frameworkrealtime.ActionSubscribe, topic, frameworkrealtime.ProtocolWS, "message", frameworkrealtime.Scope{TenantUUID: v.TenantUUID, MemberUUID: member})
					return decision.Allowed && (decision.RequiredPermission == "" || v.HasPermission(decision.RequiredPermission))
				},
			})
			if err != nil {
				fwidentity.RespondIdentityError(c, err)
				return
			}
			ws.lease = lease
			defer lease.Close(frameworkrealtime.WSIdentityRevoked)
		} else {
			authz := c.GetHeader("Authorization")
			if authz == "" {
				authz = c.Query("authorization")
			}
			tc, _, valid := middleware.ParseFromHeaders(func(name string) string {
				if strings.EqualFold(name, "Authorization") {
					return authz
				}
				return c.GetHeader(name)
			}, jwtCfg)
			if !valid || tc.AccessExpiresAt.IsZero() || !time.Now().Before(tc.AccessExpiresAt) {
				c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "TOKEN_EXPIRED"}})
				return
			}
			localExpiry = tc.AccessExpiresAt
		}
		if stopCtx.Err() != nil {
			fwidentity.RespondIdentityError(c, fwiamerrors.New(frameworkrealtime.WSIdentityRevoked, frameworkrealtime.WSIdentityRevoked))
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		ws.conn = conn
		if !localExpiry.IsZero() {
			timer := time.AfterFunc(time.Until(localExpiry), stop)
			defer timer.Stop()
		}
		go func() { <-stopCtx.Done(); _ = conn.Close() }()
		defer ws.close()

		for {
			var req wsRequest
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			switch strings.ToLower(strings.TrimSpace(req.Type)) {
			case "subscribe":
				if len(req.Topics) == 0 && ws.lease != nil {
					if _, err := ws.lease.Revalidate(stopCtx); err != nil {
						ws.recordIdentityError(err)
						return
					}
				}
				accepted := ws.subscribe(subscriber, req.Topics, deps.RealtimeDescriptors)
				if stopCtx.Err() != nil {
					return
				}
				ws.send(wsResponse{Type: "ack", Message: "subscribed", Payload: gin.H{"topics": accepted}})
			case "unsubscribe":
				if ws.lease != nil {
					if _, err := ws.lease.Revalidate(stopCtx); err != nil {
						ws.recordIdentityError(err)
						return
					}
				}
				ws.unsubscribe(req.Topics)
				ws.send(wsResponse{Type: "ack", Message: "unsubscribed", Payload: gin.H{"topics": req.Topics}})
			default:
				ws.send(wsResponse{Type: "error", Message: "unknown message type"})
			}
		}
	}
}

func resolveIdentity(c *gin.Context, jwtCfg middleware.JWTAuthConfig) (tenantUUID string, memberUUID string, ok bool) {
	authz := strings.TrimSpace(c.GetHeader("Authorization"))
	if authz == "" {
		authz = strings.TrimSpace(c.Query("authorization"))
	}
	header := func(name string) string {
		if strings.EqualFold(name, "Authorization") && authz != "" {
			return authz
		}
		return c.GetHeader(name)
	}
	tc, _, ok := middleware.ParseFromHeaders(header, jwtCfg)
	if ok && strings.TrimSpace(tc.TenantUUID) != "" {
		return strings.TrimSpace(tc.TenantUUID), strings.TrimSpace(tc.MemberUUID), true
	}
	return "", "", false
}

func (w *wsConn) subscribe(subscriber wsSubscriber, topics []string, descriptors []frameworkrealtime.Descriptor) []string {
	accepted := []string{}
	for _, topic := range topics {
		clean := strings.TrimSpace(topic)
		if clean == "" {
			continue
		}
		if w.lease != nil {
			if err := w.lease.AuthorizeTopic(w.ctx, clean); err != nil {
				w.recordIdentityError(err)
				return accepted
			}
		}
		if _, exists := w.subs[clean]; exists {
			accepted = append(accepted, clean)
			continue
		}
		decision := frameworkrealtime.Decide(descriptors, frameworkrealtime.ActionSubscribe, clean, frameworkrealtime.ProtocolWS, "message", frameworkrealtime.Scope{TenantUUID: w.tenantUUID, MemberUUID: w.memberUUID})
		if !decision.Allowed {
			w.send(wsResponse{Type: "error", Topic: clean, Message: decision.Reason})
			continue
		}
		unsub := subscriber.Subscribe(clean, func(ev fwwsbus.Event) {
			if w.tenantUUID != "" && ev.TenantUUID != "" && w.tenantUUID != ev.TenantUUID {
				return
			}
			if w.memberUUID != "" && ev.MemberUUID != "" && w.memberUUID != ev.MemberUUID {
				return
			}
			if w.lease != nil {
				if err := w.lease.AuthorizeTopic(w.ctx, clean); err != nil {
					w.recordIdentityError(err)
					return
				}
			}
			w.send(wsResponse{Type: "event", Topic: ev.Topic, Payload: ev.Payload})
		})
		if w.ctx != nil && w.ctx.Err() != nil {
			unsub()
			return accepted
		}
		w.subs[clean] = unsub
		accepted = append(accepted, clean)
	}
	return accepted
}

func (w *wsConn) recordIdentityError(err error) {
	fields := logger.Fields{"code": fwiamerrors.CodeOf(err), "status": fwiamerrors.StatusCode(err), "tenant_uuid": w.tenantUUID}
	var upstream *fwiamdelegated.IdentityRequestError
	if errors.As(err, &upstream) {
		fields["trace_id"] = upstream.TraceID
		fields["request_id"] = upstream.RequestID
		fields["source_trace_id"] = upstream.SourceTraceID
		fields["source_request_id"] = upstream.SourceRequestID
	}
	logger.WarnWith(logger.HTTPMiddleware(), w.ctx, "ws identity revalidation rejected", fields)
}

func (w *wsConn) unsubscribe(topics []string) {
	for _, topic := range topics {
		clean := strings.TrimSpace(topic)
		if clean == "" {
			continue
		}
		if unsub, exists := w.subs[clean]; exists {
			unsub()
			delete(w.subs, clean)
		}
	}
}

func (w *wsConn) close() {
	for _, unsub := range w.subs {
		if unsub != nil {
			unsub()
		}
	}
	w.subs = nil
	_ = w.conn.Close()
}

func (w *wsConn) send(msg wsResponse) {
	w.sendMu.Lock()
	defer w.sendMu.Unlock()
	if w.ctx != nil && w.ctx.Err() != nil {
		return
	}
	if w.lease != nil && !time.Now().Before(w.lease.ExpiresAt) {
		w.lease.Close(frameworkrealtime.WSIdentityExpired)
		return
	}
	if w.sendHook != nil {
		w.sendHook(msg)
		return
	}
	_ = w.conn.WriteJSON(msg)
}
