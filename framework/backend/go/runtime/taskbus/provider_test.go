package taskbus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/middleware"
)

func TestHostProvider_NewEmitterAndPublish(t *testing.T) {
	type publishRequest struct {
		Topic      string         `json:"topic"`
		TenantUUID string         `json:"tenant_uuid"`
		Payload    map[string]any `json:"payload"`
		TraceID    string         `json:"trace_id"`
	}

	var received publishRequest
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/admin/runtime/ws-bus/publish", r.URL.Path)
		authHeader = r.Header.Get("Authorization")

		defer r.Body.Close()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer server.Close()

	provider := NewHostProvider(HostProviderConfig{
		BaseURL: server.URL,
		TokenProvider: func(context.Context) (string, error) {
			return "sts-token-1", nil
		},
		TenantUUID:     "00000000-0000-0000-0000-000000000001",
		SourcePlugin:   "com.powerx.plugins.base",
		PayloadVersion: "v1",
	})

	emitter, err := provider.NewEmitter()
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta: event.Meta{
			TenantUUID: "00000000-0000-0000-0000-000000000001",
			RequestID:  "req-1",
			TraceID:    "trace-1",
		},
		Payload: json.RawMessage(`{"channel_id":"c1","status":"ok"}`),
	})
	require.NoError(t, err)

	require.Equal(t, "powerx.channel.master.credential_inspection.v1", received.Topic)
	require.Equal(t, "Bearer sts-token-1", authHeader)
	require.Equal(t, "00000000-0000-0000-0000-000000000001", received.TenantUUID)
	require.Equal(t, "trace-1", received.TraceID)
	require.Equal(t, "c1", received.Payload["channel_id"])
}

func TestHostProvider_UsesSTSTokenProviderInsteadOfInboundBearer(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer server.Close()

	provider := NewHostProvider(HostProviderConfig{
		BaseURL: server.URL,
		TokenProvider: func(context.Context) (string, error) {
			return "sts-token-1", nil
		},
		TenantUUID: "00000000-0000-0000-0000-000000000001",
	})

	emitter, err := provider.NewEmitter()
	require.NoError(t, err)

	ctx := middleware.WithBearerToken(context.Background(), "user-token")
	err = emitter.Emit(ctx, event.Event{
		Topic: "powerx.channel.master.credential_inspection.v1",
		Meta: event.Meta{
			TenantUUID: "00000000-0000-0000-0000-000000000001",
			RequestID:  "req-1",
			TraceID:    "trace-1",
		},
		Payload: json.RawMessage(`{"channel_id":"c1","status":"ok"}`),
	})
	require.NoError(t, err)

	require.Equal(t, "Bearer sts-token-1", authHeader)
}

func TestHostProvider_NewEmitterRequiresBaseConfig(t *testing.T) {
	provider := NewHostProvider(HostProviderConfig{})
	_, err := provider.NewEmitter()
	require.Error(t, err)
}

func TestHostEmitter_EmitRequiresTenantUUID(t *testing.T) {
	provider := NewHostProvider(HostProviderConfig{
		BaseURL: "http://127.0.0.1:65535",
		TokenProvider: func(context.Context) (string, error) {
			return "sts-token-1", nil
		},
		SourcePlugin:   "com.powerx.plugins.base",
		PayloadVersion: "v1",
	})

	emitter, err := provider.NewEmitter()
	require.NoError(t, err)

	err = emitter.Emit(context.Background(), event.Event{
		Topic:   "powerx.channel.master.credential_inspection.v1",
		Payload: json.RawMessage(`{"channel_id":"c1"}`),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant_uuid")
}
