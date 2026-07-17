package miniapp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/gin-gonic/gin"
)

func TestCustomerFrameworkHelpersExample(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := customerfw.NewMockCustomerValidator(&customerfw.CustomerContext{
		TenantUUID:    "tenant-a",
		CustomerUUID:  "customer-a",
		Authenticated: true,
	})
	resolver := customerfw.NewMockMembershipResolver(customerfw.CustomerMembership{
		TenantUUID:     "tenant-a",
		CustomerUUID:   "customer-a",
		MembershipUUID: "membership-a",
		Status:         customerfw.CustomerMembershipActive,
	})
	r := gin.New()
	r.GET("/protected",
		customerfw.Authenticate(validator, customerfw.RequireTenant()),
		customerfw.RequireMembership(resolver),
		func(c *gin.Context) {
			cc := customerfw.MustContextFromGin(c)
			c.JSON(http.StatusOK, gin.H{"customer_uuid": cc.CustomerUUID, "membership_uuid": cc.MembershipUUID})
		},
	)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+customerfw.TestToken("customer-a", "tenant-a"))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
