package master

import (
	"context"
	"encoding/json"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/channel"
)

// HandleCredentialInspectionEvent is a minimal "job -> consumer" migration example:
// instead of the job writing DB/log directly, the dispatcher calls this consumer handler.
func HandleCredentialInspectionEvent(logger *pxlog.Entry) func(context.Context, event.Event) error {
	if logger == nil {
		logger = pxlog.NewEntry(pxlog.StandardLogger())
	}

	return func(ctx context.Context, ev event.Event) error {
		var payload channel.CredentialInspectionPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}

		pxlog.InfoCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":         "channel",
			"biz_scene":      "credential_inspection_consume",
			"biz_domain":     "channel",
			"component":      "jobs.channel.master.credential_inspection_consumer",
			"topic":          string(ev.Topic),
			"tenant_uuid":    ev.Meta.TenantUUID,
			"trace_id":       ev.Meta.TraceID,
			"channel_id":     payload.ChannelID,
			"status":         payload.Status,
			"credentialType": payload.CredentialType,
		}), "credential inspection event consumed")

		return nil
	}
}
