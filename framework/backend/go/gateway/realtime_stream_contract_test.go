package gateway_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/stretchr/testify/require"
)

type realtimeContractInvoker struct {
	*gateway.Client
	stream invokeStreamFunc
}

func (i realtimeContractInvoker) InvokeStream(ctx context.Context, input gateway.StreamInput, emit func(gateway.StreamEvent) error) error {
	return i.stream(ctx, input, emit)
}

// This fixture tests SDK transport and recovery, not Core credential/grant checks.
func TestRealtimeHTTPReconnectKeepsCursorAndStopsOnTerminalFailure(t *testing.T) {
	const tenant = "00000000-0000-4000-8000-000000000001"
	const namespace = "fixture.realtime"
	for _, kind := range []string{"gateway", "capability"} {
		for _, scenario := range []string{"cancel", "REALTIME_GRANT_REQUIRED", "REALTIME_CURSOR_EXPIRED"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				var cursors, streams atomic.Int32
				closed := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assertNoStreamTenantOverride(t, r)
					if r.Header.Get("Authorization") != "ApiKey fixture-credential" || r.Header.Get("X-Request-ID") == "" {
						t.Error("REALTIME_CREDENTIAL_OR_REQUEST_ID_LOST")
					}
					var input struct {
						Capability string `json:"capability_id"`
						Protocol   string `json:"preferred_protocol"`
						Payload    struct {
							Body map[string]any `json:"body"`
						} `json:"payload"`
					}
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
						return
					}
					body := input.Payload.Body
					if input.Capability != wsbus.RealtimeStreamCapability || input.Protocol != "core_internal" || body["tenant_key"] != tenant || body["namespace"] != namespace || body["contract_version"] != wsbus.RealtimeContract {
						t.Error("REALTIME_SCOPE_LOST", body)
					}
					if body["operation"] == "cursor" {
						cursors.Add(1)
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "payload": map[string]any{"contract_version": wsbus.RealtimeContract, "operation": "cursor", "tenant_key": tenant, "namespace": namespace, "cursor": "0-0"}}})
						return
					}
					n := streams.Add(1)
					if n > 2 {
						t.Error("REALTIME_TERMINAL_RETRIED")
						w.WriteHeader(http.StatusForbidden)
						return
					}
					cursor := "0-0"
					if n == 2 {
						cursor = "1-0"
						defer close(closed)
					}
					if r.URL.Path != "/api/v1/tenant/invocations/stream" || body["operation"] != "subscribe" || body["cursor"] != cursor {
						t.Error("REALTIME_RESUME_CURSOR_LOST", body)
					}
					w.Header().Set("Content-Type", "text/event-stream")
					send := func(kind string, value any) {
						raw, err := json.Marshal(value)
						if err != nil {
							t.Error(err)
							return
						}
						_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, raw)
					}
					frame := wsbus.RealtimeFrame{ContractVersion: wsbus.RealtimeContract, TenantKey: tenant, Namespace: namespace, Cursor: cursor}
					send("ready", frame)
					event := wsbus.RealtimeEvent{EventID: "00000000-0000-4000-8000-000000000011", TenantUUID: tenant, Topic: "fixture.topic", TraceID: "event-trace", PayloadBase64: base64.StdEncoding.EncodeToString([]byte(`{"value":1}`))}
					send("event", wsbus.RealtimeFrame{ContractVersion: wsbus.RealtimeContract, Namespace: namespace, Cursor: "1-0", Event: &event})
					if n == 1 {
						frame.Cursor = "1-0"
						send("heartbeat", frame)
						send("reconnect", map[string]string{"cursor": "1-0"})
						return
					}
					event.EventID = "00000000-0000-4000-8000-000000000012"
					send("event", wsbus.RealtimeFrame{ContractVersion: wsbus.RealtimeContract, Namespace: namespace, Cursor: "2-0", Event: &event})
					if scenario == "cancel" {
						w.(http.Flusher).Flush()
						<-r.Context().Done()
						return
					}
					send("error", map[string]string{"code": scenario})
					event.EventID = "00000000-0000-4000-8000-000000000013"
					send("event", wsbus.RealtimeFrame{ContractVersion: wsbus.RealtimeContract, Namespace: namespace, Cursor: "3-0", Event: &event})
				}))
				t.Cleanup(server.Close)
				client, err := gateway.NewClient(gateway.Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "fixture-credential"})
				require.NoError(t, err)
				t.Cleanup(func() { _ = client.Close() })
				service, err := wsbus.NewRealtimeService(realtimeContractInvoker{client, streamForTest(t, kind, "ApiKey", server.URL, "")})
				require.NoError(t, err)
				hub, err := wsbus.NewRemoteHub(ctx, service, namespace)
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, hub.Close()) })
				received, failed := make(chan wsbus.Event, 4), make(chan error, 1)
				unsubscribe, err := hub.SubscribeScoped(ctx, tenant, "fixture.topic", func(event wsbus.Event) { received <- event }, func(err error) { failed <- err })
				require.NoError(t, err)
				for _, cursor := range []string{"1-0", "2-0"} {
					select {
					case event := <-received:
						require.Equal(t, cursor, event.Cursor)
						require.Equal(t, tenant, event.TenantUUID)
						require.Equal(t, "event-trace", event.TraceID)
					case <-ctx.Done():
						t.Fatal("REALTIME_EVENT_TIMEOUT")
					}
				}
				if scenario == "cancel" {
					unsubscribe()
				} else {
					want := wsbus.ErrRealtimeForbidden
					if scenario == "REALTIME_CURSOR_EXPIRED" {
						want = wsbus.ErrRealtimeCursorExpired
					}
					select {
					case err := <-failed:
						require.True(t, errors.Is(err, want))
					case <-ctx.Done():
						t.Fatal("REALTIME_FAILURE_TIMEOUT")
					}
				}
				select {
				case <-closed:
				case <-ctx.Done():
					t.Fatal("REALTIME_CONNECTION_NOT_CLOSED")
				}
				require.NoError(t, hub.Close())
				require.Empty(t, received)
				require.EqualValues(t, 1, cursors.Load())
				require.EqualValues(t, 2, streams.Load())
			})
		}
	}
}
