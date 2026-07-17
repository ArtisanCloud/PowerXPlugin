package customerfw

import "testing"

func TestTestingHelpers(t *testing.T) {
	token := TestToken("Customer-A", "Tenant-A")
	if token != "test:customer-a:tenant-a" {
		t.Fatalf("unexpected token %q", token)
	}
	validator := NewMockCustomerValidator(&CustomerContext{
		TenantUUID:    "tenant-a",
		CustomerUUID:  "customer-a",
		Authenticated: true,
	})
	cc, err := validator.Validate(t.Context(), token, "")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	ctx := WithCustomerContext(t.Context(), cc)
	got, ok := ContextFrom(ctx)
	if !ok || got.CustomerUUID != "customer-a" {
		t.Fatalf("unexpected context %#v ok=%v", got, ok)
	}
}
