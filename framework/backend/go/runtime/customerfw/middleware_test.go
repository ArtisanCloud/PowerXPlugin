package customerfw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthenticateInjectsCustomerContextAndAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var audits []AuditFields
	validator := validatorFunc(func(_ context.Context, _ string, tenantUUID string) (*CustomerContext, error) {
		return &CustomerContext{TenantUUID: tenantUUID, CustomerUUID: "customer-a", Source: CustomerAuthSourceDelegate, Authenticated: true}, nil
	})
	r := gin.New()
	r.GET("/protected", Authenticate(validator, RequireTenant(), WithAuditHook(func(fields AuditFields) {
		audits = append(audits, fields)
	})), func(c *gin.Context) {
		cc := MustContextFromGin(c)
		c.JSON(http.StatusOK, gin.H{"tenant_uuid": cc.TenantUUID, "customer_uuid": cc.CustomerUUID})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("tenant_uuid", "tenant-a")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(audits) != 1 || audits[0]["ok"] != true || audits[0]["tenant_uuid"] != "tenant-a" {
		t.Fatalf("unexpected audit fields: %#v", audits)
	}
}

func TestAuthenticateRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Authenticate(validatorFunc(func(context.Context, string, string) (*CustomerContext, error) {
		t.Fatal("validator should not be called")
		return nil, nil
	})))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}
