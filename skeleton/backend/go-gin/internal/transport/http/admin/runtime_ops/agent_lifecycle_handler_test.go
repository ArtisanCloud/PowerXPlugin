package runtime_ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	powerxagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type agentLifecycleStub struct{}

func (agentLifecycleStub) Invoke(context.Context, powerxagent.AgentInvokeRequest) (powerxagent.AgentInvokeResponse, error) {
	return powerxagent.AgentInvokeResponse{}, nil
}
func (agentLifecycleStub) StreamSSE(context.Context, url.Values, func(powerxagent.AgentStreamEvent) error) error {
	return nil
}

func (agentLifecycleStub) GetHealthSummary(context.Context, string) (*powerxagent.HealthSummary, error) {
	return &powerxagent.HealthSummary{Status: "healthy"}, nil
}
func (agentLifecycleStub) ListHealthHistory(context.Context, string, int, int) (*powerxagent.HealthHistory, error) {
	return &powerxagent.HealthHistory{}, nil
}
func (agentLifecycleStub) GetBridgeState(context.Context, string, int) (*json.RawMessage, error) {
	return nil, nil
}
func (agentLifecycleStub) Freeze(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}
func (agentLifecycleStub) Recover(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}
func (agentLifecycleStub) Rebalance(context.Context, string, powerxagent.BridgeRebalanceInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}

func TestAgentLifecycleHandlerUsesFrameworkRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := fwagent.NewRuntime(provider.ModeLocal, agentLifecycleStub{}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	router := gin.New()
	router.GET("/agents/:agentUUID/health", NewAgentLifecycleHandler(&app.Deps{AgentLifecycle: runtime}).Health)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/agent-uuid/health?tenant_uuid=tenant-uuid", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !json.Valid(rec.Body.Bytes()) || !bytes.Contains(rec.Body.Bytes(), []byte(`"healthy"`)) {
		t.Fatalf("expected framework local lifecycle response, body=%s", rec.Body.String())
	}
}

func TestAgentLifecycleHandlerFailsWhenSelectedAdapterIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := fwagent.NewRuntime(provider.ModeLocal, nil, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	router := gin.New()
	router.GET("/agents/:agentUUID/health", NewAgentLifecycleHandler(&app.Deps{AgentLifecycle: runtime}).Health)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/agents/agent-uuid/health?tenant_uuid=tenant-uuid", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAgentLifecycleReadHandlersUseRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := fwagent.NewRuntime(provider.ModeLocal, agentLifecycleStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	handler := NewAgentLifecycleHandler(&app.Deps{AgentLifecycle: runtime})
	router.GET("/agents/:agentUUID/health/history", handler.HealthHistory)
	router.GET("/agents/:agentUUID/bridge/state", handler.BridgeState)
	for _, path := range []string{"/agents/agent-uuid/health/history?tenant_uuid=tenant-uuid", "/agents/agent-uuid/bridge/state?tenant_uuid=tenant-uuid"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}
