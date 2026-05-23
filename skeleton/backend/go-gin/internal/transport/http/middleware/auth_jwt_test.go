package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthAllowsSignedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := authx.JWTAuthConfig{
		AllowSignedContext: true,
		ContextHMACSecret:  "ctx-secret",
		Optional:           false,
		MaxCtxAgeSeconds:   300,
	}
	router.Use(httpmw.JWTAuth(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	tc := authx.TenantContext{TenantUUID: "00000000-0000-0000-0000-000000000001", UserID: 42, Roles: []string{"admin"}}
	ctxB64, sig, _, err := authx.SignContext(tc, cfg.ContextHMACSecret)
	if err != nil {
		t.Fatalf("sign context: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-PowerX-CTX", ctxB64)
	req.Header.Set("X-PowerX-CTX-SIG", sig)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuthAllowsBearerHS256(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := authx.JWTAuthConfig{
		Issuer:           "powerx-auth",
		AcceptAudiences:  []string{"plugin:test"},
		HMACSecret:       "secret",
		Optional:         false,
		ClockSkewSeconds: 30,
	}
	router.Use(httpmw.JWTAuth(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	claims := jwt.MapClaims{
		"iss": cfg.Issuer,
		"aud": cfg.AcceptAudiences,
		"exp": time.Now().Add(time.Minute).Unix(),
		"iat": time.Now().Add(-time.Second * 30).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.HMACSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestJWTAuthMapsPowerXClaimsToTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := authx.JWTAuthConfig{
		Issuer:           "powerx-plugin-gateway",
		AcceptAudiences:  []string{"plugin:com.powerx.plugins.base"},
		HMACSecret:       "secret",
		Optional:         false,
		ClockSkewSeconds: 30,
	}
	router.Use(httpmw.JWTAuth(cfg))
	router.GET("/", func(c *gin.Context) {
		tc, ok := authx.GetTenantContext(c)
		if !ok {
			t.Fatal("tenant context missing")
		}
		if tc.TenantUUID != "00000000-0000-0000-0000-000000000001" {
			t.Fatalf("unexpected tenant uuid: %q", tc.TenantUUID)
		}
		if tc.UserID != 1001 {
			t.Fatalf("unexpected user id: %d", tc.UserID)
		}
		if !tc.IsRoot {
			t.Fatal("expected root flag from token")
		}
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"iss":     cfg.Issuer,
		"aud":     cfg.AcceptAudiences,
		"sub":     "00000000-0000-0000-0000-000000000002",
		"tid":     "00000000-0000-0000-0000-000000000001",
		"mid":     "00000000-0000-0000-0000-000000000002",
		"uid":     "00000000-0000-0000-0000-000000000003",
		"uid_n":   1001,
		"is_root": true,
		"roles":   []string{"root"},
		"scope":   "access",
		"exp":     time.Now().Add(time.Minute).Unix(),
		"iat":     time.Now().Add(-time.Second * 30).Unix(),
		"nbf":     time.Now().Add(-time.Second * 30).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.HMACSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
