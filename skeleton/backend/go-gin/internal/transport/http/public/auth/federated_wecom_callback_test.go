package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWeComServerCallbackRequiresDBConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), nil)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/webhooks/wecom/tenant/tenant-a/corp/wxcorp/app/100001?msg_signature=sig&timestamp=1&nonce=2&echostr=e",
		nil,
	)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "load wecom config failed") {
		t.Fatalf("body=%s, want load config failure", res.Body.String())
	}
}
