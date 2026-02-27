package common

import (
	"fmt"
	"strings"

	authmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ResolveTenantUUID tries to resolve tenant uuid from context/header/query.
func ResolveTenantUUID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if id, ok := authmw.TenantUUIDFromContext(c.Request.Context()); ok && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	if raw, ok := authmw.GetRawBearerToken(c); ok && strings.TrimSpace(raw) != "" {
		if tid := parseTenantFromJWT(raw); tid != "" {
			return tid
		}
	}
	if raw := strings.TrimSpace(c.GetHeader("Authorization")); raw != "" {
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			raw = strings.TrimSpace(raw[len("bearer "):])
		}
		if raw != "" {
			if tid := parseTenantFromJWT(raw); tid != "" {
				return tid
			}
		}
	}
	if v := strings.TrimSpace(c.GetHeader("X-PowerX-Tenant")); v != "" {
		return v
	}
	if v := strings.TrimSpace(c.Query("tenant_uuid")); v != "" {
		return v
	}
	return ""
}

func parseTenantFromJWT(raw string) string {
	parser := jwt.NewParser()
	var claims jwt.MapClaims
	if _, _, err := parser.ParseUnverified(raw, &claims); err != nil || claims == nil {
		return ""
	}
	if v, ok := claims["tid"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			return s
		}
	}
	if v, ok := claims["tenant_uuid"]; ok {
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			return s
		}
	}
	return ""
}

// ResolveTenantUUIDStrict enforces tenant isolation:
// - if authenticated tenant exists, it is always used;
// - if request tenant is provided but mismatches authenticated tenant, returns mismatch=true.
func ResolveTenantUUIDStrict(c *gin.Context, requested string) (tenantUUID string, mismatch bool) {
	requested = strings.TrimSpace(requested)
	authenticatedTenant := resolveAuthenticatedTenantUUID(c)
	if authenticatedTenant != "" {
		if requested != "" && requested != authenticatedTenant {
			return authenticatedTenant, true
		}
		return authenticatedTenant, false
	}

	if requested != "" {
		return requested, false
	}
	return ResolveTenantUUID(c), false
}

func resolveAuthenticatedTenantUUID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if id, ok := authmw.TenantUUIDFromContext(c.Request.Context()); ok && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	if tc, ok := authmw.GetTenantContext(c); ok {
		if id := strings.TrimSpace(tc.TenantUUID); id != "" {
			return id
		}
	}
	if raw, ok := authmw.GetRawBearerToken(c); ok && strings.TrimSpace(raw) != "" {
		if tid := parseTenantFromJWT(raw); tid != "" {
			return tid
		}
	}
	return ""
}
