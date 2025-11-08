package handlers

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// HostSimulatorHandler manages mock host sessions for CLI workflows.
type HostSimulatorHandler struct {
	logger   *slog.Logger
	sessions *hostSessionStore
}

type hostSession struct {
	ID        string    `json:"sessionId"`
	PluginID  string    `json:"pluginId"`
	Status    string    `json:"status"`
	Endpoint  string    `json:"endpoint"`
	Mock      bool      `json:"mock"`
	StartedAt time.Time `json:"startedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type hostSessionStore struct {
	sync.Mutex
	store map[string]hostSession
}

func newHostSessionStore() *hostSessionStore {
	return &hostSessionStore{
		store: map[string]hostSession{},
	}
}

func (s *hostSessionStore) put(session hostSession) {
	s.Lock()
	defer s.Unlock()
	s.store[session.ID] = session
}

func (s *hostSessionStore) get(id string) (hostSession, bool) {
	s.Lock()
	defer s.Unlock()
	session, ok := s.store[id]
	return session, ok
}

func (s *hostSessionStore) delete(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.store, id)
}

// NewHostSimulatorHandler creates a host simulator handler.
func NewHostSimulatorHandler(logger *slog.Logger) *HostSimulatorHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &HostSimulatorHandler{
		logger:   logger,
		sessions: newHostSessionStore(),
	}
}

// Start bootstraps a mock host session.
func (h *HostSimulatorHandler) Start(ctx bootstrap.Context) {
	var payload struct {
		PluginID       string `json:"pluginId"`
		RuntimeVersion string `json:"runtimeVersion"`
		Tenant         string `json:"tenant"`
		Mock           bool   `json:"mock"`
	}
	if err := ctx.BindJSON(&payload); err != nil || payload.PluginID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_HOST_REQUEST", "pluginId is required", nil)
		return
	}
	session := hostSession{
		ID:        newID("host"),
		PluginID:  payload.PluginID,
		Status:    "running",
		Endpoint:  "/mock/host/" + payload.PluginID,
		Mock:      payload.Mock,
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	h.sessions.put(session)
	router.RespondSuccess(ctx, http.StatusCreated, session, "host session started")
}

// Stop deletes a mock host session.
func (h *HostSimulatorHandler) Stop(ctx bootstrap.Context) {
	sessionID := ctx.Param("sessionId")
	if sessionID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SESSION", "sessionId is required", nil)
		return
	}
	h.sessions.delete(sessionID)
	router.RespondSuccess(ctx, http.StatusNoContent, nil, "host session stopped")
}

// Status returns host session metadata.
func (h *HostSimulatorHandler) Status(ctx bootstrap.Context) {
	sessionID := ctx.Param("sessionId")
	session, ok := h.sessions.get(sessionID)
	if !ok {
		router.RespondError(ctx, http.StatusNotFound, "SESSION_NOT_FOUND", "host session not found", nil)
		return
	}
	router.RespondSuccess(ctx, http.StatusOK, session, "host session status")
}

// Attach registers breakpoints/variables for debugging.
func (h *HostSimulatorHandler) Attach(ctx bootstrap.Context) {
	sessionID := ctx.Param("sessionId")
	if sessionID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SESSION", "sessionId is required", nil)
		return
	}
	var payload struct {
		Breakpoints []map[string]interface{} `json:"breakpoints"`
		Variables   map[string]string        `json:"variables"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_PAYLOAD", "unable to parse payload", nil)
		return
	}
	if session, ok := h.sessions.get(sessionID); ok {
		session.UpdatedAt = time.Now().UTC()
		h.sessions.put(session)
	}
	data := map[string]any{
		"attached":    true,
		"breakpoints": len(payload.Breakpoints),
	}
	router.RespondSuccess(ctx, http.StatusOK, data, "breakpoints attached")
}

// Logs provides recent log lines for a host session.
func (h *HostSimulatorHandler) Logs(ctx bootstrap.Context) {
	sessionID := ctx.Param("sessionId")
	if sessionID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SESSION", "sessionId is required", nil)
		return
	}
	logs := []string{
		"host session " + sessionID + " running",
		"ready to accept reload events",
	}
	router.RespondSuccess(ctx, http.StatusOK, logs, "host logs")
}
