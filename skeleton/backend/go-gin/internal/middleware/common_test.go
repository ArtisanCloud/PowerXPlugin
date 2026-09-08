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
	router.GET("/api/v1/plugin/agent/stream/sse", func(c *gin.Context) {
		time.Sleep(25 * time.Millisecond)
		c.String(http.StatusOK, "event:end\ndata:{}\n\n")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugin/agent/stream/sse", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()
	Timeout(10*time.Millisecond, router).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected SSE request to bypass timeout, got status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTimeoutStillAppliesToRegularRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	finished := make(chan struct{})
	router.GET("/api/v1/templates", func(c *gin.Context) {
		defer close(finished)
		time.Sleep(25 * time.Millisecond)
		c.String(http.StatusOK, "late")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	rec := httptest.NewRecorder()
	Timeout(10*time.Millisecond, router).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestTimeout {
		t.Fatalf("expected regular request timeout, got status %d body=%s", rec.Code, rec.Body.String())
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("late handler did not finish")
	}
}

func TestTimeoutSkipsWebSocketUpgrade(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	w := httptest.NewRecorder()
	Timeout(time.Nanosecond, http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
		if writer != w {
			t.Error("upgrade writer was buffered")
		}
		if _, ok := r.Context().Deadline(); ok {
			t.Error("upgrade given ordinary timeout")
		}
		writer.WriteHeader(200)
	})).ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
}
