package middleware

import (
	"errors"
	"net/http"
	"strings"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	customerobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/customer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CustomerAuth enforces customer authentication for /mini-app routes.
// 注意：当客户端未显式携带 tenant_uuid 时，会从 customer token 校验结果中注入 tenant_uuid，
// 以便后续 EnsureTenant() 能正确识别租户上下文（适用于 standalone 与宿主网关两种模式）。
func CustomerAuth(authenticator customersvc.Authenticator, audit *customerobs.AuditLogger) gin.HandlerFunc {
	validator := customersvc.NewFrameworkValidator(authenticator)
	return CustomerAuthWithValidator(validator, audit)
}

// CustomerAuthWithValidator is the Runtime-facing form. The caller has already
// selected local or delegated customer authentication through customerfw.
func CustomerAuthWithValidator(validator customerfw.CustomerTokenValidator, audit *customerobs.AuditLogger) gin.HandlerFunc {
	return customerfw.Authenticate(
		validator,
		customerfw.RequireTenant(),
		customerfw.WithRequestTenantResolver(resolveCustomerRequestTenant),
		customerfw.WithTenantInjector(injectCustomerTenant),
		customerfw.WithAuditHook(func(fields customerfw.AuditFields) {
			if audit == nil {
				return
			}
			tenantUUID, _ := fields["tenant_uuid"].(string)
			customerUUID, _ := fields["customer_uuid"].(string)
			source, _ := fields["source"].(string)
			ok, _ := fields["ok"].(bool)
			audit.LogValidation(tenantUUID, customerUUID, source, ok, 0, nil)
		}),
		customerfw.WithErrorWriter(writeCustomerFrameworkError),
	)
}

func CustomerMembership(resolver customerfw.CustomerMembershipResolver) gin.HandlerFunc {
	return customerfw.RequireMembership(resolver, customerfw.WithMembershipErrorWriter(writeCustomerFrameworkError))
}

func resolveCustomerRequestTenant(c *gin.Context) string {
	requestTenantUUID, _ := TenantUUIDFromContext(c.Request.Context())
	requestTenantUUID = strings.ToLower(strings.TrimSpace(requestTenantUUID))
	if requestTenantUUID == "" {
		if raw := strings.TrimSpace(c.GetHeader("tenant_uuid")); raw != "" {
			if _, err := uuid.Parse(raw); err == nil {
				requestTenantUUID = strings.ToLower(raw)
			}
		}
	}
	if requestTenantUUID == "" {
		if raw := strings.TrimSpace(c.Query("tenant_uuid")); raw != "" {
			if _, err := uuid.Parse(raw); err == nil {
				requestTenantUUID = strings.ToLower(raw)
			}
		}
	}
	return requestTenantUUID
}

func injectCustomerTenant(c *gin.Context, resolvedTenantUUID string) {
	resolvedTenantUUID = strings.ToLower(strings.TrimSpace(resolvedTenantUUID))
	if resolvedTenantUUID == "" {
		return
	}
	ctx := ContextWithTenantUUID(c.Request.Context(), resolvedTenantUUID)
	c.Request = c.Request.WithContext(ctx)
	c.Set("tenant_uuid", resolvedTenantUUID)
}

func writeCustomerFrameworkError(c *gin.Context, err error) {
	var hostError *customerfw.Error
	if errors.As(err, &hostError) && hostError != nil && hostError.StatusCode >= 400 && hostError.StatusCode <= 599 {
		contracts.ResponseErrorWithReason(c, customerfw.HTTPStatus(err), string(customerfw.CodeOf(err)), customerfw.ReasonOf(err))
		return
	}
	code := customerfw.CodeOf(err)
	switch code {
	case customerfw.CodeCustomerTokenMissing:
		contracts.ResponseUnauthorized(c, "customer token missing")
	case customerfw.CodeCustomerDelegateUnavailable:
		contracts.ResponseServiceUnavailable(c, "customer auth delegate unavailable", nil)
	case customerfw.CodeCustomerTenantMismatch:
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "customer tenant mismatch")
	case customerfw.CodeCustomerForbidden:
		contracts.ResponseError(c, http.StatusForbidden, string(code), string(code))
	case customerfw.CodeCustomerTenantRequired:
		contracts.ResponseUnauthorized(c, "tenant context missing")
	case customerfw.CodeCustomerMembershipRequired, customerfw.CodeCustomerMembershipDisabled:
		// The transport exposes a stable machine code only. Product-facing
		// translations belong to the caller/UI locale bundle; never surface a
		// Framework adapter's implementation message.
		contracts.ResponseError(c, http.StatusForbidden, string(code), string(code))
	default:
		contracts.ResponseUnauthorized(c, "customer token invalid")
	}
}
