package customerfw

import (
	"context"
	"testing"
)

type validatorFunc func(context.Context, string, string) (*CustomerContext, error)

func (f validatorFunc) Validate(ctx context.Context, token string, tenantUUID string) (*CustomerContext, error) {
	return f(ctx, token, tenantUUID)
}

func TestValidateTokenCredentialsRejectsConflict(t *testing.T) {
	validator := validatorFunc(func(_ context.Context, token string, tenantUUID string) (*CustomerContext, error) {
		customerUUID := "customer-a"
		if token == "b" {
			customerUUID = "customer-b"
		}
		return &CustomerContext{TenantUUID: tenantUUID, CustomerUUID: customerUUID, Authenticated: true}, nil
	})
	_, err := ValidateTokenCredentials(context.Background(), validator, "tenant-a", []tokenCredential{
		{Name: "authorization", Token: "a"},
		{Name: "x-customer-token", Token: "b"},
	})
	if CodeOf(err) != CodeCustomerTokenInvalid {
		t.Fatalf("expected token invalid conflict, got %v", err)
	}
}

func TestValidateTokenCredentialsAllowsSameCustomer(t *testing.T) {
	validator := validatorFunc(func(_ context.Context, _ string, tenantUUID string) (*CustomerContext, error) {
		return &CustomerContext{TenantUUID: tenantUUID, CustomerUUID: "customer-a", Authenticated: true}, nil
	})
	cc, err := ValidateTokenCredentials(context.Background(), validator, "tenant-a", []tokenCredential{
		{Name: "authorization", Token: "a"},
		{Name: "x-customer-token", Token: "a"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cc.CustomerUUID != "customer-a" || cc.TenantUUID != "tenant-a" {
		t.Fatalf("unexpected context: %#v", cc)
	}
}
