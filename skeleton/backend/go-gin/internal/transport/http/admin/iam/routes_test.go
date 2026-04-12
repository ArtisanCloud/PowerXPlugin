package iam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes_ModeEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	admin := engine.Group("/admin")

	deps := &app.Deps{
		IAMMode:       iamservice.IAMModeDelegated,
		IAMModeSource: "config",
	}
	RegisterRoutes(admin, deps)

	req := httptest.NewRequest(http.MethodGet, "/admin/iam/mode", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch, got=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp contracts.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response")
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected response data: %#v", resp.Data)
	}
	if got, _ := data["mode"].(string); got != "delegated" {
		t.Fatalf("mode mismatch, got=%s", got)
	}
	if got, _ := data["source"].(string); got != "config" {
		t.Fatalf("source mismatch, got=%s", got)
	}
}
