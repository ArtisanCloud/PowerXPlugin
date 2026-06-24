package agent_registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsePowerXSyncErrorMapsAdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/sync", func(c *gin.Context) {
		responsePowerXSyncError(c, "POWERX_SKILL_SYNC_FAILED", errors.New("forbidden: admin only"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)

	var payload struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.False(t, payload.Success)
	require.Equal(t, "POWERX_ADMIN_REQUIRED", payload.Error.Code)
	require.Contains(t, payload.Error.Message, "Gateway 凭证")
	require.Equal(t, "forbidden: admin only", payload.Error.Details["cause"])
}

func TestResponsePowerXSyncErrorKeepsFallbackForOtherErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/sync", func(c *gin.Context) {
		responsePowerXSyncError(c, "POWERX_AGENT_SYNC_FAILED", errors.New("upstream timeout"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/sync", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "POWERX_AGENT_SYNC_FAILED")
	require.Contains(t, w.Body.String(), "upstream timeout")
}
