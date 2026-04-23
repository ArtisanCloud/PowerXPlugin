package iam

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	basemodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	integrationmodel "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/integration"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
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

func TestRegisterRoutes_LarkSyncTaskRoutes_TriggerListClear(t *testing.T) {
	gin.SetMode(gin.TestMode)
	basemodel.ForceSchemaForTests("")
	t.Cleanup(func() { basemodel.ForceSchemaForTests("public") })
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(httpmw.TenantUUIDContextKey, "00000000-0000-0000-0000-000000000001")
		c.Next()
	})
	admin := engine.Group("/admin")

	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: "file:iam-lark-sync-routes-test?mode=memory&cache=shared"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE integration_secrets (
id TEXT PRIMARY KEY,
tenant_uuid TEXT NOT NULL,
integration_type TEXT NOT NULL,
current_secret_ref TEXT,
pending_secret_ref TEXT,
rotation_interval_days INTEGER NOT NULL DEFAULT 30,
last_rotated_at DATETIME,
next_rotation_due_at DATETIME,
status TEXT NOT NULL DEFAULT 'ACTIVE',
audit_log JSON,
metadata JSON,
created_at DATETIME,
updated_at DATETIME
)`).Error; err != nil {
		t.Fatalf("create integration_secrets failed: %v", err)
	}
	if err := db.Exec(`CREATE TABLE iam_channel_sync_tasks (
id TEXT PRIMARY KEY,
tenant_uuid TEXT NOT NULL,
provider TEXT NOT NULL,
action TEXT NOT NULL,
status TEXT NOT NULL,
summary TEXT,
error_message TEXT,
request_payload JSON,
result_payload JSON,
started_at DATETIME,
finished_at DATETIME,
created_at DATETIME,
updated_at DATETIME
)`).Error; err != nil {
		t.Fatalf("create iam_channel_sync_tasks failed: %v", err)
	}
	if err := db.Create(&integrationmodel.SecretCredential{
		ID:               "secret-lark-routes",
		TenantUuid:       "00000000-0000-0000-0000-000000000001",
		IntegrationType:  "iam_federated_lark",
		RotationInterval: 30,
		Status:           integrationmodel.SecretStatusActive,
		Metadata: datatypes.JSONMap{
			"tenant_key":    "tenant_key_001",
			"app_id":        "cli_lark_app_001",
			"app_secret":    "secret_lark_001",
			"callback_host": "https://debug.artisan-cloud.com",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("seed lark config failed: %v", err)
	}

	deps := &app.Deps{DB: db}
	RegisterRoutes(admin, deps)

	triggerReq := httptest.NewRequest(http.MethodPost, "/admin/iam/channels/lark/sync-tasks", bytes.NewBufferString(`{"action":"full_sync"}`))
	triggerReq.Header.Set("Content-Type", "application/json")
	triggerRes := httptest.NewRecorder()
	engine.ServeHTTP(triggerRes, triggerReq)
	if triggerRes.Code != http.StatusOK {
		t.Fatalf("trigger status=%d body=%s", triggerRes.Code, triggerRes.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/iam/channels/lark/sync-tasks?limit=20", nil)
	listRes := httptest.NewRecorder()
	engine.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRes.Code, listRes.Body.String())
	}
	var listResp contracts.APIResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response err=%v", err)
	}
	listData, ok := listResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected list data: %#v", listResp.Data)
	}
	items, ok := listData["items"].([]any)
	if !ok {
		t.Fatalf("unexpected list items: %#v", listData["items"])
	}
	if len(items) == 0 {
		t.Fatalf("expected sync task item after trigger")
	}

	clearReq := httptest.NewRequest(http.MethodDelete, "/admin/iam/channels/lark/sync-tasks", nil)
	clearRes := httptest.NewRecorder()
	engine.ServeHTTP(clearRes, clearReq)
	if clearRes.Code != http.StatusOK {
		t.Fatalf("clear status=%d body=%s", clearRes.Code, clearRes.Body.String())
	}
}
