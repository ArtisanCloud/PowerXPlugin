package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

func TestServiceFailuresAcrossPublicClientsAndCredentials(t *testing.T) {
	raw, err := os.ReadFile("../runtime/powerx/hostcontract/testdata/plugin-service-errors-stream.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name   string          `json:"name"`
			Status int             `json:"status"`
			Body   json.RawMessage `json:"body"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, clientKind := range []string{"gateway", "capability"} {
		for _, scheme := range []string{"Bearer", "ApiKey"} {
			for _, test := range fixture.Cases {
				t.Run(clientKind+"/"+scheme+"/"+test.Name, func(t *testing.T) {
					calls := 0
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						if r.Header.Get("Authorization") != scheme+" fixture-credential" || r.Method != "POST" || r.URL.Path != "/api/v1/tenant/invocations" {
							t.Error("SERVICE_TRANSPORT_CHANGED")
						}
						w.WriteHeader(test.Status)
						_, _ = w.Write(test.Body)
					}))
					defer server.Close()
					var resultError error
					if clientKind == "gateway" {
						cfg := gateway.Config{BaseURL: server.URL, AuthScheme: "bearer", BearerToken: "fixture-credential"}
						if scheme == "ApiKey" {
							cfg.AuthScheme = "apikey"
							cfg.APIKey = "fixture-credential"
							cfg.BearerToken = ""
						}
						client, err := gateway.NewClient(cfg)
						if err != nil {
							t.Fatal(err)
						}
						defer client.Close()
						_, resultError = client.Invoke(context.Background(), gateway.InvokeRequest{CapabilityID: "com.example.service", PreferredProtocol: "rest", Payload: map[string]any{"body": map[string]any{}}})
						var e *gateway.InvocationError
						if !errors.As(resultError, &e) || e.StatusCode != test.Status || e.ErrorCode != test.Name || e.ReasonCode != test.Name || e.TraceID != "11111111-1111-4111-8111-111111111111" || e.RequestID == "" || e.Details == nil || len(e.Errors) != 1 || e.Errors[0].Code != test.Name {
							t.Fatal("GATEWAY_FAILURE_LOST", resultError)
						}
						if test.Name == "CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH" && (e.Details.ExpectedProviderVersion != "1.2.3" || e.Details.ActualProviderVersion != "1.2.2") {
							t.Fatal("GATEWAY_RUNTIME_VERSION_LOST")
						}
					} else {
						cfg := capability.Config{BaseURL: server.URL, AuthScheme: "bearer", BearerToken: "fixture-credential"}
						if scheme == "ApiKey" {
							cfg.AuthScheme = "apikey"
							cfg.APIKey = "fixture-credential"
							cfg.BearerToken = ""
						}
						client, err := capability.NewClient(cfg, server.Client())
						if err != nil {
							t.Fatal(err)
						}
						_, resultError = client.Invoke(context.Background(), capability.InvokeInput{CapabilityID: "com.example.service", PreferredProtocol: "rest", Payload: map[string]any{"body": map[string]any{}}})
						var e *capability.HTTPError
						if !errors.As(resultError, &e) || e.StatusCode != test.Status || e.ErrorCode != test.Name || e.ReasonCode != test.Name || e.TraceID != "11111111-1111-4111-8111-111111111111" || e.Details == nil {
							t.Fatal("CAPABILITY_FAILURE_LOST", resultError)
						}
						if test.Name == "MESSAGING_IDEMPOTENCY_CONFLICT" && (e.Details.Retryable == nil || *e.Details.Retryable || e.Details.NextActionCode != "inspect_existing_request") {
							t.Fatal("CAPABILITY_RECOVERY_LOST")
						}
					}
					if calls != 1 {
						t.Fatal("IMPLICIT_RETRY", calls)
					}
					if strings.Contains(fmt.Sprint(resultError), "fixture-credential") {
						t.Fatal("SERVICE_CREDENTIAL_LEAKED")
					}
				})
			}
		}
	}
}
