package runtime_ops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeLabLocalProxyUsesCoreWithoutChangingLocalProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
		if r.URL.Path == "/api/v1/tenant/knowledge/catalog" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"reason_code":"KNOWLEDGE_FORBIDDEN"}`))
			return
		}
		require.Equal(t, "/api/v1/tenant/knowledge/spaces", r.URL.Path)
		require.Empty(t, r.Header.Get("X-Tenant-UUID"))
		require.Empty(t, r.URL.Query().Get("tenant_uuid"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"items":[{"space_uuid":"11111111-1111-4111-8111-111111111111","name":"Core space","status":"active"}]}}`))
	}))
	defer core.Close()
	local := fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{})
	deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, KnowledgeProvider: local, CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "server-key"}}}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
	req := httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/spaces", nil)
	req.Header.Set("Authorization", "Bearer plugin-local-user-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var response struct {
		Data struct {
			Source string `json:"source"`
			Spaces []struct {
				ID string `json:"id"`
			} `json:"spaces"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "delegated", response.Data.Source)
	require.Len(t, response.Data.Spaces, 1)
	require.Equal(t, "11111111-1111-4111-8111-111111111111", response.Data.Spaces[0].ID)
	require.Equal(t, 1, calls)
	require.Same(t, local, deps.KnowledgeProvider)
	require.Equal(t, fwprovider.ModeLocal, deps.ProviderMode)
	// A denied Core catalog fails independently, never substituting the Local catalog.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/catalog", nil))
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), string(fwknowledge.CodeForbidden))
	require.Equal(t, 2, calls)
}

func TestKnowledgeLabFailsClosedWithoutServerCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, KnowledgeProvider: fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{}), CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: "http://127.0.0.1:1", AuthScheme: "apikey"}}}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
	for _, path := range []string{"provider", "spaces", "spaces/11111111-1111-4111-8111-111111111111/ingestions", "spaces/11111111-1111-4111-8111-111111111111/policy"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/"+path, nil))
		require.Equal(t, http.StatusServiceUnavailable, rec.Code, path)
		require.Contains(t, rec.Body.String(), "knowledgeLab.proxyUnavailable")
	}
}

func TestKnowledgeLabPolicyUsesStrictTenantEnvelopeAndPreservesDenial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
		var body struct {
			CapabilityID      string         `json:"capability_id"`
			PreferredProtocol string         `json:"preferred_protocol"`
			Payload           map[string]any `json:"payload"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		require.NoError(t, decoder.Decode(&body))
		require.Equal(t, knowledgeCapabilityListFusionStrategies, body.CapabilityID)
		require.Equal(t, "/api/v1/admin/knowledge-spaces/11111111-1111-4111-8111-111111111111/fusion-strategies", body.Payload["endpoint"])
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-Id", "test-denied-trace")
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"trace_id":"test-denied-trace","errors":[{"code":"registry.capability_forbidden","message":"denied"}]}`)
	}))
	defer core.Close()
	cfg := &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, APIPrefix: "/api/v1", AuthScheme: "apikey", APIKey: "server-key"}}
	deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, Config: cfg, CapabilityGateway: capgateway.NewClient(cfg, nil)}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/spaces/11111111-1111-4111-8111-111111111111/policy", nil))
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "test-denied-trace")
	require.Contains(t, rec.Body.String(), string(fwknowledge.CodeForbidden))
	require.Equal(t, 1, calls)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/spaces/not-a-uuid/policy", nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, 1, calls)
}

func TestKnowledgeLabDepartmentsUseHostUUIDsAndPreserveDenial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name     string
		status   int
		body     string
		expected int
	}{
		{"success", 200, `{"data":{"items":[{"department_uuid":"11111111-1111-4111-8111-111111111111","tenant_uuid":"22222222-2222-4222-8222-222222222222","name":"fixture"}]}}`, 200},
		{"denied", 403, `{"error":{"reason_code":"IAM_FORBIDDEN"}}`, 403},
		{"missing-uuid", 200, `{"data":{"items":[{"tenant_uuid":"22222222-2222-4222-8222-222222222222","name":"fixture"}]}}`, 502},
		{"mixed-tenant", 200, `{"data":{"items":[{"department_uuid":"11111111-1111-4111-8111-111111111111","tenant_uuid":"22222222-2222-4222-8222-222222222222","name":"fixture"},{"department_uuid":"33333333-3333-4333-8333-333333333333","tenant_uuid":"44444444-4444-4444-8444-444444444444","name":"fixture"}]}}`, 502},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/api/v1/tenant/iam/departments", r.URL.Path)
				require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
				require.Empty(t, r.URL.RawQuery)
				require.Empty(t, r.Header.Get("tenant_uuid"))
				w.Header().Set("X-Trace-Id", "directory-test-trace")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer core.Close()
			deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, KnowledgeProvider: fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{}), CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "server-key"}}}
			router := gin.New()
			registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/departments", nil)
			req.Header.Set("Authorization", "Bearer browser-token")
			req.Header.Set("tenant_uuid", "local-tenant")
			router.ServeHTTP(rec, req)
			require.Equal(t, test.expected, rec.Code, rec.Body.String())
			require.Equal(t, 1, calls)
			if test.status == 403 {
				require.Contains(t, rec.Body.String(), "IAM_FORBIDDEN")
				require.Contains(t, rec.Body.String(), "directory-test-trace")
			}
			if test.expected == 200 {
				require.Contains(t, rec.Body.String(), "department_uuid")
				require.Contains(t, rec.Body.String(), "fixture")
			}
			rec = httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/departments?tenant_uuid=other", nil))
			require.Equal(t, 400, rec.Code)
			require.Equal(t, 1, calls)
		})
	}
}

func TestKnowledgeLabTypedCreateAndStrictRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	calls := 0
	success := false
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/api/v1/tenant/knowledge/spaces", r.URL.Path)
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "11111111-1111-4111-8111-111111111111", body["department_uuid"])
		require.NotContains(t, body, "tenant_uuid")
		if success {
			profile := &fwknowledge.ProfileRef{UUID: "11111111-1111-4111-8111-111111111111", Key: "p1_general", Version: 1}
			item := fwknowledge.CreatedSpace{SpaceUUID: "22222222-2222-4222-8222-222222222222", Name: "fixture", Status: "pending_iam", DepartmentUUID: "11111111-1111-4111-8111-111111111111", StrategyKey: "A_simple", SceneKey: "support_faq", PolicyTemplateUUID: "11111111-1111-4111-8111-111111111111", Profiles: fwknowledge.ProfileMapping{Ingestion: profile, Index: profile, RAG: profile}, Quotas: fwknowledge.SpaceQuotas{CPUCores: 4, StorageGB: 200, IngestionConcurrency: 2}}
			require.NoError(t, json.NewEncoder(w).Encode(gin.H{"data": gin.H{"item": item}}))
			return
		}
		w.Header().Set("X-Trace-Id", "create-trace")
		w.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprint(w, `{"reason_code":"KNOWLEDGE_STRATEGY_UNAVAILABLE"}`)
	}))
	defer core.Close()
	deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "server-key"}}}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
	body := `{"name":"fixture","department_uuid":"11111111-1111-4111-8111-111111111111","strategy_key":"A_simple"}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/runtime/knowledge-lab/spaces", strings.NewReader(body)))
	require.Equal(t, http.StatusPreconditionFailed, rec.Code)
	require.Contains(t, rec.Body.String(), "KNOWLEDGE_STRATEGY_UNAVAILABLE")
	require.Contains(t, rec.Body.String(), "create-trace")
	require.Equal(t, 1, calls)
	for _, invalid := range []string{`{"tenant_uuid":"override"}`, `{"name":"fixture","departmentCode":"local"}`, body + ` {}`} {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/runtime/knowledge-lab/spaces", strings.NewReader(invalid)))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, 1, calls)
	}
	success = true
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/runtime/knowledge-lab/spaces", strings.NewReader(body)))
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), "pending_iam")
	require.Contains(t, rec.Body.String(), "22222222-2222-4222-8222-222222222222")
	require.Equal(t, 2, calls)

}

func TestKnowledgeLabDocumentUsesTypedCoreWithoutLocalWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const space = "11111111-1111-4111-8111-111111111111"
	calls := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
		require.Equal(t, "/api/v1/tenant/knowledge/spaces/"+space+"/documents", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "fixture", body["content"])
		require.Equal(t, "text/plain", body["content_type"])
		require.Len(t, body["checksum"], 64)
		require.NotContains(t, body, "tenant_uuid")
		require.NotContains(t, body, "ingestionProfile")
		require.NotContains(t, body, "segmentMode")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"job_uuid":"22222222-2222-4222-8222-222222222222","document_uuid":"33333333-3333-4333-8333-333333333333","status":"queued","operation":"upsert"}}`))
	}))
	defer core.Close()
	deps := &app.Deps{ProviderMode: fwprovider.ModeLocal, CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "server-key"}}}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/admin/runtime"), deps)
	for _, body := range []string{
		`{"title":"fixture","content":"fixture","content_type":"text/plain","tags":[]}`,
		`{"title":"fixture","content":"fixture","content_type":"text/plain","tenant_uuid":"override"}`,
		`{"title":"fixture","content":"fixture","content_type":"text/plain","ingestion":{"chunk_size":800}}`,
		`{"title":"fixture","content":"fixture","content_type":"text/html"}`,
	} {
		out := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/admin/runtime/knowledge-lab/spaces/"+space+"/documents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(out, req)
		if calls == 1 && !strings.Contains(body, "override") && !strings.Contains(body, "ingestion") && !strings.Contains(body, "html") {
			require.Equal(t, http.StatusAccepted, out.Code)
			require.Contains(t, out.Body.String(), `"status":"queued"`)
		} else {
			require.Equal(t, http.StatusBadRequest, out.Code)
		}
	}
	require.Equal(t, 1, calls)
}
