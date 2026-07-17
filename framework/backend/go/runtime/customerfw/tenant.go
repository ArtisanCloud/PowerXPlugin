package customerfw

import (
	"context"
	"strings"
)

type tenantContextKey struct{}

func WithTenantUUID(ctx context.Context, tenantUUID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	tenantUUID = normalizeID(tenantUUID)
	if tenantUUID == "" {
		return ctx
	}
	return context.WithValue(ctx, tenantContextKey{}, tenantUUID)
}

func TenantUUIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if v, ok := ctx.Value(tenantContextKey{}).(string); ok && strings.TrimSpace(v) != "" {
		return normalizeID(v), true
	}
	return "", false
}

type TenantResolver func(ctx context.Context, request any) (string, error)

func ResolveTenant(requestTenant, tokenTenant, bootstrapTenant string, required bool) (string, error) {
	requestTenant = normalizeID(requestTenant)
	tokenTenant = normalizeID(tokenTenant)
	bootstrapTenant = normalizeID(bootstrapTenant)
	resolved := requestTenant
	if resolved == "" {
		resolved = tokenTenant
	}
	if resolved == "" {
		resolved = bootstrapTenant
	}
	for _, candidate := range []string{tokenTenant, bootstrapTenant} {
		if resolved != "" && candidate != "" && resolved != candidate {
			return "", NewError(CodeCustomerTenantMismatch, "customer tenant mismatch")
		}
	}
	if required && resolved == "" {
		return "", NewError(CodeCustomerTenantRequired, "customer tenant required")
	}
	return resolved, nil
}
