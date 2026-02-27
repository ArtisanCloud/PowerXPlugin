package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	models "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	customerrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/customer"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/mini-app"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func TestMiniAppLocalAuth_RegisterLoginAndCallProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppLocalAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	// register
	regBody := map[string]any{
		"tenant_uuid": tenantUUID,
		"email":       "demo@example.com",
		"password":    "P@ssword1!",
	}
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/register", tenantUUID, regBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var regOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &regOut)
	customerUUID := regOut["data"].(map[string]any)["customer_uuid"].(string)
	if customerUUID == "" {
		t.Fatalf("missing customer_uuid: %#v", regOut)
	}

	// login
	loginBody := map[string]any{
		"tenant_uuid": tenantUUID,
		"login":       "demo@example.com",
		"password":    "P@ssword1!",
	}
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/login", tenantUUID, loginBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var loginOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &loginOut)
	token := loginOut["data"].(map[string]any)["token"].(string)
	if token == "" {
		t.Fatalf("missing token: %#v", loginOut)
	}

	// call protected endpoint with customer token
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/mini-app/ping", "", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("ping expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var pingOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &pingOut)
	gotCustomer := pingOut["data"].(map[string]any)["customer"].(map[string]any)
	if gotCustomer["customer_uuid"] != customerUUID {
		t.Fatalf("customer_uuid mismatch: got=%v want=%s", gotCustomer["customer_uuid"], customerUUID)
	}
	_ = deps
}

func TestMiniAppLocalAuth_LoginWithoutTenantAutoSelect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppLocalAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	// register under a single tenant
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/register", tenantUUID, map[string]any{
		"tenant_uuid": tenantUUID,
		"email":       "single-tenant@example.com",
		"password":    "P@ssword1!",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// login without tenant header/body
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/login", "", map[string]any{
		"login":    "single-tenant@example.com",
		"password": "P@ssword1!",
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var loginOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &loginOut)
	token := loginOut["data"].(map[string]any)["token"].(string)
	if token == "" {
		t.Fatalf("missing token: %#v", loginOut)
	}

	parsed, err := jwt.Parse(token, func(tk *jwt.Token) (any, error) {
		return []byte(deps.Config.CustomerAuth.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse token: %v valid=%v", err, parsed != nil && parsed.Valid)
	}
	claims, _ := parsed.Claims.(jwt.MapClaims)
	if gotTenant, _ := claims["tenant_uuid"].(string); gotTenant != tenantUUID {
		t.Fatalf("expected tenant_uuid=%s, got %v", tenantUUID, claims["tenant_uuid"])
	}
}

func TestMiniAppLocalAuth_LoginWithoutTenantMultipleTenantsReturns409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, _ := setupMiniAppLocalAuthRouter(t)
	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	// same login registered in 2 tenants
	for _, tenant := range []string{tenantA, tenantB} {
		rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/register", tenant, map[string]any{
			"tenant_uuid": tenant,
			"email":       "multi-tenant@example.com",
			"password":    "P@ssword1!",
		}, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("register expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	}

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/login", "", map[string]any{
		"login":    "multi-tenant@example.com",
		"password": "P@ssword1!",
	}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "TENANT_SELECTION_REQUIRED")
}

func TestMiniAppLocalAuth_MiniAppTemplatesPublishedOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppLocalAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	// register + login to get token
	doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/register", tenantUUID, map[string]any{
		"tenant_uuid": tenantUUID,
		"email":       "tpl@example.com",
		"password":    "P@ssword1!",
	}, nil)
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/login", tenantUUID, map[string]any{
		"tenant_uuid": tenantUUID,
		"login":       "tpl@example.com",
		"password":    "P@ssword1!",
	}, nil)
	var loginOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &loginOut)
	token := loginOut["data"].(map[string]any)["token"].(string)

	// seed templates: 1 published+approved, 1 draft
	if err := deps.DB.Exec(`INSERT INTO template (tenant_uuid, name, description, content, status, review_status) VALUES (?, ?, ?, ?, ?, ?)`,
		tenantUUID, "Published", "Desc", "Content", "published", "approved").Error; err != nil {
		t.Fatalf("insert published template: %v", err)
	}
	if err := deps.DB.Exec(`INSERT INTO template (tenant_uuid, name, description, content, status, review_status) VALUES (?, ?, ?, ?, ?, ?)`,
		tenantUUID, "Draft", "Desc", "Content", "draft", "pending").Error; err != nil {
		t.Fatalf("insert draft template: %v", err)
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/mini-app/templates?page=1&page_size=20", "", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("templates list expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	mustJSON(t, rec.Body.Bytes(), &out)
	data := out["data"].(map[string]any)
	list := data["list"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected 1 published template, got %d", len(list))
	}
	item := list[0].(map[string]any)
	if item["name"] != "Published" {
		t.Fatalf("unexpected template: %#v", item)
	}

	// read published template
	id := int(item["id"].(float64))
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/mini-app/templates/"+strconv.Itoa(id), tenantUUID, nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("template read expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMiniAppLocalAuth_LoginDisabledReturns423(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppLocalAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	regBody := map[string]any{
		"tenant_uuid": tenantUUID,
		"email":       "disabled@example.com",
		"password":    "P@ssword1!",
	}
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/register", tenantUUID, regBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var regOut map[string]any
	mustJSON(t, rec.Body.Bytes(), &regOut)
	customerUUID := regOut["data"].(map[string]any)["customer_uuid"].(string)

	repo := customerrepo.NewRepository(deps.DB)
	if err := repo.UpdateStatus(context.Background(), tenantUUID, customerUUID, "disabled"); err != nil {
		t.Fatalf("disable customer: %v", err)
	}

	loginBody := map[string]any{
		"tenant_uuid": tenantUUID,
		"login":       "disabled@example.com",
		"password":    "P@ssword1!",
	}
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/mini-app/auth/login", tenantUUID, loginBody, nil)
	if rec.Code != http.StatusLocked {
		t.Fatalf("login expected 423, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "CUSTOMER_DISABLED")
}

func TestMiniAppLocalAuth_ExpiredTokenRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine, deps := setupMiniAppLocalAuthRouter(t)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	expired := signCustomerJWTWithExp(t, deps.Config.CustomerAuth.JWTSecret, tenantUUID, "00000000-0000-0000-0000-000000000002", time.Now().Add(-1*time.Minute))
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/mini-app/ping", tenantUUID, nil, map[string]string{
		"Authorization": "Bearer " + expired,
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "UNAUTHORIZED")
}

func setupMiniAppLocalAuthRouter(t *testing.T) (*gin.Engine, *app.Deps) {
	t.Helper()
	engine := gin.New()
	g := engine.Group("/api/v1")

	models.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Use explicit DDL for sqlite compatibility (avoid jsonb defaults).
	ddl := `CREATE TABLE IF NOT EXISTS customer_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		customer_uuid TEXT NOT NULL,
		email TEXT,
		phone TEXT,
		password_hash TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		metadata TEXT,
		email_verified INTEGER NOT NULL DEFAULT 0,
		phone_verified INTEGER NOT NULL DEFAULT 0
	);`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create customer_accounts: %v", err)
	}
	templateDDL := `CREATE TABLE IF NOT EXISTS template (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME,
		name TEXT NOT NULL,
		description TEXT,
		content TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		review_status TEXT NOT NULL DEFAULT 'pending',
		review_comment TEXT,
		reviewed_by TEXT,
		reviewed_at DATETIME,
		publish_channel TEXT,
		published_at DATETIME,
		cleanup_reason TEXT,
		cleaned_at DATETIME
	);`
	if err := db.Exec(templateDDL).Error; err != nil {
		t.Fatalf("create template: %v", err)
	}

	cfg := &config.Config{
		Server:  &config.ServerConfig{APIPrefix: "/api/v1"},
		Logging: &config.LoggingConfig{DebugMode: true},
		Context: &config.ContextConfig{
			HMACSecret: "dev-hmac-secret",
			Issuer:     "powerx-local",
			Audience:   "powerx:plugin",
		},
		CustomerAuth: &config.CustomerAuthConfig{
			Mode:      "local",
			JWTSecret: "test-secret",
		},
	}
	deps := &app.Deps{Config: cfg, Ctx: context.Background(), DB: db}

	miniapp.RegisterAPIRoutes(g, deps)
	return engine, deps
}

func doJSON(t *testing.T, engine *gin.Engine, method, path, tenantUUID string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		buf = bytes.NewBuffer(raw)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tenantUUID != "" {
		req.Header.Set("X-PowerX-Tenant", tenantUUID)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func mustJSON(t *testing.T, data []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode response: %v body=%s", err, string(data))
	}
}

func signCustomerJWTWithExp(t *testing.T, secret, tenantUUID, customerUUID string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"tenant_uuid":   tenantUUID,
		"customer_uuid": customerUUID,
		"iat":           time.Now().Add(-2 * time.Hour).Unix(),
		"exp":           exp.Unix(),
		"sub":           customerUUID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	out, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return out
}
