package services

import (
	"log/slog"
	"time"
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
	if duration > t.threshold {
		t.logger.Warn("offline publish SLA exceeded", slog.String("publishId", publishID), slog.Duration("duration", duration))
	} else {
		t.logger.Info("offline publish review", slog.Duration("duration", duration))
	}
}
