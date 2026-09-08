package customerfw

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDefaultWriterPreservesHostStatusWithoutLeakingMessage(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429, 502, 503} {
		err := mapDelegatedMembershipError(status, []byte(`{"error":{"reason_code":"CUSTOMER_UPSTREAM_DEPENDENCY"}}`))
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		defaultErrorWriter(c, mapValidatorError(fmt.Errorf("secret-message: %w", err)))
		if w.Code != status || !strings.Contains(w.Body.String(), `"reason_code":"CUSTOMER_UPSTREAM_DEPENDENCY"`) || strings.Contains(w.Body.String(), "secret-message") {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func TestDelegatedErrorCodeEnvelope(t *testing.T) {
	for _, raw := range []string{`{"error_code":"CUSTOMER_FORBIDDEN"}`, `{"error":{"error_code":"CUSTOMER_FORBIDDEN"}}`} {
		err := mapDelegatedMembershipError(403, []byte(raw))
		if CodeOf(err) != CodeCustomerForbidden || ReasonOf(err) != "CUSTOMER_FORBIDDEN" || HTTPStatus(err) != 403 {
			t.Fatalf("err=%v", err)
		}
	}
}
