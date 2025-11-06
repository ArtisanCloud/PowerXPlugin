package assets

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterDevRoute wires a development asset endpoint used by the Nuxt dev server.
func RegisterDevRoute(engine *gin.Engine, fullPath string) {
	if engine == nil || fullPath == "" {
		return
	}

	handler := func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":          "dev",
			"timestamp":   time.Now().UnixMilli(),
			"matcher":     gin.H{"static": gin.H{}, "wildcard": gin.H{}, "dynamic": gin.H{}},
			"prerendered": []interface{}{},
		})
	}

	engine.GET(fullPath, handler)
	engine.OPTIONS(fullPath, handler)
}
