package handlers

import (
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/marketplace/services"
)

// PublishHandler handles CLI publish submission payloads for Marketplace reviewers.
type PublishHandler struct {
	Scanner services.Scanner
}

func NewPublishHandler(scanner services.Scanner) *PublishHandler {
	return &PublishHandler{Scanner: scanner}
}

// Submit publishes a version to the review queue.
func (h *PublishHandler) Submit(ctx bootstrap.Context) {
	var payload services.PublishPayload
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "unable to parse payload", nil)
		return
	}
	result := h.Scanner.Scan(payload)
	response := services.ReviewRecord{
		PublishID:    payload.PublishID,
		VersionID:    payload.VersionID,
		Channel:      payload.Channel,
		SubmittedAt:  time.Now().UTC(),
		Status:       "pending",
		ScanFindings: result,
	}
	router.RespondSuccess(ctx, http.StatusAccepted, response, "publish request enqueued")
}
