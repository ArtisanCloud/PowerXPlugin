package skills

import (
	"net/http"
	"net/http/httptest"
	"testing"

	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	entmodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSkillRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	entmodels.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dbtemplate.Template{}))
	return db
}

func TestSkillsListAndSchemaRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterAPIRoutes(api, &app.Deps{DB: newSkillRouteTestDB(t)})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/skills", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "powerxplugin.template.basic")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/plugin/skills/powerxplugin.template.basic/schema", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "input_schema")
}
