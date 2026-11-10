package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

type invokeStreamFunc func(context.Context, hostcontract.StreamInput, func(hostcontract.StreamEvent) error) error

func streamForTest(t *testing.T, kind, scheme, base, prefix string) invokeStreamFunc {
	t.Helper()
	if kind == "gateway" {
		cfg := gateway.Config{BaseURL: base, APIPrefix: prefix, AuthScheme: "bearer", BearerToken: "fixture-credential"}
		if scheme == "ApiKey" {
			cfg.AuthScheme = "apikey"
			cfg.APIKey = "fixture-credential"
			cfg.BearerToken = ""
		}
		client, err := gateway.NewClient(cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = client.Close() })
		return client.InvokeStream
	}
	cfg := capability.Config{BaseURL: base, APIPrefix: prefix, AuthScheme: "bearer", BearerToken: "fixture-credential"}
	if scheme == "ApiKey" {
		cfg.AuthScheme = "apikey"
		cfg.APIKey = "fixture-credential"
		cfg.BearerToken = ""
	}
	client, err := capability.NewClient(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	return client.InvokeStream
}

func failureForTest(t *testing.T, err error) (int, string, *hostcontract.InvocationFailureDetails) {
	t.Helper()
	var gw *gateway.InvocationError
	var cap *capability.HTTPError
	if errors.As(err, &gw) {
		return gw.StatusCode, gw.ReasonCode, gw.Details
	}
	if errors.As(err, &cap) {
		return cap.StatusCode, cap.ReasonCode, cap.Details
	}
	t.Fatal("TYPED_STREAM_FAILURE_REQUIRED", err)
	return 0, "", nil
}

func streamInputForTest() hostcontract.StreamInput {
	return hostcontract.StreamInput{CapabilityID: "com.corex.ai.llm.stream", TenantUUID: "00000000-0000-4000-8000-000000000001", Environment: "dev", Body: map[string]any{"model_key": "fixture-model", "inputs": []any{map[string]any{"type": "text", "text": "fixture"}}}, RequestID: "caller-request", TraceID: "caller-trace"}
}

func TestPublicStreamCredentialsFixedPayloadAndPrefix(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, scheme := range []string{"Bearer", "ApiKey"} {
			t.Run(kind+"/"+scheme, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					assertNoStreamTenantOverride(t, r)
					if r.Method != "POST" || r.URL.Path != "/proxy/api/v2/tenant/invocations/stream" || r.Header.Get("Authorization") != scheme+" fixture-credential" || r.Header.Get("Accept") != "text/event-stream" || r.Header.Get("X-Request-ID") != "caller-request" || r.Header.Get("X-Trace-ID") != "caller-trace" {
						t.Error("STREAM_TRANSPORT_INVALID")
					}
					var input map[string]any
					if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
						t.Error(err)
					}
					payload := input["payload"].(map[string]any)
					if len(input) != 3 || len(payload) != 2 || input["capability_id"] != "com.corex.ai.llm.stream" || input["preferred_protocol"] != "rest" || payload["query"].(map[string]any)["env"] != "dev" || payload["body"].(map[string]any)["model_key"] != "fixture-model" {
						t.Error("STREAM_BUSINESS_PAYLOAD_INVALID", input)
					}
					w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
					w.Header().Set("X-Trace-ID", "core-trace")
					_, _ = fmt.Fprint(w, ": heartbeat\n\nid: event-1\nevent: token\ndata: {\"delta\":\ndata: \"hi\"}\n\nretry: 1\nevent: end\ndata: {}\n\n")
				}))
				defer server.Close()
				stream := streamForTest(t, kind, scheme, server.URL+"/proxy/api/v2", "/proxy/api/v2")
				var events []hostcontract.StreamEvent
				if err := stream(context.Background(), streamInputForTest(), func(event hostcontract.StreamEvent) error { events = append(events, event); return nil }); err != nil {
					t.Fatal(err)
				}
				if len(events) != 2 || events[0].Event != "token" || events[0].ID != "event-1" || events[0].Data != "{\"delta\":\n\"hi\"}" || events[0].TraceID != "core-trace" || events[1].Event != "end" || calls != 1 {
					t.Fatal("STREAM_EVENTS_LOST", events, calls)
				}
			})
		}
	}
}

func TestPublicStreamDenialsRecoveryAndNoReplay(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, test := range []struct {
			status int
			code   string
		}{{401, "IAM_UNAUTHORIZED"}, {403, "IAM_FORBIDDEN"}, {404, "CAPABILITY_NOT_FOUND"}, {409, "CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH"}, {503, "MESSAGING_TEMPORARILY_UNAVAILABLE"}, {400, "CAPABILITY_SERVICE_STREAM_UNSUPPORTED"}} {
			t.Run(kind+"/"+test.code, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					w.WriteHeader(test.status)
					_ = json.NewEncoder(w).Encode(map[string]any{"error_code": test.code, "reason_code": test.code, "details": map[string]any{"trace_id": "core-trace", "retryable": true, "next_action_code": "inspect_request", "expected_provider_version": "2.0.0", "actual_provider_version": "1.0.0"}, "message": "private-upstream-text"})
				}))
				defer server.Close()
				stream := streamForTest(t, kind, "ApiKey", server.URL, "")
				err := stream(context.Background(), streamInputForTest(), func(hostcontract.StreamEvent) error { t.Error("EVENT_AFTER_REJECTION"); return nil })
				status, code, details := failureForTest(t, err)
				if status != test.status || code != test.code || details == nil || details.TraceID != "core-trace" || details.Retryable == nil || !*details.Retryable || details.NextActionCode != "inspect_request" || calls != 1 {
					t.Fatal("STREAM_DENIAL_LOST", err, details, calls)
				}
				if strings.Contains(err.Error(), "private-upstream-text") {
					t.Fatal("STREAM_ERROR_LEAKED")
				}
			})
		}
	}
}

func TestPublicStreamRejectsRedirectAndNonSSE(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, scenario := range []string{"redirect", "json", "too-large"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					switch scenario {
					case "redirect":
						http.Redirect(w, r, "/not-declared", 307)
					case "json":
						w.Header().Set("Content-Type", "application/json")
						_, _ = fmt.Fprint(w, `{"private":"body"}`)
					case "too-large":
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprint(w, "data: "+strings.Repeat("x", hostcontract.MaxInvocationResponseBytes+1)+"\n\n")
					}
				}))
				defer server.Close()
				err := streamForTest(t, kind, "Bearer", server.URL, "")(context.Background(), streamInputForTest(), func(hostcontract.StreamEvent) error { t.Error("INVALID_STREAM_EMITTED"); return nil })
				status, _, _ := failureForTest(t, err)
				if status < 300 || calls != 1 || strings.Contains(err.Error(), "private") {
					t.Fatal("INVALID_STREAM_ACCEPTED", err, calls)
				}
			})
		}
	}
}

func TestPublicStreamCancellationAndCallbackCloseUpstream(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, scenario := range []string{"cancel", "callback", "timeout"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				closed := make(chan struct{})
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprint(w, "event: token\ndata: {}\n\n")
					w.(http.Flusher).Flush()
					select {
					case <-r.Context().Done():
					case <-time.After(2 * time.Second):
					}
					close(closed)
				}))
				defer server.Close()
				stream := streamForTest(t, kind, "Bearer", server.URL, "")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if scenario == "timeout" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, 200*time.Millisecond)
					defer stop()
				}
				callbackErr := errors.New("FIXTURE_CALLBACK_STOP")
				err := stream(ctx, streamInputForTest(), func(hostcontract.StreamEvent) error {
					if scenario == "callback" {
						return callbackErr
					}
					if scenario == "cancel" {
						cancel()
					}
					return nil
				})
				switch scenario {
				case "callback":
					if !errors.Is(err, callbackErr) {
						t.Fatal("CALLBACK_ERROR_LOST", err)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) {
						t.Fatal("CANCELLATION_LOST", err)
					}
				case "timeout":
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatal("TIMEOUT_LOST", err)
					}
				}
				select {
				case <-closed:
				case <-time.After(time.Second):
					t.Fatal("UPSTREAM_NOT_CLOSED")
				}
				if calls.Load() != 1 {
					t.Fatal("STREAM_REPLAYED")
				}
			})
		}
	}
}

func TestPublicStreamInvalidInputMakesNoRequest(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
			defer server.Close()
			stream := streamForTest(t, kind, "ApiKey", server.URL, "")
			for _, mutate := range []func(*hostcontract.StreamInput){func(v *hostcontract.StreamInput) { v.CapabilityID = "" }, func(v *hostcontract.StreamInput) { v.Environment = "custom" }, func(v *hostcontract.StreamInput) { v.Body = nil }, func(v *hostcontract.StreamInput) { v.Body = []string{"not-object"} }, func(v *hostcontract.StreamInput) { v.RequestID = "bad\r\nrequest" }} {
				input := streamInputForTest()
				mutate(&input)
				if err := stream(context.Background(), input, func(hostcontract.StreamEvent) error { return nil }); err == nil {
					t.Fatal("INVALID_INPUT_ACCEPTED")
				}
			}
			if err := stream(context.Background(), streamInputForTest(), nil); err == nil || calls != 0 {
				t.Fatal("INVALID_INPUT_SUBMITTED", calls)
			}
		})
	}
}
