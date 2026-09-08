package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"strings"
)

// SessionEvent contains durable execution state, not token deltas. Error events
// are delivered to the callback and also returned as an error after end.
type SessionEvent struct {
	Type       string
	Invocation *ServiceInvocation
	Status     string
	ReasonCode string
}

func (c *Client) StreamSessionEvents(ctx context.Context, id, invocation string, fn func(SessionEvent) error) error {
	p, e := invocationPath(id, invocation)
	if e != nil {
		return e
	}
	if fn == nil {
		return sessionInvalid()
	}
	resp, e := c.sessionRequest(ctx, "GET", p+"/events", "", nil)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return transportError(resp)
	}
	media, _, e := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if e != nil || media != "text/event-stream" || resp.StatusCode != 200 {
		return sessionDependency()
	}
	e = consumeSessionEvents(resp.Body, id, invocation, fn)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return e
}

var errSessionStreamEnd = errors.New("AGENT_SESSION_STREAM_END")

func consumeSessionEvents(r io.Reader, session, invocation string, fn func(SessionEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var kind string
	var data strings.Builder
	var streamError error
	var final bool
	flush := func() error {
		if data.Len() == 0 {
			kind = ""
			return nil
		}
		raw := []byte(data.String())
		ev := SessionEvent{Type: kind}
		switch kind {
		case "state", "final":
			var v ServiceInvocation
			if json.Unmarshal(raw, &v) != nil || !v.valid() || v.SessionUUID != session || v.InvocationUUID != invocation {
				return sessionDependency()
			}
			if kind == "final" {
				if v.Status != "succeeded" || final || streamError != nil {
					return sessionDependency()
				}
				final = true
			}
			ev.Invocation = &v
			ev.Status = v.Status
		case "error":
			var v struct {
				Reason string `json:"reason_code"`
			}
			if json.Unmarshal(raw, &v) != nil || strings.TrimSpace(v.Reason) == "" || final {
				return sessionDependency()
			}
			ev.ReasonCode = v.Reason
			streamError = sessionError(0, v.Reason)
		case "end":
			var v struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(raw, &v) != nil {
				return sessionDependency()
			}
			if (v.Status == "succeeded" && !final) || (v.Status != "succeeded" && v.Status != "failed" && v.Status != "cancelled") || ((v.Status == "failed" || v.Status == "cancelled") && streamError == nil) {
				return sessionDependency()
			}
			ev.Status = v.Status
		default:
			return sessionDependency()
		}
		if e := fn(ev); e != nil {
			return e
		}
		data.Reset()
		kind = ""
		if ev.Type == "end" {
			return errSessionStreamEnd
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if e := flush(); e != nil {
				if e == errSessionStreamEnd {
					return streamError
				}
				return e
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, ok := strings.Cut(line, ":")
		if !ok {
			value = ""
		}
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			kind = value
		case "data":
			if data.Len()+len(value) > 1<<20 {
				return sessionDependency()
			}
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(value)
		}
	}
	if scanner.Err() != nil {
		return sessionDependency()
	}
	// EOF without an explicit end is an interrupted subscription, not success.
	return sessionError(502, "AGENT_SESSION_STREAM_INTERRUPTED")
}
