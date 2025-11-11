package handlers

import (
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// SandboxValidationHandler orchestrates sandbox validation runs.
type SandboxValidationHandler struct{}

// NewSandboxValidationHandler builds the handler.
func NewSandboxValidationHandler() *SandboxValidationHandler {
	return &SandboxValidationHandler{}
}

// Deploy triggers sandbox validation job.
func (h *SandboxValidationHandler) Deploy(ctx bootstrap.Context) {
	var payload struct {
		HostSessionID string   `json:"hostSessionId"`
		DatasetID     string   `json:"datasetId"`
		TestPlanID    string   `json:"testPlanId"`
		Flags         []string `json:"flags"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SANDBOX_PAYLOAD", "invalid payload", nil)
		return
	}
	if payload.HostSessionID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_SANDBOX_PAYLOAD", "hostSessionId is required", nil)
		return
	}
	response := map[string]any{
		"validationId": newID("sandbox"),
		"status":       "running",
		"startedAt":    time.Now().UTC(),
	}
	router.RespondSuccess(ctx, http.StatusAccepted, response, "sandbox validation started")
}
