package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type LLMStreamEvent struct {
	Type         string `json:"type"`
	TraceID      string `json:"trace_id,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Index        int    `json:"index,omitempty"`
	Text         string `json:"text,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
	Code         string `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
}

type LLMStreamInput struct {
	LLMInvokeInput
	IncludeUsage bool
	Environment  string
}

func (c *Client) LLMStream(ctx context.Context, input LLMStreamInput, onEvent func(LLMStreamEvent) error) error {
	if c == nil || c.http == nil || c.authorization == nil {
		return errors.New("AI_STREAM_UNAVAILABLE")
	}
	if strings.TrimSpace(input.ModelKey) == "" || len(input.Inputs) == 0 || onEvent == nil {
		return errors.New("AI_STREAM_INVALID_ARGUMENT")
	}
	body := struct {
		ModelKey      string         `json:"model_key"`
		Inputs        []ContentItem  `json:"inputs"`
		Params        map[string]any `json:"params,omitempty"`
		StreamOptions struct {
			IncludeUsage bool `json:"include_usage"`
		} `json:"stream_options"`
	}{ModelKey: input.ModelKey, Inputs: input.Inputs, Params: input.Params}
	body.StreamOptions.IncludeUsage = input.IncludeUsage
	complete := false
	err := hostcontract.InvokeStream(ctx, hostcontract.StreamTransport{BaseURL: c.baseURL, APIPrefix: c.apiPrefix, Client: c.http, Authorization: c.authorization.Authorization, Timeout: c.streamTimeout}, hostcontract.StreamInput{CapabilityID: "com.corex.ai.llm.stream", PreferredProtocol: "core_internal", Environment: input.Environment, Body: body}, func(event hostcontract.StreamEvent) error {
		if complete {
			return errors.New("AI_STREAM_RESPONSE_INVALID")
		}
		switch event.Event {
		case "start", "delta", "usage", "done", "error":
		default:
			return errors.New("AI_STREAM_RESPONSE_INVALID")
		}
		var payload struct {
			OK           bool   `json:"ok"`
			TraceID      string `json:"traceId"`
			Delta        string `json:"delta"`
			Index        int    `json:"index"`
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
			Code         string `json:"code"`
			Message      string `json:"message"`
		}
		if json.Unmarshal([]byte(event.Data), &payload) != nil {
			return errors.New("AI_STREAM_RESPONSE_INVALID")
		}
		trace := payload.TraceID
		if trace == "" {
			trace = event.TraceID
		}
		kind := event.Event
		if kind == "done" {
			if !payload.OK {
				return errors.New("AI_STREAM_RESPONSE_INVALID")
			}
			kind = "final"
			complete = true
		}
		if err := onEvent(LLMStreamEvent{Type: kind, TraceID: trace, Delta: payload.Delta, Index: payload.Index, Text: payload.Text, FinishReason: payload.FinishReason, Code: payload.Code, Message: payload.Message}); err != nil {
			return err
		}
		if event.Event == "error" {
			return errors.New("AI_STREAM_FAILED")
		}
		if event.Event == "done" {
			return onEvent(LLMStreamEvent{Type: "end", TraceID: trace, FinishReason: payload.FinishReason})
		}
		return nil
	})
	var failure *hostcontract.InvocationFailure
	if errors.As(err, &failure) {
		return &HTTPError{StatusCode: failure.StatusCode, ErrorCode: failure.ErrorCode, ReasonCode: failure.ReasonCode, TraceID: failure.TraceID, RequestID: failure.RequestID, Details: failure.Details.Map()}
	}
	if err != nil {
		return err
	}
	if !complete {
		return errors.New("AI_STREAM_INCOMPLETE")
	}
	return nil
}

func (c *Client) LLMSessionStream(ctx context.Context, sessionID string, onEvent func(LLMStreamEvent) error) error {
	if c == nil || c.http == nil || c.authorization == nil {
		return errors.New("powerx ai client is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("powerx ai session_id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/ai/llm/sessions/"+url.PathEscape(strings.TrimSpace(sessionID))+"/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	authorization, err := c.authorization.Authorization(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return DecodeHTTPError(resp.StatusCode, resp.Header, raw)
	}
	return decodeLLMSSE(resp.Body, onEvent)
}

func decodeLLMSSE(r io.Reader, onEvent func(LLMStreamEvent) error) error {
	if onEvent == nil {
		return errors.New("ai.stream_callback_required")
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var kind string
	var data strings.Builder
	flush := func() error {
		if kind == "" && data.Len() == 0 {
			return nil
		}
		var payload struct {
			TraceID      string `json:"traceId"`
			Delta        string `json:"delta"`
			Index        int    `json:"index"`
			Text         string `json:"text"`
			FinishReason string `json:"finish_reason"`
			Code         string `json:"code"`
			Message      string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data.String()), &payload); err != nil {
			return err
		}
		event := LLMStreamEvent{Type: kind, TraceID: payload.TraceID, Delta: payload.Delta, Index: payload.Index, Text: payload.Text, FinishReason: payload.FinishReason, Code: payload.Code, Message: payload.Message}
		kind = ""
		data.Reset()
		if onEvent != nil {
			return onEvent(event)
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			kind = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}
