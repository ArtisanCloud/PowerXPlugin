package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func TestMiniAppCustomerMembership_RejectsMissingMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	resolver := customerfw.NewMockMembershipResolver()
	r.GET("/protected",
		func(c *gin.Context) {
			cc := &customerfw.CustomerContext{
				TenantUUID:    "00000000-0000-0000-0000-000000000001",
				CustomerUUID:  "00000000-0000-0000-0000-000000000002",
				Authenticated: true,
			}
			customerfw.SetGinContext(c, cc)
			c.Set("tenant_uuid", cc.TenantUUID)
			c.Next()
		},
		customerfw.RequireMembership(resolver, customerfw.WithMembershipErrorWriter(func(c *gin.Context, err error) {
			contracts.ResponseError(c, http.StatusForbidden, string(customerfw.CodeOf(err)), err.Error())
		})),
		httpmw.EnsureTenant(),
		func(c *gin.Context) {
			t.Fatal("handler should not execute")
		},
	)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil).WithContext(context.Background()))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), string(customerfw.CodeCustomerMembershipRequired))
}
