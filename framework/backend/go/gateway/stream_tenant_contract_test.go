package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

func assertNoStreamTenantOverride(t *testing.T, r *http.Request) {
	t.Helper()
	for name := range r.Header {
		if strings.Contains(strings.ToLower(name), "tenant") {
			t.Error("STREAM_TENANT_HEADER_FORBIDDEN", name)
		}
	}
	if r.URL.RawQuery != "" {
		t.Error("STREAM_QUERY_FORBIDDEN", r.URL.RawQuery)
	}
}

func TestPublicStreamTenantContextPreservesProtocolAndBody(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		for _, scheme := range []string{"ApiKey", "Bearer"} {
			for _, protocol := range []string{"rest", "core_internal"} {
				for _, tenant := range []string{"", "00000000-0000-4000-8000-000000000001"} {
					t.Run(kind+"/"+scheme+"/"+protocol+"/"+tenant, func(t *testing.T) {
						input := streamInputForTest()
						input.TenantUUID = tenant
						input.PreferredProtocol = protocol
						if protocol == "core_internal" {
							input.CapabilityID = "com.corex.runtime.realtime.stream"
							input.Environment = ""
							input.Body = map[string]any{"contract_version": "realtime.service.v1", "tenant_key": "00000000-0000-4000-8000-000000000001", "namespace": "fixture.realtime", "operation": "subscribe", "cursor": "17-2"}
						}
						var calls atomic.Int32
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							calls.Add(1)
							assertNoStreamTenantOverride(t, r)
							if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/invocations/stream" || r.Header.Get("Authorization") != scheme+" fixture-credential" || r.Header.Get("Accept") != "text/event-stream" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Request-ID") != input.RequestID || r.Header.Get("X-Trace-ID") != input.TraceID {
								t.Error("STREAM_TRANSPORT_INVALID")
							}
							var received map[string]any
							if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
								t.Error(err)
								return
							}
							payload := map[string]any{"body": input.Body}
							if input.Environment != "" {
								payload["query"] = map[string]any{"env": input.Environment}
							}
							if !reflect.DeepEqual(received, map[string]any{"capability_id": input.CapabilityID, "preferred_protocol": protocol, "payload": payload}) {
								t.Error("STREAM_BUSINESS_PAYLOAD_INVALID", received)
							}
							w.Header().Set("Content-Type", "text/event-stream")
							w.Header().Set("X-Trace-ID", "core-trace")
							w.Header().Set("X-Request-ID", "core-request")
							_, _ = fmt.Fprint(w, "event: ready\ndata: {}\n\n")
						}))
						defer server.Close()
						events := 0
						err := streamForTest(t, kind, scheme, server.URL, "")(context.Background(), input, func(event hostcontract.StreamEvent) error {
							events++
							if event.Event != "ready" || event.TraceID != "core-trace" || event.RequestID != "core-request" {
								t.Error("STREAM_DIAGNOSTICS_LOST", event)
							}
							return nil
						})
						if err != nil || calls.Load() != 1 || events != 1 {
							t.Fatal("STREAM_NOT_DELIVERED", err, calls.Load(), events)
						}
					})
				}
			}
		}
	}
}

func TestPublicStreamInvalidTenantSkipsCredentialsAndRequest(t *testing.T) {
	for _, kind := range []string{"gateway", "capability"} {
		t.Run(kind, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
			defer server.Close()
			credentialCalls := 0
			provider := func(context.Context) (string, error) {
				credentialCalls++
				return "", errors.New("FIXTURE_CREDENTIAL_UNAVAILABLE")
			}
			var stream invokeStreamFunc
			if kind == "gateway" {
				client, err := gateway.NewClient(gateway.Config{BaseURL: server.URL, AuthScheme: "bearer", TokenProvider: provider})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = client.Close() })
				stream = client.InvokeStream
			} else {
				client, err := capability.NewClientWithTokenProvider(capability.Config{BaseURL: server.URL}, capability.TokenProviderFunc(provider), nil)
				if err != nil {
					t.Fatal(err)
				}
				stream = client.InvokeStream
			}
			for _, tenant := range []string{"invalid", " ", "00000000-0000-4000-8000-000000000001 ", "AAAAAAAA-0000-4000-8000-000000000001", "{00000000-0000-4000-8000-000000000001}", "00000000000040008000000000000001", "urn:uuid:00000000-0000-4000-8000-000000000001"} {
				input := streamInputForTest()
				input.TenantUUID = tenant
				err := stream(context.Background(), input, func(hostcontract.StreamEvent) error { t.Error("INVALID_TENANT_EVENT"); return nil })
				status, code, _ := failureForTest(t, err)
				if status != http.StatusBadRequest || code != "CAPABILITY_STREAM_INVALID_ARGUMENT" {
					t.Error("INVALID_TENANT_FAILURE_LOST", tenant, err)
				}
			}
			if credentialCalls != 0 || requests.Load() != 0 {
				t.Fatal("INVALID_TENANT_STARTED_IO", credentialCalls, requests.Load())
			}
		})
	}
}
