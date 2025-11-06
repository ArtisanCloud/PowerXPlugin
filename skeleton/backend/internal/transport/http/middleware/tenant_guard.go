package middleware

import (
	"net/http"
	"strconv"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

const TenantIDContextKey = "tenant_id_uint64"

// EnsureTenant ensures a valid tenant exists on the request and propagates it through contexts.
func EnsureTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tenantID, ok := resolveTenantID(c); ok && tenantID > 0 {
			attachTenant(c, tenantID)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "tenant context missing",
		})
	}
}

func resolveTenantID(c *gin.Context) (uint64, bool) {
	if id, ok := authx.TenantIDFromContext(c.Request.Context()); ok && id > 0 {
		return id, true
	}

	if id, ok := tenantIDFromGinState(c); ok && id > 0 {
		return id, true
	}

	if tc, ok := authx.GetTenantContext(c); ok && tc.TenantID > 0 {
		return uint64(tc.TenantID), true
	}

	if id, ok := parseTenantID(c.GetHeader("X-Tenant-ID")); ok {
		return id, true
	}

	if id, ok := parseTenantID(c.Query("tenant_id")); ok {
		return id, true
	}

	return 0, false
}

func tenantIDFromGinState(c *gin.Context) (uint64, bool) {
	if id, ok := c.Get(TenantIDContextKey); ok {
		switch v := id.(type) {
		case uint64:
			if v > 0 {
				return v, true
			}
		case int64:
			if v > 0 {
				return uint64(v), true
			}
		case int:
			if v > 0 {
				return uint64(v), true
			}
		}
	}
	return 0, false
}

func parseTenantID(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	val, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || val == 0 {
		return 0, false
	}
	return val, true
}

func attachTenant(c *gin.Context, tenantID uint64) {
	if tenantID == 0 {
		return
	}

	c.Set(TenantIDContextKey, tenantID)

	if tc, ok := authx.GetTenantContext(c); ok {
		if tc.TenantID == 0 {
			tc.TenantID = int64(tenantID)
			authx.SetTenantContext(c, tc)
		}
	} else {
		authx.SetTenantContext(c, authx.TenantContext{TenantID: int64(tenantID)})
	}

	ctx := authx.ContextWithTenantID(c.Request.Context(), tenantID)
	if ctx != nil {
		c.Request = c.Request.WithContext(ctx)
	}
}

// TenantIDFromContext returns the resolved tenant ID if present.
func TenantIDFromContext(c *gin.Context) (uint64, bool) {
	if id, ok := authx.TenantIDFromContext(c.Request.Context()); ok && id > 0 {
		return id, true
	}
	if id, ok := tenantIDFromGinState(c); ok && id > 0 {
		return id, true
	}
	if tc, ok := authx.GetTenantContext(c); ok && tc.TenantID > 0 {
		return uint64(tc.TenantID), true
	}
	return 0, false
}

// TenantIDString returns tenant id as string if present.
func TenantIDString(c *gin.Context) (string, bool) {
	if id, ok := TenantIDFromContext(c); ok && id > 0 {
		return strconv.FormatUint(id, 10), true
	}
	return "", false
}
