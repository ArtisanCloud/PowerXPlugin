package wsbus

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/google/uuid"
)

const RealtimeContract = "realtime.service.v1"
const RealtimePublishCapability = "com.corex.runtime.realtime.service_publish"
const RealtimeStreamCapability = "com.corex.runtime.realtime.stream"

var ErrRealtimeInvalid = errors.New("REALTIME_INVALID_RESPONSE")
var ErrRealtimeUnavailable = errors.New("REALTIME_UNAVAILABLE")
var ErrRealtimeCursorExpired = errors.New("REALTIME_CURSOR_EXPIRED")
var ErrRealtimeForbidden = errors.New("REALTIME_GRANT_REQUIRED")
var streamCursor = regexp.MustCompile(`^(0|[1-9][0-9]{0,19})-(0|[1-9][0-9]{0,19})$`)
var streamTopic = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,512}$`)

type RealtimeEvent struct {
	EventID       string          `json:"event_id"`
	TenantUUID    string          `json:"tenant_uuid"`
	Topic         string          `json:"topic"`
	MemberUUID    string          `json:"member_uuid,omitempty"`
	TraceID       string          `json:"trace_id"`
	Payload       json.RawMessage `json:"-"`
	PayloadBase64 string          `json:"payload_base64,omitempty"`
}
type RealtimeFrame struct {
	ContractVersion string         `json:"contract_version"`
	Namespace       string         `json:"namespace"`
	TenantKey       string         `json:"tenant_key,omitempty"`
	Cursor          string         `json:"cursor"`
	Event           *RealtimeEvent `json:"event,omitempty"`
}
type RealtimeService interface {
	PublishEvent(context.Context, string, string, RealtimeEvent) error
	Cursor(context.Context, string, string) (string, error)
	SubscribeEvents(context.Context, string, string, string, func(RealtimeFrame) error) error
}
type RealtimeInvoker interface {
	Invoke(context.Context, gateway.InvokeRequest) (*gateway.Response, error)
	InvokeStream(context.Context, gateway.StreamInput, func(gateway.StreamEvent) error) error
}
type realtimeService struct{ invoker RealtimeInvoker }

func NewRealtimeService(invoker RealtimeInvoker) (RealtimeService, error) {
	if invoker == nil {
		return nil, ErrRealtimeUnavailable
	}
	return &realtimeService{invoker}, nil
}
func realtimeBody(tenant, namespace, operation string) map[string]any {
	return map[string]any{"contract_version": RealtimeContract, "tenant_key": tenant, "namespace": namespace, "operation": operation}
}
func validateRealtimeScope(tenant, namespace string) error {
	id, err := uuid.Parse(tenant)
	if err != nil || id.String() != tenant || namespace == "" || len(namespace) > 128 {
		return ErrRealtimeInvalid
	}
	return nil
}
func (s *realtimeService) invoke(ctx context.Context, tenant, namespace, operation, capability string, body map[string]any) (string, error) {
	if err := validateRealtimeScope(tenant, namespace); err != nil {
		return "", err
	}
	response, err := s.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: capability, TenantUUID: tenant, PreferredProtocol: "core_internal", Payload: map[string]any{"body": body}})
	if err != nil {
		return "", err
	}
	if response == nil || response.Data["status"] != "completed" || response.Data["protocol_used"] != "core_internal" || response.Data["fallback_used"] != false {
		return "", ErrRealtimeInvalid
	}
	raw, err := json.Marshal(response.Data["payload"])
	if err != nil {
		return "", ErrRealtimeInvalid
	}
	var out struct {
		Contract  string `json:"contract_version"`
		Operation string `json:"operation"`
		Tenant    string `json:"tenant_key"`
		Namespace string `json:"namespace"`
		Cursor    string `json:"cursor"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Contract != RealtimeContract || out.Operation != operation || out.Tenant != tenant || out.Namespace != namespace || !streamCursor.MatchString(out.Cursor) {
		return "", ErrRealtimeInvalid
	}
	return out.Cursor, nil
}
func (s *realtimeService) PublishEvent(ctx context.Context, tenant, namespace string, event RealtimeEvent) error {
	if event.TenantUUID != tenant || !streamTopic.MatchString(event.Topic) || !json.Valid(event.Payload) || len(event.Payload) > 64<<10 {
		return ErrRealtimeInvalid
	}
	if id, err := uuid.Parse(event.EventID); err != nil || id.String() != event.EventID {
		return ErrRealtimeInvalid
	}
	body := realtimeBody(tenant, namespace, "publish")
	wire := event
	wire.PayloadBase64 = base64.StdEncoding.EncodeToString(event.Payload)
	wire.Payload = nil
	body["event"] = wire
	_, err := s.invoke(ctx, tenant, namespace, "publish", RealtimePublishCapability, body)
	return err
}
func (s *realtimeService) Cursor(ctx context.Context, tenant, namespace string) (string, error) {
	return s.invoke(ctx, tenant, namespace, "cursor", RealtimeStreamCapability, realtimeBody(tenant, namespace, "cursor"))
}
func (s *realtimeService) SubscribeEvents(ctx context.Context, tenant, namespace, cursor string, consume func(RealtimeFrame) error) error {
	if err := validateRealtimeScope(tenant, namespace); err != nil {
		return err
	}
	if !streamCursor.MatchString(cursor) || consume == nil {
		return ErrRealtimeInvalid
	}
	body := realtimeBody(tenant, namespace, "subscribe")
	body["cursor"] = cursor
	ready, reconnect := false, false
	expectedCursor := cursor
	err := s.invoker.InvokeStream(ctx, gateway.StreamInput{CapabilityID: RealtimeStreamCapability, PreferredProtocol: "core_internal", TenantUUID: tenant, Body: body}, func(message gateway.StreamEvent) error {
		if message.Event == "error" {
			var failure struct {
				Code string `json:"code"`
			}
			if json.Unmarshal([]byte(message.Data), &failure) != nil {
				return ErrRealtimeInvalid
			}
			if failure.Code == "REALTIME_CURSOR_EXPIRED" {
				return ErrRealtimeCursorExpired
			}
			if failure.Code == "REALTIME_GRANT_REQUIRED" {
				return ErrRealtimeForbidden
			}
			if failure.Code != "REALTIME_UNAVAILABLE" {
				return ErrRealtimeInvalid
			}
			return ErrRealtimeUnavailable
		}
		var frame RealtimeFrame
		decoder := json.NewDecoder(bytes.NewReader([]byte(message.Data)))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&frame) != nil || decoder.Decode(new(any)) != io.EOF || !streamCursor.MatchString(frame.Cursor) {
			return ErrRealtimeInvalid
		}
		if reconnect {
			return ErrRealtimeInvalid
		}
		switch message.Event {
		case "ready":
			if ready || frame.ContractVersion != RealtimeContract || frame.Namespace != namespace || frame.TenantKey != tenant || frame.Cursor != cursor || frame.Event != nil {
				return ErrRealtimeInvalid
			}
			ready = true
		case "event":
			if !ready || frame.ContractVersion != RealtimeContract || frame.Namespace != namespace || frame.Event == nil || frame.Event.TenantUUID != tenant || !streamTopic.MatchString(frame.Event.Topic) {
				return ErrRealtimeInvalid
			}
			raw, e := base64.StdEncoding.Strict().DecodeString(frame.Event.PayloadBase64)
			if e != nil || base64.StdEncoding.EncodeToString(raw) != frame.Event.PayloadBase64 || len(raw) > 64<<10 || !json.Valid(raw) {
				return ErrRealtimeInvalid
			}
			frame.Event.Payload = raw
			if id, e := uuid.Parse(frame.Event.EventID); e != nil || id.String() != frame.Event.EventID {
				return ErrRealtimeInvalid
			}
			if err := consume(frame); err != nil {
				return err
			}
			if compareRealtimeCursor(frame.Cursor, expectedCursor) > 0 {
				expectedCursor = frame.Cursor
			}
		case "heartbeat":
			if !ready || frame.ContractVersion != RealtimeContract || frame.Namespace != namespace || frame.TenantKey != tenant || frame.Event != nil || frame.Cursor != expectedCursor {
				return ErrRealtimeInvalid
			}
		case "reconnect":
			if !ready || frame.Event != nil || frame.Cursor != expectedCursor {
				return ErrRealtimeInvalid
			}
			reconnect = true
		default:
			return ErrRealtimeInvalid
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !ready || !reconnect {
		return ErrRealtimeUnavailable
	}
	return nil
}
