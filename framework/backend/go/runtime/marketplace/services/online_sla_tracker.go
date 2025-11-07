package services

import (
	"log/slog"
	"time"
)

// SLATracker calculates average review time and emits warnings if exceeding thresholds.
type SLATracker struct {
	logger    *slog.Logger
	threshold time.Duration
}

func NewOnlineSLATracker(threshold time.Duration, logger *slog.Logger) *SLATracker {
	if logger == nil {
		logger = slog.Default()
	}
	if threshold == 0 {
		threshold = 4 * time.Hour
	}
	return &SLATracker{logger: logger, threshold: threshold}
}

func (t *SLATracker) Record(reviewDuration time.Duration, publishID string) {
	if reviewDuration > t.threshold {
		t.logger.Warn("online publish SLA exceeded", slog.String("publishId", publishID), slog.Duration("duration", reviewDuration))
	} else {
		t.logger.Info("online publish review", slog.Duration("duration", reviewDuration))
	}
}
