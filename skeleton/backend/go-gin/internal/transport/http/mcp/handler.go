package mcp

import (
	"net/http"
	"strings"
	"time"

	frameworksse "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ssebus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/mcp/stream"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Handler serves MCP stream endpoints (SSE/WebSocket).
type Handler struct {
	broker *stream.Broker
}

// NewHandler builds a stream handler.
func NewHandler() *Handler {
	return &Handler{broker: stream.DefaultBroker()}
}

// RegisterRoutes binds SSE/WS handlers to the engine.
func RegisterRoutes(engine *gin.Engine, apiPrefix string) {
	if engine == nil {
		return
	}
	h := NewHandler()
	paths := []string{"/mcp/sse", "/mcp/ws"}
	if trimmed := strings.TrimSpace(apiPrefix); trimmed != "" && trimmed != "/" {
		base := strings.TrimRight(trimmed, "/")
		paths = append(paths, base+"/mcp/sse", base+"/mcp/ws")
	}
	for _, p := range paths {
		if strings.HasSuffix(p, "/sse") {
			engine.GET(p, h.ServeSSE)
		} else {
			engine.GET(p, h.ServeWebsocket)
		}
	}
}

// ServeSSE streams session events over Server-Sent Events.
func (h *Handler) ServeSSE(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
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
