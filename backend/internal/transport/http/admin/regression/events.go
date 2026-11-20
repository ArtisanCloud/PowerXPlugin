package regression

import (
	"fmt"
	Realtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) Events(c *gin.Context) {
	a, ok := h.actor(c)
	if !ok {
		return
	}
	if !Regression.ValidUUID(a.MemberUUID) {
		h.fail(c, Regression.NewError(Regression.InvalidRequest))
		return
	}
	id := c.Param("run_uuid")
	if _, err := h.runner.Get(c.Request.Context(), a, id); err != nil {
		h.fail(c, err)
		return
	}
	notify, stop := h.runner.Subscribe(a.TenantUUID, id)
	defer stop()
	cursor := c.Query("after_event_uuid")
	events, err := h.runner.Store.Events(c.Request.Context(), a.TenantUUID, id, cursor)
	if err != nil {
		h.fail(c, err)
		return
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.fail(c, Regression.NewError(Regression.InvalidRequest))
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()
	for {
		for _, event := range events {
			if err := Regression.ValidateMeta(event.Meta, a.TenantUUID, a.PluginID); err != nil {
				return
			}
			if _, err = fmt.Fprintf(c.Writer, "id: %s\n", event.UUID); err != nil {
				return
			}
			env := Realtime.NewSSEEnvelope("regression."+id, "powerx.regression.run.updated.v1", event, Realtime.Scope{PluginID: a.PluginID, TenantUUID: a.TenantUUID, MemberUUID: a.MemberUUID, TraceID: event.Meta.TraceID, RequestID: event.Meta.RequestID})
			if err = Realtime.WriteSSEEnvelope(c.Writer, env); err != nil {
				return
			}
			flusher.Flush()
			cursor = event.UUID
		}
		d, err := h.runner.Get(c.Request.Context(), a, id)
		if err != nil {
			return
		}
		if Regression.Terminal(d.Run.Phase) && d.Run.CleanupState != "running" {
			remaining, err := h.runner.Store.Events(c.Request.Context(), a.TenantUUID, id, cursor)
			if err != nil || len(remaining) == 0 {
				return
			}
			events = remaining
			continue
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-notify:
			events, err = h.runner.Store.Events(c.Request.Context(), a.TenantUUID, id, cursor)
			if err != nil {
				return
			}
		}
	}
}
