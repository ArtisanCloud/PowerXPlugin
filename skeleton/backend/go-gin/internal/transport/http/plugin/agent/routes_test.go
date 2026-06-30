package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	cfgpkg "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testTenantUUID = "6b5d0240-9920-46da-b707-88200e0f51ea"

type stubGateway struct{}
type captureGateway struct {
	stubGateway
	params gateway.AgentStreamParams
}
type captureSessionGateway struct {
	stubGateway
	listOpts gateway.AgentSessionListOptions
}

func (stubGateway) Enabled() bool { return true }
func (stubGateway) Invoke(context.Context, gateway.InvokeParams) (*gateway.InvokeResult, error) {
	return nil, nil
}
func (stubGateway) ListPlatformCapabilities(context.Context, gateway.ListPlatformCapabilitiesOptions) ([]gateway.PlatformCapabilityRecord, error) {
	return nil, nil
}
func (stubGateway) ResolveGatewayTenantUUID(context.Context) (string, error) {
	return testTenantUUID, nil
}
func (stubGateway) ListAgents(context.Context, string) ([]gateway.AgentRecord, error) {
	return []gateway.AgentRecord{
		{ID: float64(1), UUID: "00000000-0000-4000-8000-000000000001", Key: "system.default", Name: "System Default Agent", Status: "active"},
		{ID: float64(2), UUID: "00000000-0000-4000-8000-000000000002", Key: "template.crud", Name: "Template CRUD Agent", Status: "active"},
	}, nil
}
func (stubGateway) GetAgent(context.Context, string) (*gateway.AgentRecord, error) { return nil, nil }
func (stubGateway) SyncPluginSkill(context.Context, gateway.PluginSkillSyncParams) (*gateway.PluginSkillSyncResult, error) {
	return nil, nil
}
func (stubGateway) SyncPluginAgent(context.Context, gateway.PluginAgentSyncParams) (*gateway.PluginAgentSyncResult, error) {
	return nil, nil
}
func (stubGateway) RegisterCatalog(context.Context, *capabilities.CatalogSnapshot, []capabilities.ProtocolAsset) error {
	return nil
}
func (stubGateway) CreateAgentSession(_ context.Context, params gateway.AgentSessionParams) (*gateway.AgentSessionRecord, error) {
	if strings.TrimSpace(params.AgentUUID) == "" && strings.TrimSpace(params.AgentID) == "" {
		return nil, io.ErrUnexpectedEOF
	}
	return &gateway.AgentSessionRecord{
		ID:        float64(101),
		UUID:      "10000000-0000-4000-8000-000000000101",
		AgentID:   float64(1),
		Title:     params.Title,
		Status:    "active",
		SessionID: "10000000-0000-4000-8000-000000000101",
	}, nil
}
func (stubGateway) ListAgentSessions(_ context.Context, opts gateway.AgentSessionListOptions) ([]gateway.AgentSessionRecord, error) {
	if strings.TrimSpace(opts.AgentID) == "" {
		return nil, io.ErrUnexpectedEOF
	}
	return []gateway.AgentSessionRecord{
		{
			ID:        float64(101),
			UUID:      "10000000-0000-4000-8000-000000000101",
			AgentID:   float64(1),
			Title:     "System Default Agent 会话",
			Status:    "active",
			SessionID: "10000000-0000-4000-8000-000000000101",
		},
	}, nil
}
func (stubGateway) ListAgentSessionMessages(_ context.Context, opts gateway.AgentSessionMessageListOptions) ([]gateway.AgentSessionMessageRecord, error) {
	if strings.TrimSpace(opts.SessionID) == "" {
		return nil, io.ErrUnexpectedEOF
	}
	return []gateway.AgentSessionMessageRecord{
		{ID: float64(1), Role: "user", Content: "hello"},
		{ID: float64(2), Role: "assistant", Content: "world"},
	}, nil
}
func (stubGateway) DeleteAgentSession(_ context.Context, opts gateway.AgentSessionMutationOptions) error {
	if strings.TrimSpace(opts.SessionID) == "" {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (stubGateway) ArchiveAgentSession(_ context.Context, opts gateway.AgentSessionMutationOptions) error {
	if strings.TrimSpace(opts.SessionID) == "" {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (stubGateway) StreamAgentSSE(_ context.Context, params gateway.AgentStreamParams) (*gateway.AgentStream, error) {
	if params.AgentID == "" || params.SessionID == "" || params.TraceID == "" || params.Query == "" {
		return nil, io.ErrUnexpectedEOF
	}
	return &gateway.AgentStream{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Trace-ID": []string{params.TraceID}},
		Body:       io.NopCloser(strings.NewReader("event: final\ndata: {\"type\":\"final\"}\n\n")),
	}, nil
}
func (stubGateway) Close() error { return nil }

func TestListSessionsUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/sessions?agent_uuid=00000000-0000-4000-8000-000000000001&status=active", nil)
	req.Header.Set("tenant_uuid", testTenantUUID)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "System Default Agent 会话")
	require.Contains(t, w.Body.String(), "10000000-0000-4000-8000-000000000101")
}

func TestListSessionMessagesUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/sessions/10000000-0000-4000-8000-000000000101/messages", nil)
	req.Header.Set("tenant_uuid", testTenantUUID)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "hello")
	require.Contains(t, w.Body.String(), "world")
}

func TestDeleteSessionUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plugin/agent/sessions/10000000-0000-4000-8000-000000000101", nil)
	req.Header.Set("tenant_uuid", testTenantUUID)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"ok":true`)
}

func TestDeleteSessionUsesGatewayCredentialTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plugin/agent/sessions/10000000-0000-4000-8000-000000000101", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"ok":true`)
}
func (g *captureGateway) StreamAgentSSE(ctx context.Context, params gateway.AgentStreamParams) (*gateway.AgentStream, error) {
	g.params = params
	return g.stubGateway.StreamAgentSSE(ctx, params)
}

func (g *captureSessionGateway) ListAgentSessions(ctx context.Context, opts gateway.AgentSessionListOptions) ([]gateway.AgentSessionRecord, error) {
	g.listOpts = opts
	return g.stubGateway.ListAgentSessions(ctx, opts)
}

func TestListSessionsDoesNotUseLocalAuthTenantForGatewayRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capture := &captureSessionGateway{}
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{
		CapabilityGateway: capture,
		Config: &cfgpkg.Config{
			Gateway: &cfgpkg.GatewayConfig{
				AuthScheme: "apikey",
				APIKey:     "token",
			},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/sessions?agent_uuid=00000000-0000-4000-8000-000000000001&status=active", nil)
	ctx := authx.ContextWithTenantUUID(req.Context(), "00000000-0000-0000-0000-000000000001")
	router.ServeHTTP(w, req.WithContext(ctx))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "", capture.listOpts.TenantUUID)
}

func TestListAgentsUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/agents?env=dev", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "System Default Agent")
	require.Contains(t, w.Body.String(), "template.crud")
}

func TestCreateSessionUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	body := strings.NewReader(`{"agent_uuid":"00000000-0000-4000-8000-000000000001","title":"System Default Agent 会话","meta":{"source":"powerxplugin.local_chat"}}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugin/agent/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	data := payload["data"].(map[string]any)
	require.Equal(t, "10000000-0000-4000-8000-000000000101", data["session_id"])
	require.NotContains(t, w.Body.String(), "session_system_default_agent")
}

func TestStreamSSEUsesPluginBackendGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: stubGateway{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/stream/sse?agent_id=00000000-0000-4000-8000-000000000001&session_id=10000000-0000-4000-8000-000000000101&trace_id=trace_test&q=hello", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, w.Body.String(), "event: final")
}

func TestStreamSSEPrefersExplicitUUIDParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &captureGateway{}
	RegisterAPIRoutes(router.Group("/api/v1"), &app.Deps{CapabilityGateway: gateway})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/stream/sse?agent_uuid=00000000-0000-4000-8000-000000000001&agent_id=legacy-agent&session_uuid=10000000-0000-4000-8000-000000000101&session_id=legacy-session&trace_id=trace_test&q=hello&intent=agent.bound_capabilities&regen_from_message_id=215", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "00000000-0000-4000-8000-000000000001", gateway.params.AgentID)
	require.Equal(t, "10000000-0000-4000-8000-000000000101", gateway.params.SessionID)
	require.Equal(t, "agent.bound_capabilities", gateway.params.Intent)
	require.Equal(t, "215", gateway.params.RegenFromMessageID)
}
