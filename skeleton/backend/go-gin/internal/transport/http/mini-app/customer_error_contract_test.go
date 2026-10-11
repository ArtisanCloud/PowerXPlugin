package miniapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type failingAuth struct{ err error }

func (a failingAuth) Register(context.Context, customerfw.RegisterInput) (*customerfw.AuthResult, error) {
	return nil, a.err
}
func (a failingAuth) Login(context.Context, customerfw.LoginInput) (*customerfw.AuthResult, error) {
	return nil, a.err
}
func (a failingAuth) Validate(context.Context, string) (*customerfw.CustomerContext, error) {
	return nil, a.err
}

func TestCustomerHandlersPreserveDelegatedFailures(t *testing.T) {
	for _, tc := range []struct {
		status int
		reason string
		code   customerfw.ErrorCode
	}{
		{400, "CUSTOMER_INVALID_ARGUMENT", customerfw.CodeCustomerInvalidArgument},
		{401, "CUSTOMER_UNAUTHORIZED", customerfw.CodeCustomerTokenInvalid},
		{403, "CUSTOMER_FORBIDDEN", customerfw.CodeCustomerForbidden},
		{404, "CUSTOMER_MEMBERSHIP_NOT_FOUND", customerfw.CodeCustomerMembershipRequired},
		{403, "CUSTOMER_MEMBERSHIP_INACTIVE", customerfw.CodeCustomerMembershipDisabled},
		{429, "CUSTOMER_RATE_LIMITED", customerfw.CodeCustomerDelegateUnavailable},
		{502, "CUSTOMER_UPSTREAM_DEPENDENCY", customerfw.CodeCustomerDelegateUnavailable},
		{503, "CUSTOMER_UPSTREAM_DEPENDENCY", customerfw.CodeCustomerDelegateUnavailable},
	} {
		err := fmt.Errorf("wrapped: %w", &customerfw.Error{Code: tc.code, StatusCode: tc.status, ReasonCode: tc.reason, Message: "secret-must-not-leak"})
		h := &CustomerHandler{deps: &app.Deps{ProviderMode: provider.ModeDelegated}, auth: failingAuth{err: err}}
		for name, handler := range map[string]gin.HandlerFunc{"register": h.Register, "login": h.Login, "validate": h.Validate} {
			t.Run(fmt.Sprintf("%s/%d/%s", name, tc.status, tc.reason), func(t *testing.T) {
				router := gin.New()
				router.POST("/", handler)
				body := `{"tenant_uuid":"11111111-1111-4111-8111-111111111111","email":"test@example.invalid","login":"test","password":"test","token":"test"}`
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				var out struct {
					Error struct {
						Code   string `json:"code"`
						Reason string `json:"reason_code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
					t.Fatal(err)
				}
				if w.Code != tc.status || out.Error.Code != string(tc.code) || out.Error.Reason != tc.reason || strings.Contains(w.Body.String(), "secret-must-not-leak") {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
			})
		}
	}
}
