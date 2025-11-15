package handlers

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// InstallURLHandler handles remote install via marketplace URL.
type InstallURLHandler struct{}

// Install processes install via remote URL.
func (InstallURLHandler) Install(ctx bootstrap.Context) {
	var payload struct {
		PluginID string `json:"pluginId"`
		Version  string `json:"version"`
		Source   string `json:"sourceUrl"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse install payload", nil)
		return
	}
	router.RespondSuccess(ctx, http.StatusAccepted, map[string]any{
		"pluginId": payload.PluginID,
		"version":  payload.Version,
		"status":   "installing",
	}, "installation started")
}
