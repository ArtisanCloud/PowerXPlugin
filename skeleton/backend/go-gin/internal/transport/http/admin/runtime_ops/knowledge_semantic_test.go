package runtime_ops

import (
	"encoding/json"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	provider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKnowledgeSemanticLabPreservesFiltersAndCoreErrors(t *testing.T) {
	const space = "11111111-1111-4111-8111-111111111111"
	calls := 0
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "ApiKey server-key", r.Header.Get("Authorization"))
		require.Equal(t, "/api/v1/tenant/knowledge/retrieval/query", r.URL.Path)
		require.Empty(t, r.Header.Get("X-Tenant-UUID"))
		var query fw.SemanticQuery
		require.NoError(t, json.NewDecoder(r.Body).Decode(&query))
		require.Equal(t, " 完整需求😀 ", query.Query)
		require.Equal(t, []string{"case_category"}, query.Filters.CategoryCodes)
		w.Header().Set("X-Trace-ID", "core-trace")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"reason_code":"KNOWLEDGE_FORBIDDEN","request_id":"core-request","details":{"controlled":"fixture"}}`))
	}))
	defer core.Close()
	deps := &app.Deps{ProviderMode: provider.ModeLocal, KnowledgeProvider: fw.NewLocalProvider(fw.LocalProviderConfig{}), CapabilityGateway: &knowledgeGatewayStub{}, Config: &config.Config{Gateway: &config.GatewayConfig{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "server-key"}}}
	router := gin.New()
	registerKnowledgeLabRoutes(router.Group("/runtime"), deps)
	for _, body := range []string{`{"schema":"powerx.knowledge.retrieval-query/v1","query":"fixture","space_uuids":["` + space + `"],"mode":"semantic","top_k":1,"filters":{"sql":"1=1"}}`, `{"tenant_uuid":"fixture"}`} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/runtime/knowledge-lab/retrieval/query", strings.NewReader(body)))
		require.Equal(t, 400, rec.Code)
	}
	require.Zero(t, calls)
	body := `{"schema":"powerx.knowledge.retrieval-query/v1","query":" 完整需求😀 ","space_uuids":["` + space + `"],"mode":"hybrid","top_k":10,"filters":{"document_uuids":[],"category_codes":["case_category"],"tag_uuids":[],"artifact_roles":["knowledge_profile","source_chunk"]}}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/runtime/knowledge-lab/retrieval/query", strings.NewReader(body)))
	require.Equal(t, 403, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "core-trace")
	require.Contains(t, rec.Body.String(), "core-request")
	require.Equal(t, 1, calls)
	require.Equal(t, provider.ModeLocal, deps.ProviderMode)
}
