package iam

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
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

func TestRegisterRoutes_ChannelFederatedConfigRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	admin := engine.Group("/admin")

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam-routes-test?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	deps := &app.Deps{DB: db}
	RegisterRoutes(admin, deps)

	type tc struct {
		method string
		path   string
		body   string
	}
	cases := []tc{
		{method: http.MethodGet, path: "/admin/iam/channels/dingtalk/config"},
		{method: http.MethodPut, path: "/admin/iam/channels/dingtalk/config", body: `{"corp_id":"c","app_key":"k","app_secret":"s"}`},
		{method: http.MethodGet, path: "/admin/iam/channels/dingtalk/sync-tasks"},
		{method: http.MethodPost, path: "/admin/iam/channels/dingtalk/sync-tasks", body: `{"action":"full_sync"}`},
		{method: http.MethodDelete, path: "/admin/iam/channels/dingtalk/sync-tasks"},
		{method: http.MethodGet, path: "/admin/iam/channels/lark/config"},
		{method: http.MethodPut, path: "/admin/iam/channels/lark/config", body: `{"tenant_key":"t","app_id":"a","app_secret":"s"}`},
		{method: http.MethodGet, path: "/admin/iam/channels/lark/sync-tasks"},
		{method: http.MethodPost, path: "/admin/iam/channels/lark/sync-tasks", body: `{"action":"full_sync"}`},
		{method: http.MethodDelete, path: "/admin/iam/channels/lark/sync-tasks"},
	}
	for _, item := range cases {
		var req *http.Request
		if item.body == "" {
			req = httptest.NewRequest(item.method, item.path, nil)
		} else {
			req = httptest.NewRequest(item.method, item.path, bytes.NewBufferString(item.body))
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Fatalf("route missing: %s %s", item.method, item.path)
		}
	}
}
