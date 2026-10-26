package ai_settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/gin-gonic/gin"
)

func registerMediaRoutes(group *gin.RouterGroup, h *Handler) {
	group.GET("/media/models", h.MediaModels)
	group.GET("/media/providers", h.MediaProviders)
	group.POST("/media/:modality/invoke", h.MediaInvoke)
	group.GET("/media/tasks/:task_uuid", h.MediaTask)
	group.POST("/media/tasks/:task_uuid/cancel", h.MediaCancel)
	group.GET("/media/tasks/:task_uuid/events", h.MediaEvents)
}
func (h *Handler) mediaService() (fwai.GenerativeService, error) {
	if h == nil || h.deps == nil || h.deps.AIInvocation == nil {
		return nil, &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_CLIENT_UNAVAILABLE"}
	}
	return h.deps.AIInvocation.Generative()
}
func mediaFailure(c *gin.Context, err error) {
	upstream := &powerxai.HTTPError{StatusCode: 503, ReasonCode: "AI_DEPENDENCY_UNAVAILABLE"}
	var typed *powerxai.HTTPError
	var local *hostapi.HTTPError
	if errors.As(err, &typed) {
		upstream = typed
	} else if errors.As(err, &local) {
		upstream = &powerxai.HTTPError{StatusCode: local.StatusCode, ReasonCode: local.ReasonCode}
	} else {
		switch err.Error() {
		case "AI_REQUEST_INVALID", "AI_TASK_UUID_INVALID", "AI_EVENT_CURSOR_INVALID", "AI_IDEMPOTENCY_KEY_REQUIRED":
			upstream.StatusCode = 400
			upstream.ReasonCode = err.Error()
		case "AI_MODEL_CAPABILITY_UNSUPPORTED":
			upstream.StatusCode = 422
			upstream.ReasonCode = err.Error()
		case "AI_SETTINGS_CATALOG_RESPONSE_INVALID":
			upstream.StatusCode = http.StatusBadGateway
			upstream.ReasonCode = err.Error()
		}
	}
	if upstream.TraceID != "" {
		c.Header("X-Trace-ID", upstream.TraceID)
	}
	if upstream.RequestID != "" {
		c.Header("X-Request-ID", upstream.RequestID)
	}
	code := upstream.ErrorCode
	if code == "" {
		code = upstream.ReasonCode
	}
	c.JSON(upstream.StatusCode, gin.H{"error_code": code, "reason_code": upstream.ReasonCode, "message": upstream.ReasonCode, "details": upstream.Details, "trace_id": upstream.TraceID, "request_id": upstream.RequestID})
}
func (h *Handler) MediaModels(c *gin.Context) {
	if !mediaQuery(c, "provider", "app", "capability") {
		return
	}
	service, err := h.mediaService()
	if err != nil {
		mediaFailure(c, err)
		return
	}
	out, err := service.ListMediaModels(c.Request.Context(), powerxai.MediaCatalogFilter{Provider: c.Query("provider"), App: c.Query("app"), Capability: c.Query("capability")})
	if err != nil {
		mediaFailure(c, err)
		return
	}
	contracts.ResponseSuccess(c, out)
}
func (h *Handler) MediaProviders(c *gin.Context) {
	if !mediaQuery(c, "capability") {
		return
	}
	service, err := h.mediaService()
	if err != nil {
		mediaFailure(c, err)
		return
	}
	out, err := service.ListMediaProviders(c.Request.Context(), c.Query("capability"))
	if err != nil {
		mediaFailure(c, err)
		return
	}
	contracts.ResponseSuccess(c, out)
}
func (h *Handler) MediaInvoke(c *gin.Context) {
	if c.Request.URL.RawQuery != "" || c.GetHeader("X-PowerX-Env") != "" {
		mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
		return
	}
	var input powerxai.MediaInvokeInput
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 16<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(input.ModelKey) == "" || len(input.Inputs) == 0 {
		mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
		return
	}
	service, err := h.mediaService()
	if err != nil {
		mediaFailure(c, err)
		return
	}
	var out any
	switch c.Param("modality") {
	case "vlm":
		out, err = service.VLMInvoke(c.Request.Context(), input)
	case "image":
		out, err = service.ImageInvoke(c.Request.Context(), input)
	case "video":
		out, err = service.VideoInvoke(c.Request.Context(), input)
	case "tts":
		out, err = service.TTSInvoke(c.Request.Context(), input)
	default:
		err = errors.New("AI_MODEL_CAPABILITY_UNSUPPORTED")
	}
	if err != nil {
		mediaFailure(c, err)
		return
	}
	if c.Param("modality") == "vlm" {
		contracts.ResponseSuccess(c, out)
	} else {
		c.JSON(http.StatusAccepted, gin.H{"data": out})
	}
}
func (h *Handler) MediaTask(c *gin.Context)   { h.mediaTaskOperation(c, false) }
func (h *Handler) MediaCancel(c *gin.Context) { h.mediaTaskOperation(c, true) }
func (h *Handler) mediaTaskOperation(c *gin.Context, cancel bool) {
	if !mediaQuery(c) {
		return
	}
	if c.Request.URL.RawQuery != "" || !powerxai.ValidMediaTaskUUID(c.Param("task_uuid")) || (cancel && c.Request.ContentLength != 0) {
		mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
		return
	}
	service, err := h.mediaService()
	if err != nil {
		mediaFailure(c, err)
		return
	}
	var out *powerxai.MediaTask
	if cancel {
		out, err = service.CancelMediaTask(c.Request.Context(), c.Param("task_uuid"))
	} else {
		out, err = service.GetMediaTask(c.Request.Context(), c.Param("task_uuid"))
	}
	if err != nil {
		mediaFailure(c, err)
		return
	}
	contracts.ResponseSuccess(c, out)
}
func (h *Handler) MediaEvents(c *gin.Context) {
	if !mediaQuery(c) {
		return
	}
	if c.Request.URL.RawQuery != "" {
		mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
		return
	}
	service, err := h.mediaService()
	if err != nil {
		mediaFailure(c, err)
		return
	}
	started := false
	controller := http.NewResponseController(c.Writer)
	err = service.SubscribeMediaTask(c.Request.Context(), c.Param("task_uuid"), powerxai.MediaTaskSubscriptionOptions{LastEventID: c.GetHeader("Last-Event-ID")}, func(event powerxai.MediaTaskEvent) error {
		if event.Type == "reconnect" {
			return nil
		} // Core credential renewal is owned by the server SDK.
		if !started {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-store")
			c.Header("X-Accel-Buffering", "no")
			started = true
		}
		_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if event.Type == "keepalive" {
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return err
			}
		} else {
			raw, err := json.Marshal(event.Task)
			if err != nil {
				return err
			}
			if _, err = fmt.Fprintf(c.Writer, "id: %s\nevent: status\ndata: %s\n\n", event.ID, raw); err != nil {
				return err
			}
		}
		return controller.Flush()
	})
	if err == nil || c.Request.Context().Err() != nil {
		return
	}
	if !started {
		mediaFailure(c, err)
		return
	}
	var typed *powerxai.HTTPError
	event := "reconnect"
	reason := "AI_EVENT_CONNECTION_LOST"
	if errors.As(err, &typed) && (typed.StatusCode == 401 || typed.StatusCode == 403) {
		event = "error"
		reason = typed.ReasonCode
	}
	data := gin.H{"reason_code": reason}
	if typed != nil {
		data["reason_code"] = typed.ReasonCode
		data["error_code"] = typed.ErrorCode
		data["http_status"] = typed.StatusCode
		data["details"] = typed.Details
		data["trace_id"] = typed.TraceID
		data["request_id"] = typed.RequestID
	}
	raw, _ := json.Marshal(data)
	_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw)
	_ = controller.Flush()
}

func mediaQuery(c *gin.Context, allowed ...string) bool {
	if c.GetHeader("X-PowerX-Env") != "" {
		mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
		return false
	}
	for key, values := range c.Request.URL.Query() {
		valid := false
		for _, name := range allowed {
			if name == key {
				valid = true
				break
			}
		}
		if !valid || len(values) != 1 {
			mediaFailure(c, errors.New("AI_REQUEST_INVALID"))
			return false
		}
	}
	return true
}
