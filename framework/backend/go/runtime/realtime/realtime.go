package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ssebus"
)

type Protocol string

const (
	ProtocolSSE Protocol = "sse"
	ProtocolWS  Protocol = "ws"
)

type Scope struct {
	PluginID   string
	TenantUUID string
	MemberUUID string
	TraceID    string
	RequestID  string
}

type Envelope struct {
	Protocol   Protocol  `json:"protocol"`
	Topic      string    `json:"topic,omitempty"`
	Channel    string    `json:"channel,omitempty"`
	EventType  string    `json:"event_type"`
	Payload    any       `json:"payload,omitempty"`
	PluginID   string    `json:"plugin_id,omitempty"`
	TenantUUID string    `json:"tenant_uuid,omitempty"`
	MemberUUID string    `json:"member_uuid,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type StreamThroughOptions struct {
	StatusCode int
	Header     http.Header
	Body       io.Reader
	RequestID  string
	TraceID    string
	OnError    func(error)
}

func TenantKey(prefix string, tenantUUID string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	tenantUUID = strings.TrimSpace(tenantUUID)
	if prefix == "" {
		return "", errors.New("realtime key prefix is required")
	}
	if tenantUUID == "" {
		return "", errors.New("tenant_uuid is required")
	}
	return prefix + ".tenant." + tenantUUID, nil
}

func MemberKey(prefix string, tenantUUID string, memberUUID string) (string, error) {
	base, err := TenantKey(prefix, tenantUUID)
	if err != nil {
		return "", err
	}
	memberUUID = strings.TrimSpace(memberUUID)
	if memberUUID == "" {
		return "", errors.New("member_uuid is required")
	}
	return base + ".member." + memberUUID, nil
}

func NewSSEEnvelope(channel string, eventType string, payload any, scope Scope) Envelope {
	return Envelope{
		Protocol:   ProtocolSSE,
		Channel:    strings.TrimSpace(channel),
		EventType:  firstNonEmpty(eventType, "message"),
		Payload:    payload,
		PluginID:   strings.TrimSpace(scope.PluginID),
		TenantUUID: strings.TrimSpace(scope.TenantUUID),
		MemberUUID: strings.TrimSpace(scope.MemberUUID),
		TraceID:    strings.TrimSpace(scope.TraceID),
		RequestID:  strings.TrimSpace(scope.RequestID),
		Timestamp:  time.Now().UTC(),
	}
}

func WriteSSEEnvelope(w http.ResponseWriter, env Envelope) error {
	if env.Timestamp.IsZero() {
		env.Timestamp = time.Now().UTC()
	}
	if env.Protocol == "" {
		env.Protocol = ProtocolSSE
	}
	if strings.TrimSpace(env.EventType) == "" {
		env.EventType = "message"
	}
	return ssebus.WriteEvent(w, env.EventType, env)
}

func WriteSSEError(w http.ResponseWriter, code string, message string, scope Scope) error {
	payload := map[string]any{
		"success": false,
		"error": map[string]string{
			"code":    strings.TrimSpace(code),
			"message": strings.TrimSpace(message),
		},
	}
	return WriteSSEEnvelope(w, NewSSEEnvelope("", "error", payload, scope))
}

func ProxySSEStream(ctx context.Context, w http.ResponseWriter, opts StreamThroughOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if w == nil {
		return errors.New("response writer is nil")
	}
	if opts.Body == nil {
		return errors.New("stream body is nil")
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return errors.New("streaming unsupported")
	}
	copySSEHeaders(w.Header(), opts.Header)
	if strings.TrimSpace(w.Header().Get("Content-Type")) == "" {
		w.Header().Set("Content-Type", "text/event-stream")
	}
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	if strings.TrimSpace(opts.RequestID) != "" {
		w.Header().Set("X-Request-ID", strings.TrimSpace(opts.RequestID))
	}
	if strings.TrimSpace(opts.TraceID) != "" {
		w.Header().Set("X-Trace-ID", strings.TrimSpace(opts.TraceID))
	}
	status := opts.StatusCode
	if status <= 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	flusher.Flush()

	errCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, opts.Body)
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil && opts.OnError != nil {
			opts.OnError(err)
		}
		flusher.Flush()
		return err
	}
}

func DecodeSSEData(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode sse data: %w", err)
	}
	return out, nil
}

func copySSEHeaders(dst http.Header, src http.Header) {
	if dst == nil || src == nil {
		return
	}
	for _, key := range []string{"Content-Type", "Cache-Control", "X-Request-ID", "X-Trace-ID"} {
		if value := strings.TrimSpace(src.Get(key)); value != "" {
			dst.Set(key, value)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
