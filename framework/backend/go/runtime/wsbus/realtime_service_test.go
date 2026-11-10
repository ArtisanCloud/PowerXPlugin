package wsbus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

type realtimeInvokerFixture struct {
	frames []gateway.StreamEvent
	calls  int
	unary  gateway.InvokeRequest
	stream gateway.StreamInput
}

func (f *realtimeInvokerFixture) Invoke(_ context.Context, in gateway.InvokeRequest) (*gateway.Response, error) {
	f.calls++
	f.unary = in
	body := in.Payload.(map[string]any)["body"].(map[string]any)
	return &gateway.Response{Data: map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"contract_version": RealtimeContract, "tenant_key": body["tenant_key"], "namespace": body["namespace"], "operation": body["operation"], "cursor": "0-0"}}}, nil
}
func (f *realtimeInvokerFixture) InvokeStream(_ context.Context, in gateway.StreamInput, emit func(gateway.StreamEvent) error) error {
	f.stream = in
	for _, frame := range f.frames {
		if err := emit(frame); err != nil {
			return err
		}
	}
	return nil
}
func TestRealtimeServiceStrictFramesAndIntegerPayload(t *testing.T) {
	const tenant = "00000000-0000-4000-8000-000000000001"
	const namespace = "fixture.realtime"
	ready := gateway.StreamEvent{Event: "ready", Data: `{"contract_version":"realtime.service.v1","tenant_key":"` + tenant + `","namespace":"fixture.realtime","cursor":"0-0"}`}
	raw := []byte(`{"id":9007199254740993}`)
	data, _ := json.Marshal(RealtimeFrame{ContractVersion: RealtimeContract, Namespace: namespace, Cursor: "1-0", Event: &RealtimeEvent{EventID: "00000000-0000-4000-8000-000000000002", TenantUUID: tenant, Topic: "fixture.event", PayloadBase64: base64.StdEncoding.EncodeToString(raw)}})
	event := gateway.StreamEvent{Event: "event", Data: string(data)}
	end := gateway.StreamEvent{Event: "reconnect", Data: `{"cursor":"1-0"}`}
	for _, tc := range []struct {
		name      string
		frames    []gateway.StreamEvent
		want      error
		delivered int
	}{
		{"valid", []gateway.StreamEvent{ready, event, end}, nil, 1},
		{"missing_ready", []gateway.StreamEvent{event}, ErrRealtimeInvalid, 0},
		{"duplicate_ready", []gateway.StreamEvent{ready, ready}, ErrRealtimeInvalid, 0},
		{"trailing_json", []gateway.StreamEvent{{Event: "ready", Data: ready.Data + `{}`}}, ErrRealtimeInvalid, 0},
		{"truncated", []gateway.StreamEvent{ready, event}, ErrRealtimeUnavailable, 1},
		{"expired", []gateway.StreamEvent{ready, {Event: "error", Data: `{"code":"REALTIME_CURSOR_EXPIRED"}`}}, ErrRealtimeCursorExpired, 0},
		{"revoked", []gateway.StreamEvent{ready, {Event: "error", Data: `{"code":"REALTIME_GRANT_REQUIRED"}`}}, ErrRealtimeForbidden, 0},
		{"unknown", []gateway.StreamEvent{ready, {Event: "unknown", Data: `{"cursor":"0-0"}`}}, ErrRealtimeInvalid, 0},
		{"foreign_scope", []gateway.StreamEvent{{Event: "ready", Data: `{"contract_version":"realtime.service.v1","tenant_key":"00000000-0000-4000-8000-000000000002","namespace":"fixture.realtime","cursor":"0-0"}`}}, ErrRealtimeInvalid, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invoker := &realtimeInvokerFixture{frames: tc.frames}
			service, err := NewRealtimeService(invoker)
			require.NoError(t, err)
			delivered := 0
			err = service.SubscribeEvents(context.Background(), tenant, namespace, "0-0", func(frame RealtimeFrame) error {
				delivered++
				require.Equal(t, raw, []byte(frame.Event.Payload))
				return nil
			})
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.want)
			}
			require.Equal(t, tc.delivered, delivered)
			require.Equal(t, "core_internal", invoker.stream.PreferredProtocol)
			require.Equal(t, tenant, invoker.stream.TenantUUID)
		})
	}
	invoker := &realtimeInvokerFixture{}
	service, err := NewRealtimeService(invoker)
	require.NoError(t, err)
	require.NoError(t, service.PublishEvent(context.Background(), tenant, namespace, RealtimeEvent{EventID: "00000000-0000-4000-8000-000000000002", TenantUUID: tenant, Topic: "fixture.event", Payload: raw}))
	require.Equal(t, RealtimePublishCapability, invoker.unary.CapabilityID)
	require.Equal(t, "core_internal", invoker.unary.PreferredProtocol)
	require.Equal(t, tenant, invoker.unary.TenantUUID)
	wire := invoker.unary.Payload.(map[string]any)["body"].(map[string]any)["event"].(RealtimeEvent)
	require.Equal(t, base64.StdEncoding.EncodeToString(raw), wire.PayloadBase64)
	require.Empty(t, wire.Payload)
	require.ErrorIs(t, service.PublishEvent(context.Background(), tenant, namespace, RealtimeEvent{TenantUUID: tenant, Topic: "fixture.event", Payload: raw}), ErrRealtimeInvalid)
	require.Equal(t, 1, invoker.calls)
}
