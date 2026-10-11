package agent

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func sessionFailure(c *gin.Context, err error) {
	status, reason := 503, "AGENT_SESSION_UNAVAILABLE"
	var upstream *fwagent.Error
	if errors.As(err, &upstream) {
		reason = upstream.ReasonCode
		if reason == "" {
			reason = upstream.Code
		}
		if upstream.StatusCode >= 400 && upstream.StatusCode <= 599 {
			status = upstream.StatusCode
		}
	}
	contracts.ResponseErrorWithReason(c, status, reason, reason)
}
func invalidSession(c *gin.Context) {
	sessionFailure(c, &fwagent.Error{StatusCode: 400, ReasonCode: "AGENT_SESSION_INVALID_ARGUMENT"})
}
func bindSession(c *gin.Context, out any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10))
	decoder.DisallowUnknownFields()
	var raw json.RawMessage
	if decoder.Decode(&raw) != nil || len(raw) == 0 || raw[0] != '{' {
		invalidSession(c)
		return false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		invalidSession(c)
		return false
	}
	fields := json.NewDecoder(strings.NewReader(string(raw)))
	_, _ = fields.Token()
	seen := map[string]bool{}
	for fields.More() {
		key, err := fields.Token()
		name, ok := key.(string)
		if err != nil || !ok || seen[name] {
			invalidSession(c)
			return false
		}
		seen[name] = true
		var value json.RawMessage
		if fields.Decode(&value) != nil || string(value) == "null" {
			invalidSession(c)
			return false
		}
	}
	strict := json.NewDecoder(strings.NewReader(string(raw)))
	strict.DisallowUnknownFields()
	if strict.Decode(out) != nil {
		invalidSession(c)
		return false
	}
	return true
}

// Session consumes the mode-bound business contract only. Browser/admin
// credentials authenticate this plugin request; they are never forwarded to Core.
func (h *Handler) Session(c *gin.Context) {
	if h == nil || h.deps == nil || h.deps.AgentLifecycle == nil {
		sessionFailure(c, nil)
		return
	}
	svc, err := h.deps.AgentLifecycle.Sessions()
	if err != nil {
		sessionFailure(c, err)
		return
	}
	sid, iid := c.Param("session_uuid"), c.Param("invocation_uuid")
	for _, id := range []string{sid, iid} {
		if id != "" {
			u, e := uuid.Parse(id)
			if e != nil || u == uuid.Nil || u.String() != id {
				invalidSession(c)
				return
			}
		}
	}
	paged := c.Request.Method == "GET" && (c.FullPath() == "/api/v1/plugin/agent/sessions" || strings.HasSuffix(c.FullPath(), "/messages"))
	// Compare route suffix as prefixes can be customized by plugin configuration.
	if c.Request.Method == "GET" && sid == "" {
		paged = true
	}
	for key, values := range c.Request.URL.Query() {
		if !paged || (key != "page" && key != "page_size") || len(values) != 1 {
			invalidSession(c)
			return
		}
	}
	page, size := 1, 20
	if paged {
		var e error
		page, e = strconv.Atoi(c.DefaultQuery("page", "1"))
		if e != nil {
			invalidSession(c)
			return
		}
		size, e = strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if e != nil || page < 1 || page > 1000000 || size < 1 || size > 100 {
			invalidSession(c)
			return
		}
	}
	ctx := c.Request.Context()
	method, path := c.Request.Method, c.FullPath()
	var result any
	status := 200
	hasJSON := method == "PATCH" || (method == "POST" && (sid == "" || strings.HasSuffix(path, "/messages") || strings.HasSuffix(path, "/invocations")))
	if !hasJSON {
		raw, e := io.ReadAll(io.LimitReader(c.Request.Body, 1))
		if e != nil || len(raw) > 0 {
			invalidSession(c)
			return
		}
	}
	switch {
	case strings.HasSuffix(path, "/events"):
		started, ended := false, false
		err = svc.StreamSessionEvents(ctx, sid, iid, func(event fwagent.SessionEvent) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !started {
				c.Header("Content-Type", "text/event-stream")
				c.Header("Cache-Control", "no-cache")
				c.Header("X-Accel-Buffering", "no")
				started = true
			}
			var data any
			switch event.Type {
			case "state", "final":
				data = event.Invocation
			case "error":
				data = gin.H{"error_code": event.ReasonCode, "reason_code": event.ReasonCode}
			case "end":
				data = gin.H{"status": event.Status}
				ended = true
			default:
				return &fwagent.Error{ReasonCode: "AGENT_SESSION_UPSTREAM_DEPENDENCY", StatusCode: 502}
			}
			c.SSEvent(event.Type, data)
			c.Writer.Flush()
			return nil
		})
		if ctx.Err() != nil || ended {
			return
		}
		if !started {
			if err == nil {
				err = &fwagent.Error{ReasonCode: "AGENT_SESSION_STREAM_INTERRUPTED", StatusCode: 502}
			}
			sessionFailure(c, err)
			return
		}
		reason := "AGENT_SESSION_STREAM_INTERRUPTED"
		var upstream *fwagent.Error
		if errors.As(err, &upstream) && upstream.ReasonCode != "" {
			reason = upstream.ReasonCode
		}
		c.SSEvent("error", gin.H{"reason_code": reason, "error_code": reason})
		c.SSEvent("end", gin.H{"status": "failed"})
		c.Writer.Flush()
		return
	case iid != "":
		if method == "POST" {
			result, err = svc.CancelSessionInvocation(ctx, sid, iid)
			status = 202
		} else {
			result, err = svc.GetSessionInvocation(ctx, sid, iid)
		}
	case strings.HasSuffix(path, "/invocations"):
		var in struct {
			MessageUUID string `json:"message_uuid"`
		}
		if !bindSession(c, &in) {
			return
		}
		result, err = svc.InvokeSession(ctx, sid, in.MessageUUID, c.GetHeader("Idempotency-Key"))
		status = 202
	case strings.HasSuffix(path, "/messages"):
		if method == "GET" {
			result, err = svc.ListSessionMessages(ctx, sid, fwagent.SessionPage{Page: page, PageSize: size})
		} else {
			var in fwagent.AppendSessionMessageInput
			if !bindSession(c, &in) {
				return
			}
			result, err = svc.AppendSessionMessage(ctx, sid, c.GetHeader("Idempotency-Key"), in)
			status = 201
		}
	case strings.HasSuffix(path, "/archive"):
		result, err = svc.ArchiveSession(ctx, sid)
	case sid == "":
		if method == "GET" {
			result, err = svc.ListSessions(ctx, fwagent.SessionPage{Page: page, PageSize: size})
		} else {
			var in fwagent.CreateSessionInput
			if !bindSession(c, &in) {
				return
			}
			result, err = svc.CreateSession(ctx, in)
			status = 201
		}
	case method == "GET":
		result, err = svc.GetSession(ctx, sid)
	case method == "DELETE":
		result, err = svc.DeleteSession(ctx, sid)
	case method == "PATCH":
		var in struct {
			Title *string `json:"title"`
		}
		if !bindSession(c, &in) {
			return
		}
		if in.Title == nil {
			invalidSession(c)
			return
		}
		result, err = svc.RenameSession(ctx, sid, *in.Title)
	}
	if err != nil {
		sessionFailure(c, err)
		return
	}
	switch status {
	case 201:
		contracts.ResponseCreated(c, result)
	case 202:
		contracts.ResponseAccepted(c, result)
	default:
		contracts.ResponseSuccess(c, result)
	}
}
