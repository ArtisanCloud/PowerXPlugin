package handlers

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// InstallLocalHandler handles local `.pxp` uploads for tenants.
type InstallLocalHandler struct{}

func (InstallLocalHandler) Install(ctx bootstrap.Context) {
	var payload struct {
		TenantUuid string `json:"tenantId"`
		PluginID   string `json:"pluginId"`
		Version    string `json:"version"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse install payload", nil)
		return
	}
	router.RespondSuccess(ctx, http.StatusAccepted, map[string]any{
		"tenant":  payload.TenantUuid,
		"plugin":  payload.PluginID,
		"version": payload.Version,
		"status":  "installing",
	}, "local installation queued")
}
