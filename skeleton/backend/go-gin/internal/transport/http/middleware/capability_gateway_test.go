package middleware_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type enabledGateway struct{}

func (enabledGateway) Enabled() bool { return true }
func (enabledGateway) Invoke(_ context.Context, _ capgateway.InvokeParams) (*capgateway.InvokeResult, error) {
	return &capgateway.InvokeResult{}, nil
}
func (enabledGateway) ListPlatformCapabilities(_ context.Context, _ capgateway.ListPlatformCapabilitiesOptions) ([]capgateway.PlatformCapabilityRecord, error) {
	return nil, nil
}
func (enabledGateway) Close() error { return nil }

func TestRequireCapabilityGatewayDelegatedConfigMissing(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/invoke", func(c *gin.Context) {
		deps := &app.Deps{
			IAMMode: iamservice.IAMModeDelegated,
			Config: &config.Config{
				Context: &config.ContextConfig{IAMMode: "delegated"},
				Gateway: &config.GatewayConfig{
					BaseURL:    "https://gateway.example.com",
					AuthScheme: "bearer",
				},
			},
		}
		if !httpmw.RequireCapabilityGateway(c, deps) {
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/invoke", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, capgateway.ErrCodeGatewayMissingSTSClient, errObj["code"])
}

func TestRequireCapabilityGatewayDelegatedUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/invoke", func(c *gin.Context) {
		deps := &app.Deps{
			IAMMode: iamservice.IAMModeDelegated,
			Config: &config.Config{
				Context: &config.ContextConfig{IAMMode: "delegated"},
				Gateway: &config.GatewayConfig{
					BaseURL:    "https://gateway.example.com",
					AuthScheme: "bearer",
					ToolToken:  delegatedToken(),
				},
			},
		}
		if !httpmw.RequireCapabilityGateway(c, deps) {
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/invoke", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "SERVICE_UNAVAILABLE", errObj["code"])
}

func delegatedToken() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"tid":"8d8e338c-cf38-4b82-b7a3-bfea18b41189"}`))
	return header + "." + payload + ".sig"
}
