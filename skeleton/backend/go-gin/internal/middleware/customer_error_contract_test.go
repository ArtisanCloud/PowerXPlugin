package middleware

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/gin-gonic/gin"
)

func TestCustomerMiddlewarePreservesCoreErrorMetadata(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429, 502, 503} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		err := fmt.Errorf("private-details: %w", &customerfw.Error{Code: customerfw.CodeCustomerDelegateUnavailable, ReasonCode: "CUSTOMER_UPSTREAM_DEPENDENCY", StatusCode: status})
		writeCustomerFrameworkError(c, err)
		if w.Code != status || !strings.Contains(w.Body.String(), `"reason_code":"CUSTOMER_UPSTREAM_DEPENDENCY"`) || strings.Contains(w.Body.String(), "private-details") {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
}
