package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func DecodeSSE(r io.Reader) ([]AgentStreamEvent, error) {
	scanner := bufio.NewScanner(r)
	var events []AgentStreamEvent
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
		events = append(events, ev)
		eventType = ""
		data.Reset()
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return nil, err
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
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return events, nil
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

func (c *Client) StreamSSE(ctx context.Context, query url.Values) ([]AgentStreamEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, withQuery(c.url(c.cfg.SSEPath), query), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if err := c.authorize(ctx, req); err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, &Error{Code: ErrCodeTransport, Message: resp.Status}
	}
	return DecodeSSE(resp.Body)
}
