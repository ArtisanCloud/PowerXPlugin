package integration

import (
	"errors"
	"net/http"

	domain "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	idrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/integration"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	"github.com/gin-gonic/gin"
)

// Dispatch 处理统一 Envelope 投递。
func (h *Handler) Dispatch(c *gin.Context) {
	if h.dispatch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dispatch service unavailable"})
		return
	}

	var req DispatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid envelope payload"})
		return
	}

	envelope, err := req.ToDomain()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	outcome, err := h.dispatch.Dispatch(
		c.Request.Context(),
		"HTTP",
		c.FullPath(),
		c.Request.Method,
		envelope,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEnvelope):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, integrationService.ErrGrantMatrixDenied):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, idrepo.ErrIdempotencyUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency backend unavailable"})
		default:
			if h.logger != nil {
				pxlog.WarnWith(h.logger, c.Request.Context(), "integration dispatch failed", map[string]interface{}{
					"module":     "integration",
					"biz_scene":  "integration_dispatch_http",
					"biz_domain": "integration",
					"path":       c.FullPath(),
					"error":      err.Error(),
					"component":  "integration_http",
				})
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         outcome.Status,
		"trace_id":       outcome.TraceID,
		"correlation_id": outcome.CorrelationID,
		"latency_ms":     outcome.Latency.Milliseconds(),
		"replay":         outcome.Replay,
	})
}
