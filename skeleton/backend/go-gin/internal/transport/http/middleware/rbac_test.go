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

func TestRBACAcceptsVerifiedDelegatedClaimsWithoutExplicitSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jwtCfg := authx.JWTAuthConfig{
		Issuer:           "powerx-plugin-gateway",
		AcceptAudiences:  []string{"plugin:com.powerx.plugins.base"},
		HMACSecret:       "secret",
		Optional:         false,
		ClockSkewSeconds: 30,
	}
	rbacCfg := &authx.RBACConfig{
		Enabled:          true,
		DefaultDeny:      true,
		DelegateToPowerX: true,
		PowerXIssuer:     jwtCfg.Issuer,
		PowerXAudience:   jwtCfg.AcceptAudiences[0],
		RoutePermissions: map[string]authx.Permission{
			"GET:/api/v1/templates": {Resource: "template.template", Action: "read"},
		},
	}
	router.Use(httpmw.JWTAuth(jwtCfg))
	router.Use(httpmw.RBAC(rbacCfg, nil, nil))
	router.GET("/api/v1/templates", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	claims := jwt.MapClaims{
		"iss":              jwtCfg.Issuer,
		"aud":              jwtCfg.AcceptAudiences,
		"tid":              "00000000-0000-0000-0000-000000000001",
		"uid_n":            1001,
		"mid_n":            4201,
		"permission_codes": []string{"template.template:read"},
		"policy_version":   "iam:sha256:test",
		"perms_hash":       "sha256:test",
		"exp":              time.Now().Add(time.Minute).Unix(),
		"iat":              time.Now().Add(-time.Second * 30).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(jwtCfg.HMACSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRBACAllowsRootClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authx.SetTenantContext(c, authx.TenantContext{
			TenantUUID: "00000000-0000-0000-0000-000000000001",
			IsRoot:     true,
		})
		c.Next()
	})
	router.Use(httpmw.RBAC(&authx.RBACConfig{
		Enabled:     true,
		DefaultDeny: true,
		RoutePermissions: map[string]authx.Permission{
			"GET:/api/v1/templates": {Resource: "templates", Action: "read"},
		},
	}, nil, nil))
	router.GET("/api/v1/templates", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}
