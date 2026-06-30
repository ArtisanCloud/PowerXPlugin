package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	miniapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/mini-app"
	"github.com/gin-gonic/gin"
)

func TestMiniAppCustomerValidateDelegateUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	cfg := &config.Config{
		Server:  &config.ServerConfig{APIPrefix: "/api/v1"},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:             "delegate",
			DelegateEndpoint: "http://127.0.0.1:1",
			DelegateTimeout:  "20ms",
		},
	}
	miniapp.RegisterAPIRoutes(engine.Group("/api/v1"), &app.Deps{Config: cfg})
	body, _ := json.Marshal(map[string]any{"token": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mini-app/auth/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}
