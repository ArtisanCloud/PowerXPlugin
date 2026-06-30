package customerfw

import (
	"context"
	"fmt"
	"sync"
)

func WithCustomerContext(ctx context.Context, cc *CustomerContext) context.Context {
	return WithContext(ctx, cc)
}

func TestToken(customerUUID, tenantUUID string) string {
	return fmt.Sprintf("test:%s:%s", normalizeID(customerUUID), normalizeID(tenantUUID))
}

type MockCustomerValidator struct {
	mu       sync.Mutex
	Contexts map[string]*CustomerContext
	Err      error
}

func NewMockCustomerValidator(contexts ...*CustomerContext) *MockCustomerValidator {
	v := &MockCustomerValidator{Contexts: map[string]*CustomerContext{}}
	for _, cc := range contexts {
		if cc == nil {
			continue
		}
		token := TestToken(cc.CustomerUUID, cc.TenantUUID)
		v.Contexts[token] = NormalizeContext(cc)
	}
	return v
}

func (v *MockCustomerValidator) Validate(_ context.Context, token string, tenantUUID string) (*CustomerContext, error) {
	if v == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer token validator unavailable")
	}
	if v.Err != nil {
		return nil, v.Err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	cc, ok := v.Contexts[token]
	if !ok || cc == nil {
		return nil, NewError(CodeCustomerTokenInvalid, "customer token invalid")
	}
	copy := *cc
	if copy.TenantUUID == "" {
		copy.TenantUUID = normalizeID(tenantUUID)
	}
	copy.Authenticated = true
	return NormalizeContext(&copy), nil
}
