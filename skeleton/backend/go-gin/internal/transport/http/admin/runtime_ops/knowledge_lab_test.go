package runtime_ops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		require.Equal(t, "/api/v1/tenant/knowledge/spaces", r.URL.Path)
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
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
	// Missing catalog must fail independently, never substitute the local catalog.
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/runtime/knowledge-lab/catalog", nil))
	require.NotEqual(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), string(fwknowledge.CodeUnsupportedCapability))
	require.Equal(t, 1, calls)
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
