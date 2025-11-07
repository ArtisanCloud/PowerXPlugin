package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// DevPluginHandler exposes register/reload/delete endpoints consumed by px-plugin dev.
type DevPluginHandler struct {
	logger *slog.Logger
}

func NewDevPluginHandler(logger *slog.Logger) *DevPluginHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &DevPluginHandler{logger: logger}
}

func (h *DevPluginHandler) Register(ctx bootstrap.Context) {
	var payload RegisterRequest
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse register payload", nil)
		return
	}
	response := RegisterResponse{
		SessionID:    newID("sess"),
		ReloadToken:  newID("reload"),
		AdminPreview: "/admin/dev-hotload/preview",
	}
	h.logger.Info("dev session registered", slog.String("plugin", payload.Manifest.ID), slog.String("session", response.SessionID))
	router.RespondSuccess(ctx, http.StatusCreated, response, "session registered")
}

func (h *DevPluginHandler) Reload(ctx bootstrap.Context) {
	var payload ReloadRequest
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse reload payload", nil)
		return
	}
	h.logger.Info("dev session reload", slog.String("session", payload.SessionID), slog.Int("files", len(payload.ChangedFiles)))
	router.RespondSuccess(ctx, http.StatusOK, router.Envelope{Message: "reload accepted"}, "reload accepted")
}

func (h *DevPluginHandler) Delete(ctx bootstrap.Context) {
	sessionID := ctx.Param("sessionId")
	if sessionID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SESSION", "sessionId is required", nil)
		return
	}
	h.logger.Info("dev session deleted", slog.String("session", sessionID))
	router.RespondSuccess(ctx, http.StatusNoContent, nil, "session removed")
}

// payload definitions -------------------------------------------------------

type RegisterRequest struct {
	Manifest Manifest `json:"manifest"`
	Tenant   string   `json:"tenant,omitempty"`
}

type Manifest struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type RegisterResponse struct {
	SessionID    string `json:"sessionId"`
	ReloadToken  string `json:"reloadToken"`
	AdminPreview string `json:"adminPreviewUrl"`
}

type ReloadRequest struct {
	SessionID    string   `json:"sessionId"`
	ReloadToken  string   `json:"reloadToken"`
	ChangedFiles []string `json:"changedFiles"`
}

// helpers ------------------------------------------------------------------

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
