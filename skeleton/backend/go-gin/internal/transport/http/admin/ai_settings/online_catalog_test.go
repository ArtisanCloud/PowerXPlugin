package ai_settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/ai_settings"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	local "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/ai_settings"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLocalOnlineCatalogRealRouteAuthorizationStrictBodyAndNoCoreInvocation(t *testing.T) {
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		require.Equal(t, "GET /api/tags", r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()
	models.InitSchemaFrom("")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	defer sql.Close()
	require.NoError(t, db.AutoMigrate(&models.LocalAISetting{}, &models.LocalAIOnlineCatalog{}))
	repository, err := repo.NewLocalSettingsRepository(db)
	require.NoError(t, err)
	ai, err := runtimeexample.NewLocalAI(nil, nil)
	require.NoError(t, err)
	service, err := local.NewLocalSettingsService(repository, nil, ai)
	require.NoError(t, err)
	service.Catalog, err = local.LoadCatalog(filepath.Join("..", "..", "..", "..", "..", "..", "etc", "ai", "providers.d"))
	require.NoError(t, err)
	tenant := "11111111-1111-4111-8111-111111111111"
	root := false
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenant, IsRoot: root})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenant))
		c.Next()
	})
	RegisterRoutes(router.Group("/api/v1/admin"), &app.Deps{LocalAISettings: service})
	input := map[string]any{"environment": "development", "modality": "llm", "provider": "ollama", "app": "", "endpoint": server.URL, "credential_ref": "", "action": "refresh"}
	call := func(body map[string]any) *httptest.ResponseRecorder {
		raw, e := json.Marshal(body)
		require.NoError(t, e)
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai-settings/catalog/local-online", strings.NewReader(string(raw)))
		r.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, r)
		return rec
	}
	rec := call(input)
	require.Equal(t, 403, rec.Code)
	require.Zero(t, reads.Load())
	root = true
	for _, extra := range []string{"api_key", "tenant_uuid", "source", "headers", "prompt", "contract_version"} {
		bad := map[string]any{}
		for k, v := range input {
			bad[k] = v
		}
		bad[extra] = "fixture.forbidden"
		rec = call(bad)
		require.Equal(t, 400, rec.Code, rec.Body.String())
		require.Zero(t, reads.Load())
	}
	rec = call(input)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.EqualValues(t, 1, reads.Load())
	require.Contains(t, rec.Body.String(), `"models":[]`)
	var count int64
	require.NoError(t, db.Model(&models.LocalAISetting{}).Count(&count).Error)
	require.Zero(t, count)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/ai-settings/catalog/models?source=local&environment=development&modality=llm&provider=ollama&endpoint="+server.URL, nil).WithContext(context.Background())
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, request)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"items":[]`)
	require.EqualValues(t, 1, reads.Load())
}
