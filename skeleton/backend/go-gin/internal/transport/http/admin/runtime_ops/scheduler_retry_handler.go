package runtime_ops

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// SchedulerRetryHandler handles retry/pause/resume endpoints.
type SchedulerRetryHandler struct {
	retryService  *runtimeops.SchedulerRetryService
	ticketService *runtimeops.SchedulerTicketService
	cfg           *config.Config
}

// NewSchedulerRetryHandler constructs scheduler retry handler.
func NewSchedulerRetryHandler(deps *app.Deps, svc *runtimeops.Service) *SchedulerRetryHandler {
	if svc == nil {
		svc = runtimeops.NewService()
	}
	h := &SchedulerRetryHandler{
		retryService:  svc.SchedulerRetry,
		ticketService: svc.SchedulerTicket,
	}
	if deps != nil {
		h.cfg = deps.Config
	}
	return h
}

type retryDispatchRequest struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type pauseDispatchRequest struct {
	PausedJobID string `json:"paused_job_id"`
}

type resumeTicketRequest struct {
	OperatorRole string `json:"operator_role"`
	OperatorID   string `json:"operator_id"`
	Reason       string `json:"reason"`
}

// Retry performs bounded retry and returns 202/409.
func (h *SchedulerRetryHandler) Retry(c *gin.Context) {
	if h == nil || h.retryService == nil {
		contracts.ResponseServiceUnavailable(c, "scheduler retry service unavailable", nil)
		return
	}
	dispatchID := strings.TrimSpace(c.Param("dispatchId"))
	if dispatchID == "" {
		contracts.ResponseBadRequest(c, "dispatchId is required")
		return
	}

	var req retryDispatchRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid retry payload")
			return
		}
	}

	decision, err := h.retryService.Retry(dispatchID, h.retryMaxAttempts(), req.ErrorCode, req.ErrorMessage)
	if err != nil {
		if errors.Is(err, runtimeops.ErrDispatchIDRequired) {
			contracts.ResponseBadRequest(c, err.Error())
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}

	if decision.Exhausted {
		contracts.ResponseErrorWithDetails(c, http.StatusConflict, contracts.ErrCodeConflict, "retry exhausted, pause and ticket required", gin.H{
			"dispatch_id":     dispatchID,
			"current_attempt": decision.CurrentAttempt,
			"max_attempts":    decision.MaxAttempts,
		})
		return
	}

	c.JSON(http.StatusAccepted, contracts.MakeSuccess(gin.H{
		"dispatch_id":     dispatchID,
		"status":          "processing",
		"retry_attempt":   decision.CurrentAttempt,
		"max_attempts":    decision.MaxAttempts,
		"retry_exhausted": false,
	}, "", requestIDFromContext(c)))
}

// Pause pauses an exhausted dispatch and creates recovery ticket.
func (h *SchedulerRetryHandler) Pause(c *gin.Context) {
	if h == nil || h.retryService == nil || h.ticketService == nil {
		contracts.ResponseServiceUnavailable(c, "scheduler pause service unavailable", nil)
		return
	}
	dispatchID := strings.TrimSpace(c.Param("dispatchId"))
	if dispatchID == "" {
		contracts.ResponseBadRequest(c, "dispatchId is required")
		return
	}

	var req pauseDispatchRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid pause payload")
			return
		}
	}

	if err := h.retryService.Pause(dispatchID); err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrDispatchNotFound):
			contracts.ResponseNotFound(c, err.Error())
		case errors.Is(err, runtimeops.ErrRetryNotExhausted):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}

	ticket := h.ticketService.CreatePausedTicket(dispatchID, strings.TrimSpace(req.PausedJobID))
	contracts.ResponseCreated(c, ticket)
}

// Resume resolves ticket by ops/admin and unpauses dispatch.
func (h *SchedulerRetryHandler) Resume(c *gin.Context) {
	if h == nil || h.retryService == nil || h.ticketService == nil {
		contracts.ResponseServiceUnavailable(c, "scheduler resume service unavailable", nil)
		return
	}
	ticketID := strings.TrimSpace(c.Param("ticketId"))
	if ticketID == "" {
		contracts.ResponseBadRequest(c, "ticketId is required")
		return
	}

	var req resumeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid resume payload")
		return
	}

	operatorRole := strings.TrimSpace(req.OperatorRole)
	if operatorRole == "" {
		operatorRole = strings.TrimSpace(c.GetHeader("X-Operator-Role"))
	}
	operatorID := strings.TrimSpace(req.OperatorID)
	if operatorID == "" {
		operatorID = strings.TrimSpace(c.GetHeader("X-Operator-Id"))
	}
	if operatorID == "" {
		contracts.ResponseBadRequest(c, runtimeops.ErrOperatorRequired.Error())
		return
	}

	if strings.EqualFold(strings.TrimSpace(h.resumeRoleRequired()), "ops_admin_only") {
		if !isAllowedResumeRole(operatorRole) {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, runtimeops.ErrRoleForbidden.Error())
			return
		}
	}

	ticket, err := h.ticketService.ResumeTicket(ticketID, operatorID, operatorRole)
	if err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrTicketNotFound):
			contracts.ResponseNotFound(c, err.Error())
		case errors.Is(err, runtimeops.ErrRoleForbidden):
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, err.Error())
		case errors.Is(err, runtimeops.ErrOperatorRequired), errors.Is(err, runtimeops.ErrRoleRequired), errors.Is(err, runtimeops.ErrTicketIDRequired):
			contracts.ResponseBadRequest(c, err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}

	if err := h.retryService.Resume(ticket.DispatchID); err != nil && !errors.Is(err, runtimeops.ErrDispatchNotFound) {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, ticket)
}

func (h *SchedulerRetryHandler) retryMaxAttempts() int {
	if h == nil || h.cfg == nil || h.cfg.Operations == nil {
		return 3
	}
	v := h.cfg.Operations.Scheduler.RetryMaxAttempts
	if v <= 0 {
		return 3
	}
	return v
}

func (h *SchedulerRetryHandler) resumeRoleRequired() string {
	if h == nil || h.cfg == nil || h.cfg.Operations == nil {
		return "ops_admin_only"
	}
	v := strings.TrimSpace(h.cfg.Operations.Scheduler.ResumeRoleRequired)
	if v == "" {
		return "ops_admin_only"
	}
	return v
}

func isAllowedResumeRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "ops", "admin":
		return true
	default:
		return false
	}
}

func requestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetString("request_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.GetHeader("Request-ID")); v != "" {
		return v
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}
