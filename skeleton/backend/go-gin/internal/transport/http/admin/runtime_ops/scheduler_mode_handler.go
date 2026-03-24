package runtime_ops

import (
	"net/http"

	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// SchedulerModeHandler handles runtime mode validation endpoints.
type SchedulerModeHandler struct {
	service *runtimeops.SchedulerModeService
}

// NewSchedulerModeHandler constructs scheduler mode handler.
func NewSchedulerModeHandler(deps *app.Deps) *SchedulerModeHandler {
	_ = deps
	return &SchedulerModeHandler{service: runtimeops.NewSchedulerModeService()}
}

// Validate validates runtime mode/provider consistency.
func (h *SchedulerModeHandler) Validate(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "scheduler mode service unavailable"})
		return
	}

	var req runtimeops.ModeValidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	result := h.service.Validate(req)
	if !result.Valid {
		c.JSON(http.StatusConflict, result)
		return
	}
	c.JSON(http.StatusOK, result)
}
