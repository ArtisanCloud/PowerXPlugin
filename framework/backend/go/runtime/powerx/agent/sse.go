package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

func DecodeSSE(r io.Reader) ([]AgentStreamEvent, error) {
	var events []AgentStreamEvent
	err := consumeSSE(r, func(event AgentStreamEvent) error { events = append(events, event); return nil })
	return events, err
}

func consumeSSE(r io.Reader, onEvent func(AgentStreamEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var eventType string
	var data strings.Builder
	flush := func() error {
		if eventType == "" && data.Len() == 0 {
			return nil
		}
		ev, err := decodeEvent(eventType, data.String())
		if err != nil {
			return err
		}
		if err := onEvent(ev); err != nil {
			return err
		}
		eventType = ""
		data.Reset()
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
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}
	return nil
}

func decodeEvent(eventType, raw string) (AgentStreamEvent, error) {
	var ev AgentStreamEvent
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			return ev, &Error{Code: ErrCodeStreamDecode, Message: err.Error()}
		}
	}
	if ev.Type == "" {
		ev.Type = eventType
	}
	if !IsKnownEventType(ev.Type) {
		return ev, &Error{Code: ErrCodeStreamDecode, Message: "unknown event type: " + ev.Type}
	}
	return ev, nil
}

func (c *Client) StreamSSE(ctx context.Context, query url.Values, onEvent func(AgentStreamEvent) error) error {
	if c != nil && c.cfg.Mode == ModeDelegated {
		return sessionError(400, "AGENT_SESSION_REQUIRED")
	}
	if c == nil || c.http == nil || c.tokens == nil {
		return newError(ErrCodeConfigInvalid, "agent.client_unavailable")
	}
	if onEvent == nil {
		return newError(ErrCodeConfigInvalid, "agent.stream_callback_required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, withQuery(c.url(c.cfg.SSEPath), query), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if err := c.authorize(ctx, req); err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return transportError(resp)
	}
	mediaType, _, contentTypeErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if contentTypeErr != nil || mediaType != "text/event-stream" {
		return newError(ErrCodeStreamDecode, "agent.stream_content_type_invalid")
	}
	return consumeSSE(resp.Body, onEvent)
}
