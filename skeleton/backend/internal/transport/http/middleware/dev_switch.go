package middleware

// internal/transport/http/middleware/dev_switch.go

import (
	"github.com/gin-gonic/gin"
	authx "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/middleware"
)

// DevSwitch：开发模式下，未鉴权时注入一个默认 TenantContext
func DevSwitch(enabled bool, defaultTC authx.TenantContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}
		if _, ok := authx.GetTenantContext(c); !ok {
			authx.SetTenantContext(c, defaultTC)
		}
		c.Next()
	}
}
