package mcp

import (
	"net/http"
	"time"

	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	frameworksse "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ssebus"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// Handler serves MCP stream endpoints (SSE/WebSocket).
type Handler struct {
	broker      *stream.Broker
	db          *gorm.DB
	descriptors []frameworkrealtime.Descriptor
}

// NewHandler builds a stream handler.
func NewHandler(deps *app.Deps) *Handler {
	h := &Handler{broker: stream.DefaultBroker()}
	if deps != nil {
		h.db = deps.DB
		h.descriptors = deps.RealtimeDescriptors
	}
	return h
}

// RegisterRoutes binds SSE/WS handlers to the engine.
func RegisterRoutes(router *gin.RouterGroup, deps *app.Deps) {
	if router == nil {
		return
	}
	h := NewHandler(deps)
	router.GET("/mcp/sse", h.ServeSSE)
	router.GET("/mcp/ws", h.ServeWebsocket)
}

// ServeSSE streams session events over Server-Sent Events.
func (h *Handler) ServeSSE(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}
	if !h.authorizeSession(c, sessionID) {
		return
	}
	ch, cancel := h.broker.Subscribe(sessionID)
	defer cancel()

	events := make(chan frameworksse.Event, 16)
	go func() {
		defer close(events)
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				events <- frameworksse.Event{
					Channel:   sessionID,
					EventType: evt.Type,
					Payload:   evt,
					Timestamp: evt.Timestamp,
				}
			}
		}
	}()

	frameworksse.ServeStream(c.Request.Context(), c.Writer, events, frameworksse.StreamOptions{
		HeartbeatEvery: 25 * time.Second,
		HeartbeatEvent: "ping",
		Heartbeat: func(now time.Time) any {
			return frameworksse.Event{
				Channel:   sessionID,
				EventType: "ping",
				Payload:   stream.Event{SessionID: sessionID, Type: "ping", Timestamp: now.UTC()},
				Timestamp: now.UTC(),
			}
		},
	})
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// ServeWebsocket streams events over WebSocket.
func (h *Handler) ServeWebsocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}
	if !h.authorizeSession(c, sessionID) {
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch, cancel := h.broker.Subscribe(sessionID)
	defer cancel()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(evt); err != nil {
				return
			}
		case <-time.After(25 * time.Second):
			heartbeat := stream.Event{SessionID: sessionID, Type: "ping", Timestamp: time.Now().UTC()}
			if err := conn.WriteJSON(heartbeat); err != nil {
				return
			}
		}
	}
}

// authorizeSession prevents a caller from subscribing to another tenant's MCP
// session. A missing or cross-tenant session has the same 404 response so the
// transport cannot be used as a session-existence oracle.
func (h *Handler) authorizeSession(c *gin.Context, sessionID string) bool {
	if h == nil || h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mcp session store is unavailable"})
		return false
	}
	tenantUUID, err := authx.RequireTenantUUID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context is required"})
		return false
	}
	var session model.MCPSession
	if err := h.db.WithContext(c.Request.Context()).
		Where("id = ? AND tenant_uuid = ?", sessionID, tenantUUID).
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "mcp session not found"})
			return false
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mcp session lookup failed"})
		return false
	}
	decision := frameworkrealtime.Decide(
		h.descriptors,
		frameworkrealtime.ActionSubscribe,
		"_channel.mcp.session",
		frameworkrealtime.ProtocolSSE,
		"session.ready",
		frameworkrealtime.Scope{TenantUUID: tenantUUID},
	)
	if !decision.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": decision.Reason})
		return false
	}
	return true
}
