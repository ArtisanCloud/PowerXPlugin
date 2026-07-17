package runtime_ops

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type knowledgeGatewayStub struct {
	last       capgateway.InvokeParams
	lastSpaces capgateway.KnowledgeSpaceListOptions
	lastRetire capgateway.KnowledgeSpaceRetireParams
	lastDelete capgateway.KnowledgeSpaceDeleteParams
	invokeData map[string]any
	invokeRaw  json.RawMessage
	createErr  error
}

func (g *knowledgeGatewayStub) Enabled() bool { return true }
func (g *knowledgeGatewayStub) Invoke(_ context.Context, params capgateway.InvokeParams) (*capgateway.InvokeResult, error) {
	g.last = params
	data := g.invokeData
	if data == nil {
		data = map[string]any{"ok": true}
	}
	return &capgateway.InvokeResult{TraceID: "trace-knowledge", Status: "ok", Data: data, Raw: g.invokeRaw}, nil
}
func (g *knowledgeGatewayStub) ListPlatformCapabilities(context.Context, capgateway.ListPlatformCapabilitiesOptions) ([]capgateway.PlatformCapabilityRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) ListKnowledgeSpaces(_ context.Context, opts capgateway.KnowledgeSpaceListOptions) ([]capgateway.KnowledgeSpaceRuntimeRecord, error) {
	g.lastSpaces = opts
	return []capgateway.KnowledgeSpaceRuntimeRecord{
		{
			UUID:           "space-1",
			SpaceName:      "客服知识空间",
			Status:         "active",
			DepartmentCode: "support",
			RAGProfileKey:  "default",
		},
	}, nil
}
func (g *knowledgeGatewayStub) CreateKnowledgeSpace(_ context.Context, params capgateway.KnowledgeSpaceCreateParams) (*capgateway.KnowledgeSpaceRecord, error) {
	if g.createErr != nil {
		return nil, g.createErr
	}
	return &capgateway.KnowledgeSpaceRecord{
		SpaceID:                 "space-created",
		TenantUUID:              params.TenantUUID,
		SpaceName:               params.SpaceName,
		DepartmentCode:          params.DepartmentCode,
		Status:                  "pending",
		PolicyTemplateVersionID: params.PolicyTemplateVersionID,
		RAGProfileKey:           params.RAGProfileKey,
	}, nil
}
func (g *knowledgeGatewayStub) RetireKnowledgeSpace(_ context.Context, params capgateway.KnowledgeSpaceRetireParams) (*capgateway.KnowledgeSpaceRecord, error) {
	g.lastRetire = params
	return &capgateway.KnowledgeSpaceRecord{
		SpaceID:        params.SpaceID,
		TenantUUID:     params.TenantUUID,
		SpaceName:      "插件联调知识空间",
		DepartmentCode: "dev",
		Status:         "retired",
	}, nil
}
func (g *knowledgeGatewayStub) DeleteKnowledgeSpace(_ context.Context, params capgateway.KnowledgeSpaceDeleteParams) error {
	g.lastDelete = params
	return nil
}
func (g *knowledgeGatewayStub) ResolveGatewayTenantUUID(context.Context) (string, error) {
	return "", nil
}
func (g *knowledgeGatewayStub) ListAgents(context.Context, string) ([]capgateway.AgentRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) GetAgent(context.Context, string) (*capgateway.AgentRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) SyncPluginSkill(context.Context, capgateway.PluginSkillSyncParams) (*capgateway.PluginSkillSyncResult, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) SyncPluginAgent(context.Context, capgateway.PluginAgentSyncParams) (*capgateway.PluginAgentSyncResult, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) RegisterCatalog(context.Context, *capabilities.CatalogSnapshot, []capabilities.ProtocolAsset) error {
	return nil
}
func (g *knowledgeGatewayStub) CreateAgentSession(context.Context, capgateway.AgentSessionParams) (*capgateway.AgentSessionRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) ListAgentSessions(context.Context, capgateway.AgentSessionListOptions) ([]capgateway.AgentSessionRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) ListAgentSessionMessages(context.Context, capgateway.AgentSessionMessageListOptions) ([]capgateway.AgentSessionMessageRecord, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) DeleteAgentSession(context.Context, capgateway.AgentSessionMutationOptions) error {
	return nil
}
func (g *knowledgeGatewayStub) ArchiveAgentSession(context.Context, capgateway.AgentSessionMutationOptions) error {
	return nil
}
func (g *knowledgeGatewayStub) StreamAgentSSE(context.Context, capgateway.AgentStreamParams) (*capgateway.AgentStream, error) {
	return nil, nil
}
func (g *knowledgeGatewayStub) Close() error { return nil }

func TestKnowledgeHandlerProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Logging:   &config.LoggingConfig{DebugMode: true},
		Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true},
	}})
	router.GET("/provider", handler.Provider)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/provider", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestKnowledgeHandlerSpaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Logging:   &config.LoggingConfig{DebugMode: true},
		Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true},
	}})
	router.GET("/spaces", handler.Spaces)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces?tenant_uuid=tenant-a", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"spaces"`)) {
		t.Fatalf("expected spaces response, body=%s", rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(`"debug"`)) {
		t.Fatalf("spaces endpoint must not fabricate debug spaces, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerSpacesReflectsLocalProviderDocuments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Logging:   &config.LoggingConfig{DebugMode: true},
		Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true},
	}})
	router.POST("/search", handler.Search)
	router.GET("/spaces", handler.Spaces)

	body := []byte(`{"query":"refund","space_id":"space-a","tenant_uuid":"tenant-a","visibility":"tenant","fixture":{"title":"FAQ","content":"refund policy","tags":["debug"]}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/spaces?tenant_uuid=tenant-a", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("spaces status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"spaceId":"space-a"`)) {
		t.Fatalf("expected real local space from provider document, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerSpacePolicyAndIngestions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Logging:   &config.LoggingConfig{DebugMode: true},
		Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true},
	}})
	router.GET("/spaces/:spaceID/policy", handler.Policy)
	router.GET("/spaces/:spaceID/ingestions", handler.Ingestions)

	for _, path := range []string{"/spaces/debug/policy", "/spaces/debug/ingestions"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"space_id":"debug"`)) {
			t.Fatalf("path=%s expected space_id, body=%s", path, rec.Body.String())
		}
	}
}

func TestKnowledgeHandlerDelegatedPolicyUsesGatewayCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces/:spaceID/policy", handler.Policy)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces/space-1/policy", nil)
	req.Header.Set("Authorization", "Bearer browser-token")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.last.CapabilityID != knowledgeCapabilityListFusionStrategies || gateway.last.PreferredProtocol != "rest" {
		t.Fatalf("unexpected gateway params: %+v", gateway.last)
	}
	if gateway.last.Headers["Authorization"] != "" {
		t.Fatalf("policy diagnostics must use service gateway auth, not browser authorization: %+v", gateway.last.Headers)
	}
	payloadMap, ok := gateway.last.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected gateway payload map: %+v", gateway.last.Payload)
	}
	if payloadMap["method"] != http.MethodGet || payloadMap["endpoint"] != "/api/v1/admin/knowledge-spaces/space-1/fusion-strategies" {
		t.Fatalf("policy must use PowerX REST fusion strategies endpoint, payload=%+v", payloadMap)
	}
}

func TestKnowledgeHandlerDelegatedIngestionsUsesPowerXIngestionJobsCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{
		invokeData: map[string]any{
			"items": []any{
				map[string]any{
					"jobId":               "job-1",
					"status":              "completed",
					"chunkTotal":          float64(12),
					"chunkCoveragePct":    float64(100),
					"embeddingSuccessPct": float64(100),
				},
			},
		},
	}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces/:spaceID/ingestions", handler.Ingestions)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces/space-1/ingestions?limit=10", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.last.CapabilityID != knowledgeCapabilityListIngestionJobs || gateway.last.PreferredProtocol != "rest" {
		t.Fatalf("unexpected gateway params: %+v", gateway.last)
	}
	payloadMap, ok := gateway.last.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected gateway payload map: %+v", gateway.last.Payload)
	}
	if payloadMap["method"] != http.MethodGet || payloadMap["endpoint"] != "/api/v1/admin/knowledge-spaces/space-1/ingestion-jobs" {
		t.Fatalf("ingestions must use PowerX ingestion jobs endpoint, payload=%+v", payloadMap)
	}
	query, ok := payloadMap["query"].(map[string]any)
	if !ok || query["limit"] != "10" {
		t.Fatalf("expected limit query, payload=%+v", payloadMap)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"jobId":"job-1"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("expected ingestion job records, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerDelegatedIngestionsReadsArrayRawData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{
		invokeData: map[string]any{},
		invokeRaw:  json.RawMessage(`[{"jobId":"job-raw-1","status":"completed","chunkTotal":3}]`),
	}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces/:spaceID/ingestions", handler.Ingestions)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces/space-1/ingestions?limit=20", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"jobId":"job-raw-1"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("expected raw array ingestion records, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerDelegatedIngestionsReadsTenantInvocationNestedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{
		invokeData: map[string]any{
			"status": "completed",
			"result": map[string]any{
				"code": float64(200),
				"data": []any{
					map[string]any{
						"jobId":      "job-nested-1",
						"status":     "completed",
						"chunkTotal": float64(3),
					},
				},
			},
		},
	}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces/:spaceID/ingestions", handler.Ingestions)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces/space-1/ingestions?limit=20", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"jobId":"job-nested-1"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("expected nested tenant invocation records, body=%s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"completed"`)) {
		t.Fatalf("expected tenant invocation status, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerDelegatedSpacesUseFrameworkProviderGatewayAdapter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces", handler.Spaces)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces?tenant_uuid=tenant-a", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.lastSpaces.TenantUUID != "tenant-a" || gateway.lastSpaces.PageSize != 100 {
		t.Fatalf("unexpected gateway list opts: %+v", gateway.lastSpaces)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"spaceName":"客服知识空间"`)) {
		t.Fatalf("expected delegated runtime space, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerSpacesRejectsTenantMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.GET("/spaces", handler.Spaces)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/spaces?tenant_uuid=tenant-requested", nil)
	req = req.WithContext(authmw.ContextWithTenantUUID(req.Context(), "tenant-authenticated"))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.lastSpaces.TenantUUID != "" {
		t.Fatalf("gateway should not be called on tenant mismatch, got %+v", gateway.lastSpaces)
	}
}

func TestKnowledgeHandlerCreateSpaceConflictUsesKnowledgeConflictCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{createErr: &capgateway.PlatformAPIError{
		Operation:  "platform knowledge space create",
		StatusCode: http.StatusConflict,
		Code:       http.StatusConflict,
		Message:    "同一租户下已存在该空间",
	}}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.POST("/spaces", handler.CreateSpace)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/spaces?tenant_uuid=tenant-a", bytes.NewReader([]byte(`{
		"spaceName":"插件联调知识空间",
		"departmentCode":"dev",
		"policyTemplateVersionId":"default-v1"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":"KNOWLEDGE_CONFLICT"`)) {
		t.Fatalf("expected conflict code, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerRetireSpaceUsesGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.POST("/spaces/:spaceID/retire", handler.RetireSpace)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/spaces/space-1/retire?tenant_uuid=tenant-a", bytes.NewReader([]byte(`{
		"reason":"debug cleanup",
		"requestedBy":"tester"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.lastRetire.SpaceID != "space-1" || gateway.lastRetire.TenantUUID != "tenant-a" || gateway.lastRetire.RequestedBy != "tester" {
		t.Fatalf("unexpected retire params: %+v", gateway.lastRetire)
	}
}

func TestKnowledgeHandlerDeleteSpaceUsesGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.DELETE("/spaces/:spaceID", handler.DeleteSpace)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/spaces/space-1?tenant_uuid=tenant-a", bytes.NewReader([]byte(`{
		"requestedBy":"tester",
		"dropVectors":true
	}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.lastDelete.SpaceID != "space-1" || gateway.lastDelete.TenantUUID != "tenant-a" || gateway.lastDelete.RequestedBy != "tester" || !gateway.lastDelete.DropVectors {
		t.Fatalf("unexpected delete params: %+v", gateway.lastDelete)
	}
}

func TestKnowledgeHandlerDelegatedSearchUsesQABridgeCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.POST("/search", handler.Search)

	payload := []byte(`{"query":"refund","space_id":"debug","tenant_uuid":"tenant-a","visibility":"tenant","filters":{"source":"faq"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.last.CapabilityID != knowledgeCapabilityPlanRetrieval || gateway.last.PreferredProtocol != "rest" {
		t.Fatalf("unexpected gateway params: %+v", gateway.last)
	}
	if !gateway.last.AuthRequired {
		t.Fatalf("knowledge delegated search must require gateway service auth: %+v", gateway.last)
	}
	payloadMap, ok := gateway.last.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected gateway payload map: %+v", gateway.last.Payload)
	}
	if payloadMap["method"] != http.MethodPost || payloadMap["endpoint"] != "/api/v1/admin/knowledge-spaces/debug/playground/retrieval" {
		t.Fatalf("search must use PowerX REST playground retrieval endpoint, payload=%+v", payloadMap)
	}
	body, ok := payloadMap["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected gateway body payload: %+v", payloadMap)
	}
	if body["query"] != "refund" || body["topK"] == nil {
		t.Fatalf("unexpected retrieval body: %+v", body)
	}
}

func TestKnowledgeHandlerDelegatedIngestSpaceUsesPowerXIngestionJobsCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.POST("/spaces/:spaceID/ingest", handler.IngestSpace)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/spaces/space-1/ingest", bytes.NewReader([]byte(`{
		"format":"markdown",
		"sourceUri":"https://media.powerx.example/download/space-1/fixture.md",
		"ingestionProfile":"p1_general",
		"requestedBy":"tester"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.last.CapabilityID != knowledgeCapabilityCreateIngestionJob || gateway.last.CapabilityID == knowledgeCapabilityPlanRetrieval || gateway.last.PreferredProtocol != "rest" {
		t.Fatalf("unexpected gateway params: %+v", gateway.last)
	}
	if !gateway.last.AuthRequired {
		t.Fatalf("ingest gateway invocation must require Authorization")
	}
	payloadMap, ok := gateway.last.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected gateway payload map: %+v", gateway.last.Payload)
	}
	if payloadMap["method"] != http.MethodPost || payloadMap["endpoint"] != "/api/v1/admin/knowledge-spaces/space-1/ingestion-jobs" {
		t.Fatalf("ingest must use PowerX ingestion jobs endpoint, payload=%+v", payloadMap)
	}
	body, ok := payloadMap["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected gateway body payload: %+v", payloadMap)
	}
	if _, ok := body["spaceId"]; ok {
		t.Fatalf("space id must stay in REST path, not body: %+v", body)
	}
	if body["sourceUri"] != "https://media.powerx.example/download/space-1/fixture.md" || body["requestedBy"] != "tester" {
		t.Fatalf("unexpected ingestion body: %+v", body)
	}
}

func TestKnowledgeHandlerDelegatedIngestSpaceRejectsMissingSourceURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gateway := &knowledgeGatewayStub{}
	handler := NewKnowledgeHandler(&app.Deps{
		Config: &config.Config{
			Logging:   &config.LoggingConfig{DebugMode: false},
			Knowledge: &config.KnowledgeConfig{Mode: "delegated", RequireTenant: true, DelegateTimeout: "1s"},
		},
		CapabilityGateway: gateway,
	})
	router.POST("/spaces/:spaceID/ingest", handler.IngestSpace)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/spaces/space-1/ingest", bytes.NewReader([]byte(`{
		"format":"markdown",
		"requestedBy":"tester"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gateway.last.CapabilityID != "" {
		t.Fatalf("gateway should not be invoked without sourceUri, got %+v", gateway.last)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`sourceUri is required`)) {
		t.Fatalf("expected explicit sourceUri error, body=%s", rec.Body.String())
	}
}

func TestKnowledgeHandlerResolveMediaURL(t *testing.T) {
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Gateway: &config.GatewayConfig{BaseURL: "http://127.0.0.1:8077", APIPrefix: "/api/v1"},
	}})

	cases := map[string]string{
		"https://media.example/object": "https://media.example/object",
		"/api/v1/media/assets/asset-1": "http://127.0.0.1:8077/api/v1/media/assets/asset-1",
		"/media/asset-1/resource":      "http://127.0.0.1:8077/media/asset-1/resource",
		"media/asset-1/resource":       "http://127.0.0.1:8077/media/asset-1/resource",
	}
	for input, want := range cases {
		if got := handler.resolveMediaURL(input); got != want {
			t.Fatalf("resolveMediaURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestKnowledgeGatewayDelegatedClientRejectsDocumentWithoutURI(t *testing.T) {
	gateway := &knowledgeGatewayStub{}
	client := knowledgeGatewayDelegatedClient{gateway: gateway}

	_, err := client.UpsertKnowledgeDocument(context.Background(), fwknowledge.KnowledgeDocument{
		SpaceID:     "space-1",
		DocumentID:  "doc-1",
		ContentType: "markdown",
	})
	if err == nil {
		t.Fatal("expected missing sourceUri error")
	}
	if gateway.last.CapabilityID != "" {
		t.Fatalf("gateway should not be invoked without document URI, got %+v", gateway.last)
	}
	if !strings.Contains(err.Error(), "sourceUri is required") {
		t.Fatalf("expected explicit sourceUri error, got %v", err)
	}
}

func TestKnowledgeHandlerSearchWithFixture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewKnowledgeHandler(&app.Deps{Config: &config.Config{
		Logging:   &config.LoggingConfig{DebugMode: true},
		Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true},
	}})
	router.POST("/search", handler.Search)

	payload := []byte(`{"query":"refund","space_id":"debug","tenant_uuid":"tenant-a","visibility":"tenant","fixture":{"title":"FAQ","content":"refund policy"}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("expected one search result, body=%s", rec.Body.String())
	}
}
