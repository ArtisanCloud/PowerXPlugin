package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	entmodels "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/template"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	transporthttp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestBuildJWTInDelegatedProviderMode(t *testing.T) {
	cfg := &config.Config{
		Server: &config.ServerConfig{},
	}
	r := &Router{cfg: cfg}

	t.Setenv("POWERX_PROXY", "1")
	t.Setenv("POWERX_PROVIDER_MODE", "delegated")
	t.Setenv("POWERX_SECURITY_JWT_ISSUER", "powerx-auth")
	t.Setenv("POWERX_SECURITY_JWT_AUDIENCE", "plugin:com.powerx.plugins.base")
	t.Setenv("POWERX_SECURITY_JWT_SECRET", "secret")
	t.Setenv("POWERX_SECURITY_CTX_HMAC_SECRET", "ctx-secret")

	jwtCfg := r.buildJWT()

	if jwtCfg.Optional {
		t.Fatal("expected strict JWT validation when running in delegated provider mode")
	}
	if !jwtCfg.AllowSignedContext {
		t.Fatal("expected signed context to be allowed in delegated provider mode")
	}
	if jwtCfg.Issuer != "powerx-auth" {
		t.Fatalf("unexpected issuer, got %s", jwtCfg.Issuer)
	}
}

func TestBuildJWTInDevModeStrict(t *testing.T) {
	cfg := &config.Config{
		Server:  &config.ServerConfig{},
		Logging: &config.LoggingConfig{DebugMode: true},
	}
	r := &Router{cfg: cfg}

	t.Setenv("POWERX_PROXY", "0")
	t.Setenv("POWERX_SECURITY_JWT_ISSUER", "")
	t.Setenv("POWERX_SECURITY_JWT_AUDIENCE", "")
	t.Setenv("POWERX_SECURITY_JWT_SECRET", "")

	jwtCfg := r.buildJWT()
	if jwtCfg.Optional {
		t.Fatal("expected strict JWT validation in standalone mode")
	}
	if jwtCfg.AllowSignedContext {
		t.Fatal("expected signed context disabled for local dev by default")
	}
}

func TestHostDiscoverySkillsRouteBypassesUserRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newRouterTestDB(t)
	engine := gin.New()
	registry := transporthttp.NewRegistry(engine, &app.Deps{DB: db})
	registry.RegisterHostDiscoveryRoutes(engine.Group("/api/v1"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/skills", nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("host discovery route should not require user RBAC, got status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDelegatedRouterUsesRegistryRBACEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newRouterTestDB(t)
	engine := gin.New()
	registry := transporthttp.NewRegistry(engine, &app.Deps{
		DB:  db,
		Ctx: t.Context(),
		Config: &config.Config{
			Server:   &config.ServerConfig{APIPrefix: "/api/v1"},
			Security: &config.SecurityConfig{ToolGrantSecret: "test-toolgrant-secret"},
		},
	})
	rbacCfg := &authx.RBACConfig{
		Enabled:          true,
		DefaultDeny:      true,
		DelegateToPowerX: true,
		RoutePermissions: map[string]authx.Permission{},
	}
	router := &Router{engine: engine}

	registry.RegisterAPIRoutes(engine.Group("/api/v1"))
	router.injectRBACFromRegistry(rbacCfg, registry)

	perm, ok := authx.MatchRoute(http.MethodGet, "/api/v1/templates", rbacCfg.RoutePermissions)
	if !ok {
		t.Fatal("expected delegated mode to use registry RBAC entry for template list route")
	}
	if perm.Resource != "template.template" || perm.Action != "read" {
		t.Fatalf("unexpected permission: %#v", perm)
	}
}

func newRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	entmodels.ForceSchemaForTests("")
	db, err := gorm.Open(dbx.SQLiteDialector("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&dbtemplate.Template{}); err != nil {
		t.Fatal(err)
	}
	return db
}
