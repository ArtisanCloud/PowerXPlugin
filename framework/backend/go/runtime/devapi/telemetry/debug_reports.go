package telemetry

import (
	"log/slog"
	"time"
)

// DebugReport captures diagnostics metadata for dev sessions.
type DebugReport struct {
	SessionID   string                 `json:"sessionId"`
	PluginID    string                 `json:"pluginId"`
	Tenant      string                 `json:"tenant"`
	Findings    []string               `json:"findings"`
	Metrics     map[string]float64     `json:"metrics"`
	Attachments map[string]interface{} `json:"attachments"`
	GeneratedAt time.Time              `json:"generatedAt"`
}

// Recorder persists debug reports (mock implementation).
type Recorder struct {
	logger *slog.Logger
}

// NewRecorder creates telemetry recorder.
func NewRecorder(logger *slog.Logger) *Recorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Recorder{logger: logger}
}

// Record logs telemetry for future wiring (placeholder).
func (r *Recorder) Record(report DebugReport) {
	r.logger.Info("debug report generated",
		slog.String("session", report.SessionID),
		slog.Int("findings", len(report.Findings)))
}
