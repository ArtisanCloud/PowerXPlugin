package hostcontract

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/hosthttp"
	"github.com/google/uuid"
)

const MaxInvocationResponseBytes = 4 << 20
const DefaultStreamTimeout = 5 * time.Minute

// StreamInput has no endpoint, method, identity headers, or transport context.
// Capability bindings and authorization remain authoritative in Core.
type StreamInput struct {
	CapabilityID      string
	PreferredProtocol string
	// TenantUUID carries caller tenant context and is checked for canonical UUID
	// format before credentials are acquired. It is never sent as an HTTP identity
	// override; Core derives the tenant from the authenticated service credential.
	TenantUUID  string
	Environment string
	Body        any
	RequestID   string
	TraceID     string
}

type StreamEvent struct {
	Event     string
	ID        string
	Data      string
	TraceID   string
	RequestID string
}

type StreamTransport struct {
	BaseURL       string
	APIPrefix     string
	Client        *http.Client
	Authorization func(context.Context) (string, error)
	Timeout       time.Duration
}

func streamFailure(status int, code, trace, request string) *InvocationFailure {
	return &InvocationFailure{StatusCode: status, ErrorCode: code, ReasonCode: code, TraceID: trace, RequestID: request}
}

// InvocationEndpoint applies the configured API prefix exactly once.
func InvocationEndpoint(base, prefix, route string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.TrimSpace(prefix) == "" {
		prefix = hosthttp.DefaultAPIPrefix
	}
	prefix = "/" + strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix != "/" && strings.HasSuffix(base, prefix) {
		base = strings.TrimSuffix(base, prefix)
	}
	api, err := hosthttp.APIBaseURL(base, prefix)
	if err != nil {
		return "", err
	}
	return api + route, nil
}

func InvokeStream(ctx context.Context, transport StreamTransport, input StreamInput, emit func(StreamEvent) error) error {
	if emit == nil || input.CapabilityID == "" || strings.TrimSpace(input.CapabilityID) != input.CapabilityID || strings.ContainsAny(input.CapabilityID, "\r\n\x00") {
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	switch input.Environment {
	case "", "dev", "staging", "prod":
	default:
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	if input.RequestID != safeDiagnostic(input.RequestID) || input.TraceID != safeDiagnostic(input.TraceID) {
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	requestID := input.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}
	if input.TenantUUID != "" {
		tenant, err := uuid.Parse(input.TenantUUID)
		if err != nil || tenant.String() != input.TenantUUID {
			return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", requestID)
		}
	}
	raw, err := json.Marshal(input.Body)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	var body struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           struct {
			Body  json.RawMessage `json:"body"`
			Query *struct {
				Environment string `json:"env"`
			} `json:"query,omitempty"`
		} `json:"payload"`
	}
	body.CapabilityID = input.CapabilityID
	body.PreferredProtocol = input.PreferredProtocol
	if body.PreferredProtocol == "" {
		body.PreferredProtocol = "rest"
	}
	if body.PreferredProtocol != "rest" && body.PreferredProtocol != "core_internal" {
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	body.Payload.Body = raw
	if input.Environment != "" {
		body.Payload.Query = &struct {
			Environment string `json:"env"`
		}{input.Environment}
	}
	raw, err = json.Marshal(body)
	if err != nil {
		return streamFailure(400, "CAPABILITY_STREAM_INVALID_ARGUMENT", "", "")
	}
	if transport.Client == nil || transport.Authorization == nil {
		return streamFailure(503, "CAPABILITY_STREAM_UNAVAILABLE", "", "")
	}
	endpoint, err := InvocationEndpoint(transport.BaseURL, transport.APIPrefix, "/tenant/invocations/stream")
	if err != nil {
		return streamFailure(400, "CAPABILITY_STREAM_CONFIG_INVALID", "", "")
	}
	timeout := transport.Timeout
	if timeout <= 0 {
		timeout = DefaultStreamTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	authorization, err := transport.Authorization(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return streamFailure(503, "CAPABILITY_STREAM_CREDENTIAL_UNAVAILABLE", "", "")
	}
	scheme, credential, ok := strings.Cut(authorization, " ")
	if !ok || (scheme != "Bearer" && scheme != "ApiKey") || strings.TrimSpace(credential) == "" || strings.ContainsAny(authorization, "\r\n\x00") {
		return streamFailure(401, "CAPABILITY_STREAM_CREDENTIAL_INVALID", "", "")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return streamFailure(400, "CAPABILITY_STREAM_CONFIG_INVALID", "", "")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", authorization)
	request.Header.Set("X-Request-ID", requestID)
	if input.TraceID != "" {
		request.Header.Set("X-Trace-ID", input.TraceID)
	}
	client := *transport.Client
	client.Timeout = timeout
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if ctx.Err() != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return ctx.Err()
	}
	if err != nil {
		return streamFailure(503, "CAPABILITY_STREAM_UNAVAILABLE", "", requestID)
	}
	defer response.Body.Close()
	stop := context.AfterFunc(ctx, func() { _ = response.Body.Close() })
	defer stop()
	if response.StatusCode != http.StatusOK {
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, MaxInvocationResponseBytes+1))
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if readErr != nil || len(raw) > MaxInvocationResponseBytes {
			return streamFailure(502, "CAPABILITY_STREAM_RESPONSE_INVALID", safeDiagnostic(response.Header.Get("X-Trace-ID")), safeDiagnostic(response.Header.Get("X-Request-ID")))
		}
		return DecodeInvocationFailure(response.StatusCode, response.Header, raw, "CAPABILITY_STREAM_UPSTREAM_FAILED")
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	trace := safeDiagnostic(response.Header.Get("X-Trace-ID"))
	if err != nil || mediaType != "text/event-stream" {
		return streamFailure(502, "CAPABILITY_STREAM_RESPONSE_INVALID", trace, safeDiagnostic(response.Header.Get("X-Request-ID")))
	}
	return readInvocationSSE(ctx, response.Body, trace, func(event StreamEvent) error {
		event.RequestID = safeDiagnostic(response.Header.Get("X-Request-ID"))
		return emit(event)
	})
}

func readInvocationSSE(ctx context.Context, reader io.Reader, trace string, emit func(StreamEvent) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), MaxInvocationResponseBytes)
	var data strings.Builder
	kind, id := "", ""
	hasData := false
	emitted := false
	flush := func() error {
		if !hasData {
			kind = ""
			return nil
		}
		eventKind := kind
		if eventKind == "" {
			eventKind = "message"
		}
		err := emit(StreamEvent{Event: eventKind, ID: id, Data: data.String(), TraceID: trace})
		emitted = true
		data.Reset()
		hasData = false
		kind = ""
		return err
	}
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
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
		case "id":
			if !strings.ContainsRune(value, '\x00') {
				id = value
			}
		case "data":
			if hasData {
				data.WriteByte('\n')
			}
			data.WriteString(value)
			hasData = true
			if data.Len() > MaxInvocationResponseBytes {
				return streamFailure(502, "CAPABILITY_STREAM_EVENT_TOO_LARGE", trace, "")
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := scanner.Err(); err != nil {
		return streamFailure(502, "CAPABILITY_STREAM_INTERRUPTED", trace, "")
	}
	if err := flush(); err != nil {
		return err
	}
	if !emitted {
		return streamFailure(502, "CAPABILITY_STREAM_RESPONSE_INVALID", trace, "")
	}
	return nil
}
