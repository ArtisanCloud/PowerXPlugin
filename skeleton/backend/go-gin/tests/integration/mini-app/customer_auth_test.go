package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/customer"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/mini-app"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func TestMiniAppCustomerAuth_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppAuthRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("tenant_uuid", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "UNAUTHORIZED")
	_ = deps
}

func TestMiniAppCustomerAuth_TenantMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppAuthRouter(t)

	reqTenant := "00000000-0000-0000-0000-000000000001"
	tokenTenant := "00000000-0000-0000-0000-000000000099"
	customerUUID := "00000000-0000-0000-0000-000000000002"
	token := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, tokenTenant, customerUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("tenant_uuid", reqTenant)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "TENANT_MISMATCH")
}

func TestMiniAppCustomerAuth_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppAuthRouter(t)

	tenantUUID := "00000000-0000-0000-0000-000000000001"
	customerUUID := "00000000-0000-0000-0000-000000000002"
	token := signCustomerJWT(t, deps.Config.CustomerAuth.JWTSecret, tenantUUID, customerUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mini-app/ping", nil)
	req.Header.Set("tenant_uuid", tenantUUID)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if ok, _ := out["success"].(bool); !ok {
		t.Fatalf("expected success=true, got %#v", out)
	}
	data, _ := out["data"].(map[string]any)
	if data == nil {
		t.Fatalf("missing data, got %#v", out)
	}
	if gotTenant, _ := data["tenant_uuid"].(string); gotTenant != tenantUUID {
		t.Fatalf("tenant mismatch: got %v", data["tenant_uuid"])
	}
}

func setupMiniAppAuthRouter(t *testing.T) (*gin.Engine, *app.Deps) {
	t.Helper()
	engine := gin.New()
	g := engine.Group("/api/v1")
	models.ForceSchemaForTests("")

	cfg := &config.Config{
		Server:  &config.ServerConfig{APIPrefix: "/api/v1"},
		Logging: &config.LoggingConfig{DebugMode: true},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:      "local",
			JWTSecret: "test-secret",
		},
	}
	deps := &app.Deps{Config: cfg, Ctx: context.Background()}
	dsn := "file:" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := gorm.Open(dbx.SQLiteDialector(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	createCustomerMirrorTables(t, db)
	deps.DB = db
	seedCustomerMembership(t, db, "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002")

	miniapp.RegisterAPIRoutes(g, deps)
	return engine, deps
}

func seedCustomerMembership(t *testing.T, db *gorm.DB, tenantUUID string, customerUUID string) {
	t.Helper()
	account := &customer.CustomerAccount{
		CustomerUUID: customerUUID,
		Status:       customer.StatusActive,
		TenantUuid:   tenantUUID,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("seed customer account: %v", err)
	}
	membership := &customer.CustomerTenantMembership{
		MembershipUUID: customerUUID + "-membership",
		TenantUUID:     tenantUUID,
		CustomerUUID:   customerUUID,
		Status:         customer.StatusActive,
		Source:         "local_dev",
	}
	if err := db.Create(membership).Error; err != nil {
		t.Fatalf("seed customer membership: %v", err)
	}
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response: %v body=%s", err, string(body))
	}
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected error object, got %#v", out)
	}
	if got, _ := errObj["code"].(string); got != code {
		t.Fatalf("expected error.code=%s, got %v", code, errObj["code"])
	}
}

func signCustomerJWT(t *testing.T, secret, tenantUUID, customerUUID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"tenant_uuid":   tenantUUID,
		"customer_uuid": customerUUID,
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(1 * time.Hour).Unix(),
		"sub":           customerUUID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	out, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return out
}
