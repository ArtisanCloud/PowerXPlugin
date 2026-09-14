package host_contract

import (
	dto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	ai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/gin-gonic/gin"
)

func (h *Handler) probeAIExecution(c *gin.Context, input map[string]any, result *hostcontract.ProbeResult) error {
	s, e := h.ai.Generative()
	if e != nil {
		return e
	}
	var out any
	if result.Operation == "llm.invoke" {
		var in ai.LLMInvokeInput
		if e = decodeStorage(input, &in); e == nil {
			out, e = s.LLMInvoke(c.Request.Context(), in)
		}
	} else {
		var in ai.EmbeddingInvokeInput
		if e = decodeStorage(input, &in); e == nil {
			out, e = s.EmbeddingInvoke(c.Request.Context(), in)
		}
	}
	if e == nil {
		result.Result = map[string]any{"output": out}
	}
	return e
}

func (h *Handler) probeAgentSession(c *gin.Context, input map[string]any, result *hostcontract.ProbeResult) error {
	s, e := h.agent.Sessions()
	if e != nil {
		return e
	}
	ctx := c.Request.Context()
	fields := map[string]any{}
	for k, v := range input {
		fields[k] = v
	}
	var id string
	if result.Operation != "session.create" && result.Operation != "sessions.list" {
		id, e = inputUUID(fields, "session_uuid")
		if e != nil {
			return e
		}
		delete(fields, "session_uuid")
	}
	var out any
	switch result.Operation {
	case "session.create":
		var in dto.CreateSessionInput
		if e = decodeStorage(fields, &in); e == nil {
			out, e = s.CreateSession(ctx, in)
		}
	case "sessions.list", "session.messages.list":
		var in struct {
			Page int `json:"page"`
			Size int `json:"page_size"`
		}
		if e = decodeStorage(fields, &in); e == nil {
			p := dto.SessionPage{Page: in.Page, PageSize: in.Size}
			if result.Operation == "sessions.list" {
				out, e = s.ListSessions(ctx, p)
			} else {
				out, e = s.ListSessionMessages(ctx, id, p)
			}
		}
	case "session.get", "session.archive", "session.delete":
		if e = ensureInputKeys(fields); e == nil {
			switch result.Operation {
			case "session.get":
				out, e = s.GetSession(ctx, id)
			case "session.archive":
				out, e = s.ArchiveSession(ctx, id)
			case "session.delete":
				out, e = s.DeleteSession(ctx, id)
			}
		}
	case "session.rename":
		var in struct {
			Title string `json:"title"`
		}
		if e = decodeStorage(fields, &in); e == nil {
			out, e = s.RenameSession(ctx, id, in.Title)
		}
	case "session.message.append":
		var in struct {
			Key     string `json:"idempotency_key"`
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if e = decodeStorage(fields, &in); e == nil {
			out, e = s.AppendSessionMessage(ctx, id, in.Key, dto.AppendSessionMessageInput{Role: in.Role, Content: in.Content})
		}
	case "session.invoke":
		var in struct {
			Message string `json:"message_uuid"`
			Key     string `json:"idempotency_key"`
		}
		if e = decodeStorage(fields, &in); e == nil {
			out, e = s.InvokeSession(ctx, id, in.Message, in.Key)
		}
	case "session.invocation.get", "session.invocation.cancel":
		var in struct {
			ID string `json:"invocation_uuid"`
		}
		if e = decodeStorage(fields, &in); e == nil {
			if result.Operation == "session.invocation.get" {
				out, e = s.GetSessionInvocation(ctx, id, in.ID)
			} else {
				out, e = s.CancelSessionInvocation(ctx, id, in.ID)
			}
		}
	default:
		return &hostcontract.Error{Reason: hostcontract.ReasonUnsupportedOperation}
	}
	if e == nil {
		result.Result = map[string]any{"output": out}
	}
	return e
}
