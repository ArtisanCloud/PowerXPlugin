package gateway

import (
	"context"
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
			ToolToken:  "token",
			TenantUUID: "11111111-1111-1111-1111-111111111111",
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
	resp *frameworkgateway.Response
	err  error
}

func (s *stubTransport) Invoke(ctx context.Context, req frameworkgateway.InvokeRequest) (*frameworkgateway.Response, error) {
	return s.resp, s.err
}

func (s *stubTransport) Close() error { return nil }

func TestListPlatformCapabilitiesSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tenant/capabilities" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("source"); got != "corex" {
			http.Error(w, "missing source", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer token" {
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
			ToolToken:  "token",
			TenantUUID: "00000000-0000-0000-0000-000000000001",
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
			ToolToken:  "token",
			TenantUUID: "00000000-0000-0000-0000-000000000001",
		},
	}
	client := NewClient(cfg, nil)

	_, listErr := client.ListPlatformCapabilities(context.Background(), ListPlatformCapabilitiesOptions{Source: "corex"})
	require.Error(t, listErr)
}
