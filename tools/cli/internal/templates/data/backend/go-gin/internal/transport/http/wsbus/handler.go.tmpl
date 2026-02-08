package wsbus

import (
	"net/http"
	"path"
	"strings"
	"sync"

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
	conn       *websocket.Conn
	sendMu     sync.Mutex
	subs       map[string]func()
	tenantUUID string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func RegisterRoutes(r *gin.Engine, deps *app.Deps, jwtCfg middleware.JWTAuthConfig, prefixes ...string) {
	if r == nil {
		return
	}
	handler := Handler(deps, jwtCfg)
	prefix := "/api/v1"
	if len(prefixes) > 0 {
		p := strings.TrimSpace(prefixes[0])
		if p != "" {
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			prefix = strings.TrimSuffix(p, "/")
		}
	}
	r.GET(path.Join(prefix, "ws"), handler)
}

func Handler(deps *app.Deps, jwtCfg middleware.JWTAuthConfig) gin.HandlerFunc {
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

		tenantUUID, ok := resolveTenant(c, jwtCfg)
		if !ok && !jwtCfg.Optional {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		ws := &wsConn{
			conn:       conn,
			subs:       make(map[string]func()),
			tenantUUID: tenantUUID,
		}
		defer ws.close()

		for {
			var req wsRequest
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			switch strings.ToLower(strings.TrimSpace(req.Type)) {
			case "subscribe":
				ws.subscribe(subscriber, req.Topics)
				ws.send(wsResponse{Type: "ack", Message: "subscribed", Payload: gin.H{"topics": req.Topics}})
			case "unsubscribe":
				ws.unsubscribe(req.Topics)
				ws.send(wsResponse{Type: "ack", Message: "unsubscribed", Payload: gin.H{"topics": req.Topics}})
			default:
				ws.send(wsResponse{Type: "error", Message: "unknown message type"})
			}
		}
	}
}

func resolveTenant(c *gin.Context, jwtCfg middleware.JWTAuthConfig) (string, bool) {
	tenant := strings.TrimSpace(c.Query("tenant_uuid"))
	authz := strings.TrimSpace(c.Query("authorization"))
	if authz == "" {
		authz = strings.TrimSpace(c.GetHeader("Authorization"))
	}
	header := func(name string) string {
		if strings.EqualFold(name, "Authorization") && authz != "" {
			return authz
		}
		return c.GetHeader(name)
	}
	tc, _, ok := middleware.ParseFromHeaders(header, jwtCfg)
	if ok && strings.TrimSpace(tc.TenantUUID) != "" {
		return strings.TrimSpace(tc.TenantUUID), true
	}
	if tenant != "" {
		return tenant, true
	}
	return "", false
}

func (w *wsConn) subscribe(subscriber wsSubscriber, topics []string) {
	for _, topic := range topics {
		clean := strings.TrimSpace(topic)
		if clean == "" {
			continue
		}
		if _, exists := w.subs[clean]; exists {
			continue
		}
		unsub := subscriber.Subscribe(clean, func(ev fwwsbus.Event) {
			if w.tenantUUID != "" && ev.TenantUUID != "" && w.tenantUUID != ev.TenantUUID {
				return
			}
			w.send(wsResponse{Type: "event", Topic: ev.Topic, Payload: ev.Payload})
		})
		w.subs[clean] = unsub
	}
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
	_ = w.conn.WriteJSON(msg)
}
