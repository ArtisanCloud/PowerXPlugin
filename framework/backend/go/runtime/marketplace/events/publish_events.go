package events

import "log/slog"

// EventEmitter provides minimal abstraction for broadcasting publish events.
type EventEmitter struct {
	logger *slog.Logger
}

func NewEventEmitter(logger *slog.Logger) *EventEmitter {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventEmitter{logger: logger}
}

func (e *EventEmitter) EmitApproved(publishID, versionID string, tenants []string) {
	e.logger.Info("plugin.publish.approved", slog.String("publishId", publishID), slog.String("versionId", versionID), slog.Any("tenants", tenants))
}
