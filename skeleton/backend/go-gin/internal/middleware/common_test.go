package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTimeoutSkipsSSERequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Timeout(10 * time.Millisecond))
	router.GET("/api/v1/plugin/agent/stream/sse", func(c *gin.Context) {
		time.Sleep(25 * time.Millisecond)
		c.String(http.StatusOK, "event:end\ndata:{}\n\n")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/stream/sse", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected SSE request to bypass timeout, got status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTimeoutStillAppliesToRegularRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Timeout(10 * time.Millisecond))
	router.GET("/api/v1/templates", func(c *gin.Context) {
		time.Sleep(25 * time.Millisecond)
		c.String(http.StatusOK, "late")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestTimeout {
		t.Fatalf("expected regular request timeout, got status %d body=%s", rec.Code, rec.Body.String())
	}
}
