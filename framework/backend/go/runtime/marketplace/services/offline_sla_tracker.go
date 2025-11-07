package services

import (
	"log/slog"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/observability"
)

// OfflineSLATracker enforces 1 business day SLA for offline uploads.
type OfflineSLATracker struct {
	logger    *slog.Logger
	threshold time.Duration
}

func NewOfflineSLATracker(threshold time.Duration, logger *slog.Logger) *OfflineSLATracker {
	if logger == nil {
		logger = slog.Default()
	}
	if threshold == 0 {
		threshold = 24 * time.Hour
	}
	return &OfflineSLATracker{logger: logger, threshold: threshold}
}

func (t *OfflineSLATracker) Record(duration time.Duration, publishID string) {
	durationMinutes := duration.Minutes()
	plugin := "unknown"
	status := "completed"

	if duration > t.threshold {
		status = "sla_exceeded"
		t.logger.Warn("offline publish SLA exceeded",
			slog.String("publishId", publishID),
			slog.Duration("duration", duration),
		)
	} else {
		status = "completed"
		t.logger.Info("offline publish review", slog.Duration("duration", duration))
	}

	// Record Prometheus metric
	observability.RecordOfflineApprovalDuration(durationMinutes, plugin, status)
}
