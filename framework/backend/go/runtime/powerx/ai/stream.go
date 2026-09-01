package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
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
}

func (c *Client) LLMStream(ctx context.Context, input LLMStreamInput, onEvent func(LLMStreamEvent) error) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return errors.New("powerx ai client is not configured")
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
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/ai/llm/stream", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{StatusCode: resp.StatusCode}
	}
	return decodeLLMSSE(resp.Body, onEvent)
}

func (c *Client) LLMSessionStream(ctx context.Context, sessionID string, onEvent func(LLMStreamEvent) error) error {
	if c == nil || c.http == nil || c.tokens == nil {
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
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &HTTPError{StatusCode: resp.StatusCode}
	}
	return decodeLLMSSE(resp.Body, onEvent)
}

func decodeLLMSSE(r io.Reader, onEvent func(LLMStreamEvent) error) error {
	scanner := bufio.NewScanner(r)
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
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}
