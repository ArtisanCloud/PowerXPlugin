package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mediaFixture(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("testdata/ai-media-v2.json")
	require.NoError(t, err)
	var fixture map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fixture))
	return fixture
}
func fixtureTask(t *testing.T) MediaTask {
	t.Helper()
	var task MediaTask
	require.NoError(t, json.Unmarshal(mediaFixture(t)["task_result"], &task))
	return task
}

func TestMediaV2BothCredentialsSharePublishedFixture(t *testing.T) {
	fixture := mediaFixture(t)
	for _, auth := range []string{"ApiKey test-key", "Bearer short-lived-sts"} {
		t.Run(auth[:6], func(t *testing.T) {
			var bodies []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/v1/tenant/invocations", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, auth, r.Header.Get("Authorization"))
				require.Empty(t, r.URL.RawQuery)
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				require.Equal(t, "core_internal", request["preferred_protocol"])
				payload := request["payload"].(map[string]any)
				require.Len(t, payload, 1)
				body := payload["body"].(map[string]any)
				bodies = append(bodies, body)
				result := fixture["task_result"]
				switch request["capability_id"] {
				case CapabilityMediaModels:
					result = fixture["catalog_result"]
				case CapabilityMediaProviders:
					result = json.RawMessage(`{"contract_version":2,"providers":[{"provider":"qwen-cn","label":"qwen"}]}`)
				case "com.corex.ai.vlm.invoke":
					result = json.RawMessage(`{"contract_version":2,"type":"analysis","text":"","provider":"qwen-cn","model":"qwen-vl-max","trace_id":"analysis-trace"}`)
				case "com.corex.ai.video.invoke", "com.corex.ai.tts.invoke":
					task := fixtureTask(t)
					task.Capability = VideoGeneration
					if request["capability_id"] == "com.corex.ai.tts.invoke" {
						task.Capability = SpeechSynthesis
					}
					result, _ = json.Marshal(task)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"data":{"protocol_used":"core_internal","fallback_used":false,"payload":{"result":%s}}}`, result)
			}))
			defer server.Close()
			var client *Client
			var err error
			if auth[0] == 'A' {
				client, err = NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "test-key"}, server.Client())
			} else {
				client, err = NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return "short-lived-sts", nil }), server.Client())
			}
			require.NoError(t, err)
			catalog, err := client.ListMediaModels(context.Background(), MediaCatalogFilter{Provider: "qwen-cn", Capability: ImageGeneration})
			require.NoError(t, err)
			require.Equal(t, float64(1), catalog.Items[0].Operations[ImageGeneration].Defaults["n"])
			_, err = client.ListMediaProviders(context.Background(), ImageGeneration)
			require.NoError(t, err)
			var input MediaInvokeInput
			require.NoError(t, json.Unmarshal(fixture["invoke_operation"], &input))
			task, err := client.ImageInvoke(context.Background(), input)
			require.NoError(t, err)
			require.Equal(t, fixtureTask(t).TaskUUID, task.TaskUUID)
			_, err = client.ImageInvoke(context.Background(), input)
			require.NoError(t, err)
			require.True(t, reflect.DeepEqual(bodies[2], bodies[3]))
			var expected any
			require.NoError(t, json.Unmarshal(fixture["invoke_operation"], &expected))
			require.Equal(t, expected, bodies[2])
			_, err = client.VideoInvoke(context.Background(), input)
			require.NoError(t, err)
			_, err = client.TTSInvoke(context.Background(), input)
			require.NoError(t, err)
			analysis, err := client.VLMInvoke(context.Background(), input)
			require.NoError(t, err)
			require.Equal(t, "analysis", analysis.Type)
			_, err = client.GetMediaTask(context.Background(), task.TaskUUID)
			require.NoError(t, err)
			_, err = client.CancelMediaTask(context.Background(), task.TaskUUID)
			require.NoError(t, err)
			before := len(bodies)
			input.IdempotencyKey = ""
			_, err = client.ImageInvoke(context.Background(), input)
			require.EqualError(t, err, "AI_IDEMPOTENCY_KEY_REQUIRED")
			require.Len(t, bodies, before)
		})
	}
}

func TestMediaErrorsRetainStatusFieldsAndTrace(t *testing.T) {
	for _, status := range []int{401, 403, 409, 422, 429, 502, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Trace-ID", "header-trace")
				w.Header().Set("X-Request-ID", "header-request")
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, `{"error_code":"CAPABILITY_INVOKE_FAILED","details":{"reason_code":"AI_PARAMETER_INVALID","parameter":"temperature","reason":"unsupported_parameter","upstream_status":429,"upstream_code":"provider-limit","provider_trace_id":"provider-trace"}}`)
			}))
			defer server.Close()
			client, err := NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "key"}, server.Client())
			require.NoError(t, err)
			_, err = client.ListMediaModels(context.Background(), MediaCatalogFilter{})
			var typed *HTTPError
			require.True(t, errors.As(err, &typed))
			require.Equal(t, status, typed.StatusCode)
			require.Equal(t, "AI_PARAMETER_INVALID", typed.ReasonCode)
			require.Equal(t, "CAPABILITY_INVOKE_FAILED", typed.ErrorCode)
			require.Equal(t, "header-trace", typed.TraceID)
			require.Equal(t, "header-request", typed.RequestID)
			require.Equal(t, 429, typed.UpstreamStatus)
			require.Equal(t, "provider-limit", typed.UpstreamCode)
			require.Equal(t, "provider-trace", typed.ProviderTraceID)
			require.Equal(t, "temperature", typed.Details["parameter"])
		})
	}
}

func TestMediaSSERecoversSnapshotDeduplicatesAndRefreshesCredentials(t *testing.T) {
	task := fixtureTask(t)
	var connections atomic.Int32
	var tokens atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connections.Add(1)
		require.Equal(t, "/api/v1/ai/media/tasks/"+task.TaskUUID+"/events", r.URL.Path)
		require.Empty(t, r.URL.RawQuery)
		require.Equal(t, fmt.Sprintf("Bearer token-%d", n), r.Header.Get("Authorization"))
		if n == 2 {
			require.Equal(t, task.TaskUUID+":2", r.Header.Get("Last-Event-ID"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		state := task
		state.Status = "running"
		state.Phase = "waiting_provider"
		state.Revision = 2
		raw, _ := json.Marshal(state)
		_, _ = fmt.Fprintf(w, ": keepalive\n\nid: %s\nevent: status\ndata: %s\n\n", state.EventID(), raw)
		if n == 1 {
			_, _ = fmt.Fprint(w, "event: reconnect\ndata: {\"reason_code\":\"AI_EVENT_REAUTHORIZE\"}\n\n")
			return
		}
		raw, _ = json.Marshal(task)
		_, _ = fmt.Fprintf(w, "id: %s\nevent: status\ndata: %s\n\n", task.EventID(), raw)
	}))
	defer server.Close()
	client, err := NewClientWithTokenProvider(Config{BaseURL: server.URL}, TokenProviderFunc(func(context.Context) (string, error) { return fmt.Sprintf("token-%d", tokens.Add(1)), nil }), server.Client())
	require.NoError(t, err)
	var revisions []int64
	err = client.SubscribeMediaTask(context.Background(), task.TaskUUID, MediaTaskSubscriptionOptions{ReconnectDelay: time.Millisecond}, func(event MediaTaskEvent) error {
		if event.Task != nil {
			revisions = append(revisions, event.Task.Revision)
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []int64{2, 4}, revisions)
	require.EqualValues(t, 2, connections.Load())
}

func TestMediaSSERevocationAndCursorMismatchStop(t *testing.T) {
	task := fixtureTask(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: error\ndata: {\"reason_code\":\"AI_EVENT_ACCESS_REVOKED\"}\n\n")
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "key"}, server.Client())
	require.NoError(t, err)
	err = client.SubscribeMediaTask(context.Background(), task.TaskUUID, MediaTaskSubscriptionOptions{}, func(MediaTaskEvent) error { return nil })
	var typed *HTTPError
	require.True(t, errors.As(err, &typed))
	require.Equal(t, 403, typed.StatusCode)
	require.EqualValues(t, 1, requests.Load())
	err = client.SubscribeMediaTask(context.Background(), task.TaskUUID, MediaTaskSubscriptionOptions{LastEventID: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee:1"}, func(MediaTaskEvent) error { return nil })
	require.EqualError(t, err, "AI_EVENT_CURSOR_INVALID")
	require.EqualValues(t, 1, requests.Load())
}

func TestMediaSSEBoundedTransientFailureRetainsFinalHTTPError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Contains(t, r.URL.Path, "/events")
		w.Header().Set("X-Trace-ID", "events-trace")
		w.WriteHeader(503)
		_, _ = fmt.Fprint(w, `{"error_code":"AI_EVENTS_UNAVAILABLE","details":{"reason_code":"AI_EVENTS_UNAVAILABLE"}}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "key"}, server.Client())
	require.NoError(t, err)
	err = client.SubscribeMediaTask(context.Background(), fixtureTask(t).TaskUUID, MediaTaskSubscriptionOptions{MaxReconnects: 2, ReconnectDelay: time.Millisecond}, func(MediaTaskEvent) error { return nil })
	var typed *HTTPError
	require.True(t, errors.As(err, &typed))
	require.Equal(t, 503, typed.StatusCode)
	require.Equal(t, "events-trace", typed.TraceID)
	require.EqualValues(t, 3, requests.Load())
}

func TestMediaSSECancellationClosesTheRead(t *testing.T) {
	task := fixtureTask(t)
	task.Status = "running"
	task.Phase = "waiting_provider"
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(closed)
		w.Header().Set("Content-Type", "text/event-stream")
		raw, _ := json.Marshal(task)
		_, _ = fmt.Fprintf(w, "id: %s\nevent: status\ndata: %s\n\n", task.EventID(), raw)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, AuthScheme: "apikey", APIKey: "key"}, server.Client())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = client.SubscribeMediaTask(ctx, task.TaskUUID, MediaTaskSubscriptionOptions{}, func(event MediaTaskEvent) error {
		if event.Task != nil {
			cancel()
		}
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("AI_EVENT_CONNECTION_NOT_RELEASED")
	}
}
