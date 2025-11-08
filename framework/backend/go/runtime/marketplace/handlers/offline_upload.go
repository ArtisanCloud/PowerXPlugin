package handlers

import (
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/marketplace/services"
)

// OfflineUploadHandler receives .pxp artefacts and metadata for offline review.
type OfflineUploadHandler struct {
	Validator services.OfflineValidator
}

func NewOfflineUploadHandler(validator services.OfflineValidator) *OfflineUploadHandler {
	return &OfflineUploadHandler{Validator: validator}
}

func (h *OfflineUploadHandler) Upload(ctx bootstrap.Context) {
	var payload services.OfflineUploadPayload
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse offline upload payload", nil)
		return
	}
	if err := h.Validator.Validate(payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "OFFLINE_UPLOAD_INVALID", err.Error(), nil)
		return
	}
	router.RespondSuccess(ctx, http.StatusAccepted, map[string]any{
		"publishId":   payload.PublishID,
		"status":      "pending",
		"receivedAt":  time.Now().UTC(),
		"whiteListed": payload.AllowedTenants,
	}, "offline upload accepted")
}
