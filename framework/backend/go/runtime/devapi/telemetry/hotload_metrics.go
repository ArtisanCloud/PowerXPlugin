package telemetry

import (
	"log/slog"
	"time"
)

// HotloadEvent captures dev hotload telemetry signals to external sinks.
type HotloadEvent struct {
	SessionID string
	Tenant    string
	Latency   time.Duration
	Status    string
}

var sink = slog.Default()

// RecordReload logs reload telemetry that can later be wired to Kafka/Redis.
func RecordReload(event HotloadEvent) {
	sink.Info("dev hotload reload", slog.String("session", event.SessionID), slog.Duration("latency", event.Latency), slog.String("tenant", event.Tenant), slog.String("status", event.Status))
}
