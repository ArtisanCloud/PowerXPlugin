package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/stretchr/testify/require"
)

func TestInvokeUsesMockWhenConfigured(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			UseMock: []string{"media"},
		},
	}
	client := NewClient(cfg, nil)
	result, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
		Payload:      map[string]any{"folder": "inbox"},
	})
	require.NoError(t, err)
	require.True(t, result.Mock)
	require.Equal(t, "mock", result.Status)
	require.Contains(t, result.Data, "echoPayload")
}

func TestInvokeReturnsUnavailableWhenNotConfigured(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{},
	}
	client := NewClient(cfg, nil)
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.eventfabric.publish",
		Action:       "Create",
	})
	var unavailable *UnavailableError
	require.Error(t, err)
	require.True(t, errors.As(err, &unavailable))
	require.Contains(t, unavailable.Error(), "gateway 不可用")
}

func TestInvokeUsesRealTransport(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	client.transport = &stubTransport{
		resp: &frameworkgateway.Response{
			TraceID: "trace-123",
			Status:  "ok",
			Data: map[string]any{
				"mediaId": "asset-1",
			},
		},
	}

	result, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
	})
	require.NoError(t, err)
	require.False(t, result.Mock)
	require.Equal(t, "trace-123", result.TraceID)
	require.Equal(t, "asset-1", result.Data["mediaId"])
}

type stubTransport struct {
	resp    *frameworkgateway.Response
	err     error
	lastReq frameworkgateway.InvokeRequest
}

func (s *stubTransport) Invoke(ctx context.Context, req frameworkgateway.InvokeRequest) (*frameworkgateway.Response, error) {
	s.lastReq = req
	return s.resp, s.err
}

func (s *stubTransport) Close() error { return nil }

func TestListPlatformCapabilitiesSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/capabilities" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("source"); got != "corex" {
			http.Error(w, "missing source", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "ApiKey token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{
			"code":200,
			"message":"success",
			"data":{
				"items":[
					{
						"capability_id":"com.corex.media.assets.read",
						"plugin_id":"corex",
						"plugin_version":"1.0.0",
						"source":"corex",
						"categories":["media"],
						"protocols":[{"channel":"rest","endpoint":"/api/v1/media/assets","method":"GET"}],
						"capabilities_hash":"abc",
						"protocol_hash":"def",
						"status":"published"
					}
				]
				}
			}`)
	})
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip test: cannot listen on local port in this environment: %v", err)
	}
	server := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: handler,
		},
	}
	server.Start()
	defer server.Close()

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    server.URL,
			AuthScheme: "apikey",
			APIKey:     "token",
			Timeout:    2 * time.Second,
		},
	}
	client := NewClient(cfg, nil)

	records, err := client.ListPlatformCapabilities(context.Background(), ListPlatformCapabilitiesOptions{Source: "corex"})
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "com.corex.media.assets.read", records[0].CapabilityID)
	require.Equal(t, "rest", records[0].Protocols[0].Channel)
}

func TestAPIKeyRequestsDoNotSendTenantHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("tenant_uuid"))
		require.Empty(t, r.Header.Get("X-Tenant-UUID"))
		require.Equal(t, "ApiKey token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":200,"message":"success","data":{"items":[]}}`)
	})
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip test: cannot listen on local port in this environment: %v", err)
	}
	server := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: handler,
		},
	}
	server.Start()
	defer server.Close()

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    server.URL,
			AuthScheme: "apikey",
			APIKey:     "token",
			Timeout:    2 * time.Second,
		},
	}
	client := NewClient(cfg, nil)

	_, err = client.ListAgents(context.Background(), "dev")
	require.NoError(t, err)
}

func TestListPlatformCapabilitiesHTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip test: cannot listen on local port in this environment: %v", err)
	}
	server := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: handler,
		},
	}
	server.Start()
	defer server.Close()

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    server.URL,
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)

	_, listErr := client.ListPlatformCapabilities(context.Background(), ListPlatformCapabilitiesOptions{Source: "corex"})
	require.Error(t, listErr)
}

func TestInvokePolicyRequiresRequestAuthorization(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	client.transport = &stubTransport{
		resp: &frameworkgateway.Response{TraceID: "trace-1", Status: "ok"},
	}
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
		AuthRequired: true,
	})
	var policyErr *PolicyError
	require.Error(t, err)
	require.True(t, errors.As(err, &policyErr))
	require.Equal(t, "GW_POLICY_AUTH_REQUIRED", policyErr.Code)
}

func TestInvokePolicyTenantScopedRequiresTenantClaim(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	client.transport = &stubTransport{
		resp: &frameworkgateway.Response{TraceID: "trace-2", Status: "ok"},
	}
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
		AuthRequired: true,
		TenantScoped: true,
		Headers: map[string]string{
			"Authorization": "Bearer no-tenant-claim",
		},
	})
	var policyErr *PolicyError
	require.Error(t, err)
	require.True(t, errors.As(err, &policyErr))
	require.Equal(t, "GW_POLICY_TENANT_TOKEN_REQUIRED", policyErr.Code)
}

func TestInvokePolicyAllowsAnonymousAndDisablesDefaultAuth(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	stub := &stubTransport{
		resp: &frameworkgateway.Response{TraceID: "trace-3", Status: "ok"},
	}
	client.transport = stub
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.public.ping",
		Action:       "GET",
		AuthRequired: false,
	})
	require.NoError(t, err)
	require.True(t, stub.lastReq.DisableAuth)
}

func TestInvokePolicyTenantScopedUsesTokenTid(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	stub := &stubTransport{
		resp: &frameworkgateway.Response{TraceID: "trace-4", Status: "ok"},
	}
	client.transport = stub

	tenantID := "11111111-1111-1111-1111-111111111111"
	jwt := buildTestJWTWithTenant(tenantID)
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
		AuthRequired: true,
		TenantScoped: true,
		Headers: map[string]string{
			"Authorization": "Bearer " + jwt,
		},
	})
	require.NoError(t, err)
	require.Equal(t, tenantID, stub.lastReq.TenantUUID)
	require.False(t, stub.lastReq.DisableAuth)
}

func TestInvokePolicyTenantScopedRejectsZeroTenantToken(t *testing.T) {
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    "https://gateway.dev.powerx",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	client.transport = &stubTransport{
		resp: &frameworkgateway.Response{TraceID: "trace-5", Status: "ok"},
	}

	zeroTenant := "00000000-0000-0000-0000-000000000000"
	jwt := buildTestJWTWithTenant(zeroTenant)
	_, err := client.Invoke(context.Background(), InvokeParams{
		CapabilityID: "com.corex.media.assets.manage",
		Action:       "List",
		AuthRequired: true,
		TenantScoped: true,
		Headers: map[string]string{
			"Authorization": "Bearer " + jwt,
		},
	})
	var policyErr *PolicyError
	require.Error(t, err)
	require.True(t, errors.As(err, &policyErr))
	require.Equal(t, "GW_POLICY_ZERO_TENANT_FORBIDDEN", policyErr.Code)
}

func TestParsePlatformErrorString(t *testing.T) {
	code, message := parsePlatformError(json.RawMessage(`"checksum mismatch"`))
	require.Empty(t, code)
	require.Equal(t, "checksum mismatch", message)
}

func TestParsePlatformErrorObject(t *testing.T) {
	code, message := parsePlatformError(json.RawMessage(`{"code":"POWERX_ADMIN_REQUIRED","message":"admin only"}`))
	require.Equal(t, "POWERX_ADMIN_REQUIRED", code)
	require.Equal(t, "admin only", message)
}

func TestNormalizeSHA256Checksum(t *testing.T) {
	raw := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	require.Equal(t, "sha256:"+raw, normalizeSHA256Checksum(raw))
	require.Equal(t, "sha256:"+raw, normalizeSHA256Checksum("sha256:"+raw))
	require.Equal(t, "sha256-"+raw, normalizeSHA256Checksum("sha256-"+raw))
	require.Equal(t, "not-a-sha", normalizeSHA256Checksum("not-a-sha"))
}

func TestGatewayEndpointDoesNotDuplicateAPIPrefix(t *testing.T) {
	require.Equal(t,
		"http://powerx.local/api/v1/agents/stream/sse",
		gatewayEndpoint(&config.GatewayConfig{BaseURL: "http://powerx.local"}, "/agents/stream/sse"),
	)
	require.Equal(t,
		"http://powerx.local/api/v1/agents/stream/sse",
		gatewayEndpoint(&config.GatewayConfig{BaseURL: "http://powerx.local/api/v1"}, "/agents/stream/sse"),
	)
}

func TestStreamAgentSSEUsesGatewayEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		require.Equal(t, "hello", r.URL.Query().Get("q"))
		require.Equal(t, "agent-1", r.URL.Query().Get("agent_id"))
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: end\ndata: {\"ok\":true}\n\n")
	})
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip test: cannot listen on local port in this environment: %v", err)
	}
	server := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: handler,
		},
	}
	server.Start()
	defer server.Close()

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    server.URL + "/api/v1",
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	stream, err := client.StreamAgentSSE(context.Background(), AgentStreamParams{
		AgentID:            "agent-1",
		SessionID:          "session-1",
		TraceID:            "trace-1",
		Query:              "hello",
		Intent:             "agent.bound_capabilities",
		RegenFromMessageID: "215",
	})
	require.NoError(t, err)
	defer stream.Body.Close()
	require.Equal(t, "/api/v1/agents/stream/sse", gotPath)
	require.Contains(t, gotQuery, "agent_id=agent-1")
	require.Contains(t, gotQuery, "intent=agent.bound_capabilities")
	require.Contains(t, gotQuery, "regen_from_message_id=215")
	require.Equal(t, "ApiKey token", gotAuth)
}

func TestStreamAgentSSEMapsUUIDParamsAndEnv(t *testing.T) {
	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/agents/sessions" {
			require.Equal(t, "00000000-0000-4000-8000-000000000001", r.URL.Query().Get("agent_uuid"))
			require.Equal(t, "dev", r.URL.Query().Get("env"))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"code":200,"data":{"items":[{"id":101,"uuid":"10000000-0000-4000-8000-000000000101"}]}}`)
			return
		}
		gotQuery = r.URL.RawQuery
		require.Equal(t, "00000000-0000-4000-8000-000000000001", r.URL.Query().Get("agent_uuid"))
		require.Equal(t, "10000000-0000-4000-8000-000000000101", r.URL.Query().Get("session_uuid"))
		require.Equal(t, "101", r.URL.Query().Get("session_id"))
		require.Equal(t, "dev", r.URL.Query().Get("env"))
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: end\ndata: {\"ok\":true}\n\n")
	})
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skip test: cannot listen on local port in this environment: %v", err)
	}
	server := &httptest.Server{
		Listener: l,
		Config: &http.Server{
			Handler: handler,
		},
	}
	server.Start()
	defer server.Close()

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{
			BaseURL:    server.URL,
			AuthScheme: "apikey",
			APIKey:     "token",
		},
	}
	client := NewClient(cfg, nil)
	stream, err := client.StreamAgentSSE(context.Background(), AgentStreamParams{
		AgentID:   "00000000-0000-4000-8000-000000000001",
		SessionID: "10000000-0000-4000-8000-000000000101",
		TraceID:   "trace-1",
		Query:     "hello",
		Env:       "dev",
	})
	require.NoError(t, err)
	defer stream.Body.Close()
	require.Contains(t, gotQuery, "agent_uuid=")
	require.NotContains(t, gotQuery, "agent_id=")
	require.Contains(t, gotQuery, "session_id=101")
}

func buildTestJWTWithTenant(tid string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"tid":"%s"}`, tid)))
	return header + "." + payload + "."
}
