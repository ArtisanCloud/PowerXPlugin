package media

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type ImageDerivativeEvent struct {
	Type     string
	Snapshot *ImageDerivatives
	Reason   string
}
type ImageDerivativeSubscriptionOptions struct {
	MaxReconnects  int
	ReconnectDelay time.Duration
}

// StreamImageDerivatives reads one Core lease. Every connection starts with a
// current snapshot; this contract has no cursor and promises no event replay.
func (c *HostClient) StreamImageDerivatives(ctx context.Context, assetUUID string, emit func(ImageDerivativeEvent) error) error {
	if !validMediaUUID(assetUUID) || emit == nil {
		return derivativeError(400, "MEDIA_DERIVATIVES_REQUEST_INVALID")
	}
	if c == nil || c.http == nil {
		return derivativeError(503, "MEDIA_CLIENT_UNAVAILABLE")
	}
	lease, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(lease, http.MethodGet, c.baseURL+"/tenant/media/assets/"+assetUUID+"/derivatives/events", nil)
	if err != nil {
		return err
	}
	if err := c.authorize(lease, req); err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	streamClient := *c.http
	streamClient.Timeout = 0
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return DecodeHostHTTPError(resp.StatusCode, resp.Header, raw)
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return derivativeError(502, "MEDIA_DERIVATIVES_EVENT_INVALID")
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var event string
	var lines []string
	stop := false
	flush := func() error {
		defer func() { event = ""; lines = nil }()
		if len(lines) == 0 {
			return nil
		}
		raw := []byte(strings.Join(lines, "\n"))
		switch event {
		case "status":
			state, err := decodeDerivativeSnapshot(raw, assetUUID)
			if err != nil {
				return err
			}
			return emit(ImageDerivativeEvent{Type: event, Snapshot: state})
		case "reconnect", "invalidate":
			var info struct {
				AssetUUID string `json:"asset_uuid"`
				Reason    string `json:"reason"`
			}
			if json.Unmarshal(raw, &info) != nil || info.Reason == "" || (event == "invalidate" && info.AssetUUID != assetUUID) {
				return derivativeError(502, "MEDIA_DERIVATIVES_EVENT_INVALID")
			}
			stop = true
			if err := emit(ImageDerivativeEvent{Type: event, Reason: info.Reason}); err != nil {
				return err
			}
			if event == "invalidate" {
				return derivativeError(403, "MEDIA_DERIVATIVES_INVALIDATED")
			}
			return nil
		default:
			return derivativeError(502, "MEDIA_DERIVATIVES_EVENT_INVALID")
		}
	}
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
			if stop {
				return nil
			}
		case strings.HasPrefix(line, ":"):
			continue
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(line[6:])
		case strings.HasPrefix(line, "data:"):
			lines = append(lines, strings.TrimPrefix(line[5:], " "))
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(lines) > 0 {
		return io.ErrUnexpectedEOF
	}
	return io.EOF
}

// Subscribe reconnects the selected service channel only. Renewal gets the
// latest snapshot from SSE itself; no idle GETs, ensure calls or polling occur.
func (c *HostClient) SubscribeImageDerivatives(ctx context.Context, assetUUID string, options ImageDerivativeSubscriptionOptions, emit func(ImageDerivativeEvent) error) error {
	if emit == nil {
		return derivativeError(400, "MEDIA_DERIVATIVES_REQUEST_INVALID")
	}
	max := options.MaxReconnects
	if max <= 0 {
		max = 3
	}
	delay := options.ReconnectDelay
	if delay <= 0 {
		delay = 250 * time.Millisecond
	}
	failures := 0
	for {
		started := time.Now()
		callbackFailed := false
		err := c.StreamImageDerivatives(ctx, assetUUID, func(event ImageDerivativeEvent) error {
			e := emit(event)
			if e != nil {
				callbackFailed = true
			}
			return e
		})
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if callbackFailed {
			return err
		}
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			var upstream *HostHTTPError
			var network net.Error
			if errors.As(err, &upstream) {
				if upstream.StatusCode != 429 && upstream.StatusCode != 503 && upstream.StatusCode != 504 {
					return err
				}
			} else if !errors.As(err, &network) {
				return err
			}
		}
		if time.Since(started) >= 30*time.Second {
			failures = 0
		}
		failures++
		if failures > max {
			if err != nil {
				return err
			}
			return derivativeError(503, "MEDIA_DERIVATIVES_RECONNECT_EXHAUSTED")
		}
		wait := delay * time.Duration(1<<min(failures-1, 4))
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
