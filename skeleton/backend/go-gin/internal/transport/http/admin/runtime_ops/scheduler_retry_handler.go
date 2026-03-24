package runtime_ops

import (
	"net/http"

	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// SchedulerRetryHandler handles retry/pause/resume scaffolding endpoints.
type SchedulerRetryHandler struct {
	retryService  *runtimeops.SchedulerRetryService
	ticketService *runtimeops.SchedulerTicketService
}

// NewSchedulerRetryHandler constructs scheduler retry handler.
func NewSchedulerRetryHandler(deps *app.Deps, svc *runtimeops.Service) *SchedulerRetryHandler {
	_ = deps
	if svc == nil {
		svc = runtimeops.NewService()
	}
	return &SchedulerRetryHandler{
		retryService:  svc.SchedulerRetry,
		ticketService: svc.SchedulerTicket,
	}
}

// Retry is a phase-1 scaffold endpoint.
func (h *SchedulerRetryHandler) Retry(c *gin.Context) {
	_ = h
	c.JSON(http.StatusNotImplemented, gin.H{"ok": false, "message": "scheduler retry endpoint scaffolded"})
}

// Pause is a phase-1 scaffold endpoint.
func (h *SchedulerRetryHandler) Pause(c *gin.Context) {
	_ = h
	c.JSON(http.StatusNotImplemented, gin.H{"ok": false, "message": "scheduler pause endpoint scaffolded"})
}

// Resume is a phase-1 scaffold endpoint.
func (h *SchedulerRetryHandler) Resume(c *gin.Context) {
	_ = h
	c.JSON(http.StatusNotImplemented, gin.H{"ok": false, "message": "scheduler resume endpoint scaffolded"})
}
