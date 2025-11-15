package handlers

import (
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/devapi/telemetry"
)

// DebugReportHandler aggregates sandbox / host diagnostics.
type DebugReportHandler struct {
	recorder *telemetry.Recorder
}

// NewDebugReportHandler builds handler.
func NewDebugReportHandler(recorder *telemetry.Recorder) *DebugReportHandler {
	if recorder == nil {
		recorder = telemetry.NewRecorder(nil)
	}
	return &DebugReportHandler{recorder: recorder}
}

// Report accepts diagnostics report payload.
func (h *DebugReportHandler) Report(ctx bootstrap.Context) {
	var payload struct {
		SessionID string                 `json:"sessionId"`
		PluginID  string                 `json:"pluginId"`
		Tenant    string                 `json:"tenant"`
		Findings  []string               `json:"findings"`
		Metrics   map[string]float64     `json:"metrics"`
		Data      map[string]interface{} `json:"attachments"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REPORT", "invalid payload", nil)
		return
	}
	report := telemetry.DebugReport{
		SessionID:   payload.SessionID,
		PluginID:    payload.PluginID,
		Tenant:      payload.Tenant,
		Findings:    payload.Findings,
		Metrics:     payload.Metrics,
		Attachments: payload.Data,
		GeneratedAt: time.Now().UTC(),
	}
	h.recorder.Record(report)
	router.RespondSuccess(ctx, http.StatusAccepted, map[string]any{
		"reportId":  newID("debug-report"),
		"generated": report.GeneratedAt,
	}, "debug report received")
}
