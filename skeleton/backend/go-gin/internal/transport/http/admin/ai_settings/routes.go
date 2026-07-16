package ai_settings

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil {
		return
	}
	h := &Handler{mode: deps.IAMMode}
	group := admin.Group("/ai-settings")
	group.GET("/mode", h.Mode)
	group.GET("/summary", h.Summary)
}

type Handler struct {
	mode iamservice.IAMMode
}

func (h *Handler) Mode(c *gin.Context) {
	mode := strings.TrimSpace(h.mode.String())
	if mode == "" {
		mode = string(iamservice.IAMModeLocal)
	}
	contracts.ResponseSuccess(c, gin.H{"mode": mode})
}

func (h *Handler) Summary(c *gin.Context) {
	contracts.ResponseServiceUnavailable(c, "AI settings provider is not configured for this plugin", gin.H{
		"mode": strings.TrimSpace(h.mode.String()),
		"code": "AI_SETTINGS_PROVIDER_NOT_CONFIGURED",
	})
}
