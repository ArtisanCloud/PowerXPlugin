package parceltool

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTypedToolRequiresInternalCredentialAndClosedInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("POWERX_INTERNAL_TOKEN", "fixture-internal-token")
	router := gin.New()
	Register(router.Group("/gateway/custom"))
	for _, tc := range []struct {
		name, body, token, tenant, method string
		status                            int
	}{
		{"success", `{"length_cm":12.5,"width_cm":4,"height_cm":2}`, "fixture-internal-token", uuid.NewString(), "POST", 200},
		{"public_marker_not_credential", `{}`, "", uuid.NewString(), "POST", 401},
		{"wrong_credential", `{}`, "wrong", uuid.NewString(), "POST", 401},
		{"invalid_tenant", `{}`, "fixture-internal-token", "invalid", "POST", 403},
		{"duplicate", `{"length_cm":1,"length_cm":2,"width_cm":4,"height_cm":2}`, "fixture-internal-token", uuid.NewString(), "POST", 400},
		{"trailing", `{"length_cm":1,"width_cm":4,"height_cm":2}{}`, "fixture-internal-token", uuid.NewString(), "POST", 400},
		{"extra", `{"length_cm":1,"width_cm":4,"height_cm":2,"url":"http://untrusted"}`, "fixture-internal-token", uuid.NewString(), "POST", 400},
		{"wrong_method", `{}`, "fixture-internal-token", uuid.NewString(), "GET", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/gateway/custom"+RelativePath, strings.NewReader(tc.body))
			r.Header.Set("X-PowerX-Internal-Token", tc.token)
			r.Header.Set("X-PowerX-Tool-Tenant-UUID", tc.tenant)
			r.Header.Set("X-PowerX-Tool-Subject-UUID", uuid.NewString())
			r.Header.Set("X-PowerX-Tool-Capability", CapabilityID)
			r.Header.Set("X-PowerX-Plugin-Public-Route", "true")
			r.Header.Set("Authorization", "Bearer invalid")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			require.Equal(t, tc.status, w.Code)
			if w.Code == http.StatusOK {
				require.JSONEq(t, `{"volume_cm3":"100"}`, w.Body.String())
			}
		})
	}
}
