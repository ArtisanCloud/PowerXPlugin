package middleware

import (
	"log"
	"os"
	"strings"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// RequestTrace 输出请求关键信息，辅助排查网关/本地两种模式的差异。
func RequestTrace() gin.HandlerFunc {
	if !traceEnabled() {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	mode := requestMode()
	return func(c *gin.Context) {
		start := time.Now()

		authMode, authPreview := detectAuth(c)
		userAgent := shorten(c.GetHeader("User-Agent"), 80)

		log.Printf("[PLUGIN-REQ-TRACE] stage=begin mode=%s method=%s path=%s auth=%s auth.head=%s ip=%s ua=%s",
			mode,
			c.Request.Method,
			c.Request.URL.Path,
			authMode,
			authPreview,
			c.ClientIP(),
			userAgent,
		)

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)
		if raw, ok := authx.GetRawBearerToken(c); ok && raw != "" {
			authPreview = shorten(raw, 40)
			authMode = "bearer(validated)"
		}

		log.Printf("[PLUGIN-REQ-TRACE] stage=end mode=%s status=%d latency=%s auth=%s auth.head=%s",
			mode,
			status,
			latency,
			authMode,
			authPreview,
		)
	}
}

func traceEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("POWERX_DEBUG_TRAFFIC")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	// 默认：PowerX 宿主关闭，独立模式开启
	return os.Getenv("POWERX_PROXY") != "1"
}

func requestMode() string {
	if os.Getenv("POWERX_PROXY") == "1" {
		return "powerx-proxy"
	}
	return "standalone"
}

func detectAuth(c *gin.Context) (mode, preview string) {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		return "bearer", shorten(auth, 40)
	}
	if ctx := c.GetHeader("X-PowerX-CTX"); ctx != "" {
		return "signed_ctx", shorten(ctx, 40)
	}
	return "none", ""
}

func shorten(raw string, keep int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= keep {
		return raw
	}
	return raw[:keep] + "..."
}
