package localagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/localagent"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLocalRoutesStrictPayloadAndTenant(t *testing.T) {
	models.InitSchemaFrom("")
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.Agent{}, &model.Skill{}); e != nil {
		t.Fatal(e)
	}
	tenant, other := uuid.NewString(), uuid.NewString()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Fixture simulates authentication middleware, not a production tenant override.
		if c.GetHeader("Test-Tenant") != "" {
			c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(context.Background(), c.GetHeader("Test-Tenant")))
		}
	})
	RegisterRoutes(router.Group("/admin"), &app.Deps{DB: db})
	request := func(method, path, body, scope string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Test-Tenant", scope)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	valid := `{"key":"fixture","name":"fixture","status":"inactive","version":"1.0.0","executor":"template"}`
	out := request(http.MethodPost, "/admin/local-intelligence/skills", valid, tenant)
	if out.Code != 200 {
		t.Fatalf("create: %d %s", out.Code, out.Body.String())
	}
	var wire struct {
		Data struct {
			UUID string `json:"uuid"`
		} `json:"data"`
	}
	if e = json.Unmarshal(out.Body.Bytes(), &wire); e != nil || wire.Data.UUID == "" {
		t.Fatal(out.Body.String(), e)
	}
	out = request(http.MethodPut, "/admin/local-intelligence/skills/"+wire.Data.UUID, valid, other)
	if out.Code != 404 {
		t.Fatalf("cross tenant: %d", out.Code)
	}
	out = request(http.MethodGet, "/admin/local-intelligence/skills", "", "")
	if out.Code != 401 {
		t.Fatalf("missing tenant: %d", out.Code)
	}
	out = request(http.MethodPost, "/admin/local-intelligence/skills", strings.TrimSuffix(valid, "}")+`,"tenant_uuid":"`+other+`"}`, tenant)
	if out.Code != 400 {
		t.Fatalf("tenant override: %d", out.Code)
	}
	out = request(http.MethodPost, "/admin/local-intelligence/skills", strings.TrimSuffix(valid, "}")+`,"endpoint":"https://example.com"}`, tenant)
	if out.Code != 400 {
		t.Fatalf("raw endpoint: %d", out.Code)
	}
	out = request(http.MethodGet, "/admin/local-intelligence/models", "", tenant)
	if out.Code != 503 {
		t.Fatalf("no local model must fail explicitly: %d", out.Code)
	}
	out = request(http.MethodPost, "/admin/local-intelligence/skills/"+wire.Data.UUID+"/invoke", `{"input":{"action":"list"}}`, tenant)
	if out.Code != 409 {
		t.Fatalf("inactive: %d", out.Code)
	}
}
