package customerfw

import (
	"context"
	"strings"
)

type CustomerTokenValidator interface {
	Validate(ctx context.Context, token string, tenantUUID string) (*CustomerContext, error)
}

type CustomerTokenValidationResult struct {
	TokenName string
	Token     string
	Context   *CustomerContext
}

type tokenCredential struct {
	Name  string
	Token string
}

func ValidateTokenCredentials(ctx context.Context, validator CustomerTokenValidator, tenantUUID string, credentials []tokenCredential) (*CustomerContext, error) {
	if validator == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer token validator unavailable")
	}
	var resolved *CustomerContext
	for _, credential := range credentials {
		token := strings.TrimSpace(credential.Token)
		if token == "" {
			continue
		}
		cc, err := validator.Validate(ctx, token, tenantUUID)
		if err != nil {
			return nil, err
		}
		cc = NormalizeContext(cc)
		if cc == nil || !cc.Authenticated || cc.CustomerUUID == "" {
			return nil, NewError(CodeCustomerUnauthenticated, "customer token invalid")
		}
		if resolved == nil {
			resolved = cc
			continue
		}
		if !sameCustomerContext(resolved, cc) {
			return nil, NewError(CodeCustomerTokenInvalid, "customer token credentials conflict")
		}
		if resolved.TenantUUID == "" && cc.TenantUUID != "" {
			resolved.TenantUUID = cc.TenantUUID
		}
		if resolved.MembershipUUID == "" && cc.MembershipUUID != "" {
			resolved.MembershipUUID = cc.MembershipUUID
		}
	}
	if resolved == nil {
		return nil, NewError(CodeCustomerTokenMissing, "customer token missing")
	}
	return resolved, nil
}

func sameCustomerContext(left, right *CustomerContext) bool {
	if left == nil || right == nil {
		return left == right
	}
	if normalizeID(left.CustomerUUID) != normalizeID(right.CustomerUUID) {
		return false
	}
	leftTenant := normalizeID(left.TenantUUID)
	rightTenant := normalizeID(right.TenantUUID)
	if leftTenant != "" && rightTenant != "" && leftTenant != rightTenant {
		return false
	}
	return true
}
