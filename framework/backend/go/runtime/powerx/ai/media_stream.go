package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type MediaTaskEvent struct {
	Type       string
	ID         string
	Task       *MediaTask
	ReasonCode string
}
type MediaTaskSubscriptionOptions struct {
	LastEventID string
	// MaxReconnects bounds consecutive transport failures. Progress or a healthy
	// long connection resets the budget, allowing five-minute stream renewal.
	MaxReconnects  int
	ReconnectDelay time.Duration
}

func mediaCursor(taskUUID, cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 2 || parts[0] != taskUUID {
		return 0, errors.New("AI_EVENT_CURSOR_INVALID")
	}
	revision, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || revision < 1 || strconv.FormatInt(revision, 10) != parts[1] {
		return 0, errors.New("AI_EVENT_CURSOR_INVALID")
	}
	return revision, nil
}

// SubscribeMediaTask reconnects using the same trusted credential provider and
// task cursor. It never queries or submits a task to keep a connection alive.
func (c *Client) SubscribeMediaTask(ctx context.Context, taskUUID string, options MediaTaskSubscriptionOptions, emit func(MediaTaskEvent) error) error {
	if !ValidMediaTaskUUID(taskUUID) || emit == nil {
		return errors.New("AI_REQUEST_INVALID")
	}
	revision, err := mediaCursor(taskUUID, options.LastEventID)
	if err != nil {
		return err
	}
	if c == nil || c.http == nil || c.authorization == nil {
		return errors.New("AI_CLIENT_UNAVAILABLE")
	}
	max := options.MaxReconnects
	if max <= 0 {
		max = 5
	}
	delay := options.ReconnectDelay
	if delay <= 0 {
		delay = 250 * time.Millisecond
	}
	cursor := options.LastEventID
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		received := false
		connectedAt := time.Now()
		terminal := false
		callbackFailed := false
		err := c.mediaTaskStream(ctx, taskUUID, cursor, func(event MediaTaskEvent) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if event.Type == "status" {
				terminal = event.Task.Terminal()
				if event.Task.Revision <= revision {
					return nil
				}
				received = true
				revision = event.Task.Revision
				cursor = event.ID
			}
			if e := emit(event); e != nil {
				callbackFailed = true
				return e
			}
			return nil
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if callbackFailed || terminal {
			return err
		}
		var apiError *HTTPError
		if errors.As(err, &apiError) && apiError.StatusCode != 429 && apiError.StatusCode != 502 && apiError.StatusCode != 503 && apiError.StatusCode != 504 {
			return err
		}
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			// Protocol violations are permanent; network reads/writes remain retryable.
			if strings.HasPrefix(err.Error(), "AI_") {
				return err
			}
		}
		if received || time.Since(connectedAt) >= 30*time.Second {
			failures = 0
		}
		failures++
		if failures > max {
			if err != nil {
				return err
			}
			return errors.New("AI_EVENT_RECONNECT_EXHAUSTED")
		}
		wait := delay * time.Duration(1<<min(failures-1, 5))
		if wait > 5*time.Second {
			wait = 5 * time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) mediaTaskStream(ctx context.Context, taskUUID, cursor string, emit func(MediaTaskEvent) error) error {
	streamCtx, cancel := context.WithTimeout(ctx, 5*time.Minute+30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(streamCtx, http.MethodGet, c.baseURL+"/api/v1/ai/media/tasks/"+taskUUID+"/events", nil)
	if err != nil {
		return err
	}
	authorization, err := c.authorization.Authorization(ctx)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Accept", "text/event-stream")
	if cursor != "" {
		request.Header.Set("Last-Event-ID", cursor)
	}
	// Streaming is bounded by the caller context and Core's stream lease, rather
	// than the ordinary JSON client's total request timeout.
	streamClient := *c.http
	streamClient.Timeout = 0
	response, err := streamClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		if readErr != nil {
			return readErr
		}
		return DecodeHTTPError(response.StatusCode, response.Header, raw)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		return errors.New("AI_EVENT_RESPONSE_INVALID")
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	var eventType, id string
	var lines []string
	terminal := false
	reconnect := false
	flush := func() error {
		defer func() { eventType = ""; id = ""; lines = nil }()
		if len(lines) == 0 {
			return nil
		}
		raw := []byte(strings.Join(lines, "\n"))
		switch eventType {
		case "status":
			var task MediaTask
			if json.Unmarshal(raw, &task) != nil {
				return errors.New("AI_EVENT_RESPONSE_INVALID")
			}
			if err := task.Validate(); err != nil {
				return err
			}
			if task.TaskUUID != taskUUID || id != task.EventID() {
				return errors.New("AI_EVENT_CURSOR_INVALID")
			}
			terminal = task.Terminal()
			return emit(MediaTaskEvent{Type: "status", ID: id, Task: &task})
		case "reconnect", "error":
			var reason struct {
				ReasonCode string `json:"reason_code"`
			}
			if json.Unmarshal(raw, &reason) != nil || reason.ReasonCode == "" {
				return errors.New("AI_EVENT_RESPONSE_INVALID")
			}
			if eventType == "error" {
				return DecodeHTTPError(http.StatusForbidden, response.Header, raw)
			}
			reconnect = true
			return emit(MediaTaskEvent{Type: eventType, ReasonCode: reason.ReasonCode})
		default:
			return errors.New("AI_EVENT_RESPONSE_INVALID")
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
			if terminal || reconnect {
				return nil
			}
		case strings.HasPrefix(line, ":"):
			if err := emit(MediaTaskEvent{Type: "keepalive"}); err != nil {
				return err
			}
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(line[6:])
		case strings.HasPrefix(line, "id:"):
			id = strings.TrimSpace(line[3:])
		case strings.HasPrefix(line, "data:"):
			lines = append(lines, strings.TrimPrefix(line[5:], " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(lines) > 0 {
		return io.ErrUnexpectedEOF
	}
	return io.EOF
}
