package middleware

import (
	"net/http"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

var defaultRootRoles = []string{"root", "superadmin"}

func RequireRoot() gin.HandlerFunc {
	return func(c *gin.Context) {
		tc, ok := authx.GetTenantContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "root privileges required"})
			return
		}
		if tc.IsRoot || hasAnyRole(tc.Roles, defaultRootRoles) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "root privileges required"})
	}
}

func hasAnyRole(actual []string, allowed []string) bool {
	if len(actual) == 0 || len(allowed) == 0 {
		return false
	}
	roleSet := make(map[string]struct{}, len(actual))
	for _, role := range actual {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized != "" {
			roleSet[normalized] = struct{}{}
		}
	}
	for _, role := range allowed {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "" {
			continue
		}
		if _, ok := roleSet[normalized]; ok {
			return true
		}
	}
	return false
}
