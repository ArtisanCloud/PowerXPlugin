package skills

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHTTPAdapterListSchemaAndInvoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reg := NewRegistry()
	require.NoError(t, reg.RegisterManifest(validManifest()))
	router := gin.New()
	NewHTTPAdapter(reg).RegisterRoutes(router.Group("/api/v1/plugin/skills"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/skills", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "mediax.video_rebuilder.cn")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/plugin/skills/mediax.video_rebuilder.cn/schema", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "input_schema")
}
