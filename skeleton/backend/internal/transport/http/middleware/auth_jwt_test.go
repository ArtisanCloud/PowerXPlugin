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

	tc := authx.TenantContext{TenantID: 1, UserID: 42, Roles: []string{"admin"}}
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
