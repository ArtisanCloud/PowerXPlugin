package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"strconv"
	"strings"
	"time"
)

// SessionEvent is a durable invocation-state frame. EventID is acknowledged
// only after the callback succeeds and is reused as Last-Event-ID on reconnect.
type SessionEvent struct {
	EventID    string
	Type       string
	Invocation *ServiceInvocation
	Status     string
	ReasonCode string
}
type sessionStreamState struct {
	cursor   int
	final    bool
	terminal error
}

func (c *Client) StreamSessionEvents(ctx context.Context, id, invocation string, fn func(SessionEvent) error) error {
	p, e := invocationPath(id, invocation)
	if e != nil {
		return e
	}
	if fn == nil {
		return sessionInvalid()
	}
	state, attempts := sessionStreamState{}, 0
	for {
		cursor := ""
		if state.cursor > 0 {
			cursor = strconv.Itoa(state.cursor)
		}
		resp, e := c.sessionRequestWithCursor(ctx, "GET", p+"/events", "", nil, cursor)
		if e != nil {
			return e
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return transportError(resp)
		}
		media, _, e := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if e != nil || media != "text/event-stream" || resp.StatusCode != 200 {
			resp.Body.Close()
			return sessionDependency()
		}
		e = consumeSessionEventsState(resp.Body, id, invocation, &state, fn)
		resp.Body.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(e, errSessionStreamEnd) {
			return nil
		}
		if e == nil || !errors.Is(e, errSessionStreamInterrupted) {
			return e
		}
		attempts++
		if !c.cfg.ReconnectPolicy.Enabled || attempts > c.cfg.ReconnectPolicy.MaxAttempts {
			return e
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.cfg.ReconnectPolicy.Backoff):
		}
	}
}

var errSessionStreamEnd = errors.New("AGENT_SESSION_STREAM_END")
var errSessionStreamInterrupted = errors.New("AGENT_SESSION_STREAM_INTERRUPTED")

// consumeSessionEvents remains a testable one-shot decoder.
func consumeSessionEvents(r io.Reader, session, invocation string, fn func(SessionEvent) error) error {
	return consumeSessionEventsState(r, session, invocation, &sessionStreamState{}, fn)
}
func consumeSessionEventsState(r io.Reader, session, invocation string, state *sessionStreamState, fn func(SessionEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var kind, eventID string
	var data strings.Builder
	flush := func() error {
		if data.Len() == 0 {
			kind = ""
			eventID = ""
			return nil
		}
		id, e := strconv.Atoi(eventID)
		if e != nil || id < 1 || id > 3 || id <= state.cursor {
			return sessionDependency()
		}
		raw := []byte(data.String())
		ev := SessionEvent{EventID: eventID, Type: kind}
		switch kind {
		case "state", "final":
			var v ServiceInvocation
			if json.Unmarshal(raw, &v) != nil || !v.valid() || v.SessionUUID != session || v.InvocationUUID != invocation {
				return sessionDependency()
			}
			if kind == "final" {
				if v.Status != "succeeded" || state.final || state.terminal != nil {
					return sessionDependency()
				}
				state.final = true
			}
			ev.Invocation = &v
			ev.Status = v.Status
		case "error":
			var v struct {
				Reason string `json:"reason_code"`
			}
			if json.Unmarshal(raw, &v) != nil || strings.TrimSpace(v.Reason) == "" || state.final || state.terminal != nil {
				return sessionDependency()
			}
			ev.ReasonCode = v.Reason
			state.terminal = sessionError(0, v.Reason)
		case "end":
			var v struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(raw, &v) != nil {
				return sessionDependency()
			}
			if (v.Status == "succeeded" && !state.final) || (v.Status != "succeeded" && v.Status != "failed" && v.Status != "cancelled") || ((v.Status == "failed" || v.Status == "cancelled") && state.terminal == nil) {
				return sessionDependency()
			}
			ev.Status = v.Status
		default:
			return sessionDependency()
		}
		if e := fn(ev); e != nil {
			return e
		}
		state.cursor = id
		data.Reset()
		kind = ""
		eventID = ""
		if ev.Type == "end" {
			if state.terminal != nil {
				return state.terminal
			}
			return errSessionStreamEnd
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if e := flush(); e != nil {
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
		case "id":
			eventID = value
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
	return errSessionStreamInterrupted
}
