package customerfw

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

type CustomerMembershipStatus string

const (
	CustomerMembershipActive    CustomerMembershipStatus = "active"
	CustomerMembershipSuspended CustomerMembershipStatus = "suspended"
	CustomerMembershipDisabled  CustomerMembershipStatus = "disabled"
	CustomerMembershipDeleted   CustomerMembershipStatus = "deleted"
	CustomerMembershipExpired   CustomerMembershipStatus = "expired"
)

type CustomerMembership struct {
	TenantUUID     string                   `json:"tenant_uuid"`
	CustomerUUID   string                   `json:"customer_uuid"`
	MembershipUUID string                   `json:"membership_uuid"`
	Status         CustomerMembershipStatus `json:"status"`
	Roles          []string                 `json:"roles,omitempty"`
	Scopes         []string                 `json:"scopes,omitempty"`
	ExpiresAt      *time.Time               `json:"expires_at,omitempty"`
}

type CustomerMembershipResolver interface {
	Resolve(ctx context.Context, customerUUID string, tenantUUID string) (*CustomerMembership, error)
	List(ctx context.Context, customerUUID string) ([]CustomerMembership, error)
}

type MembershipCachePolicy struct {
	Enabled bool
	TTL     time.Duration
}

type membershipOptions struct {
	errorWriter ErrorWriter
	cache       *MembershipCache
}

type MembershipOption func(*membershipOptions)

func WithMembershipErrorWriter(writer ErrorWriter) MembershipOption {
	return func(opts *membershipOptions) {
		opts.errorWriter = writer
	}
}

func WithMembershipCache(cache *MembershipCache) MembershipOption {
	return func(opts *membershipOptions) {
		opts.cache = cache
	}
}

func RequireMembership(resolver CustomerMembershipResolver, options ...MembershipOption) gin.HandlerFunc {
	opts := membershipOptions{errorWriter: defaultErrorWriter}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}
	return func(c *gin.Context) {
		cc, ok := ContextFromGin(c)
		if !ok || cc == nil {
			err := NewError(CodeCustomerContextMissing, "customer context missing")
			opts.errorWriter(c, err)
			c.Abort()
			return
		}
		if cc.TenantUUID == "" {
			err := NewError(CodeCustomerTenantRequired, "customer tenant required")
			opts.errorWriter(c, err)
			c.Abort()
			return
		}
		membership, err := resolveMembership(requestContext(c), resolver, opts.cache, cc)
		if err != nil {
			opts.errorWriter(c, err)
			c.Abort()
			return
		}
		applyMembership(cc, membership)
		SetGinContext(c, cc)
		c.Next()
	}
}

func ListCurrentCustomerMemberships(ctx context.Context, resolver CustomerMembershipResolver) ([]CustomerMembership, error) {
	cc, ok := ContextFrom(ctx)
	if !ok || cc == nil || cc.CustomerUUID == "" {
		return nil, NewError(CodeCustomerContextMissing, "customer context missing")
	}
	if resolver == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	return resolver.List(ctx, cc.CustomerUUID)
}

func resolveMembership(ctx context.Context, resolver CustomerMembershipResolver, cache *MembershipCache, cc *CustomerContext) (*CustomerMembership, error) {
	if resolver == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	if cache != nil {
		if membership, ok := cache.Get(cc.CustomerUUID, cc.TenantUUID); ok {
			return validateMembership(membership)
		}
	}
	membership, err := resolver.Resolve(ctx, cc.CustomerUUID, cc.TenantUUID)
	if err != nil {
		return nil, err
	}
	membership, err = validateMembership(membership)
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.Set(membership, cc.TokenExpiresAt)
	}
	return membership, nil
}

func validateMembership(membership *CustomerMembership) (*CustomerMembership, error) {
	if membership == nil {
		return nil, NewError(CodeCustomerMembershipRequired, "customer membership required")
	}
	copy := *membership
	copy.TenantUUID = normalizeID(copy.TenantUUID)
	copy.CustomerUUID = normalizeID(copy.CustomerUUID)
	copy.MembershipUUID = normalizeID(copy.MembershipUUID)
	if copy.Status == "" {
		copy.Status = CustomerMembershipActive
	}
	if copy.Status != CustomerMembershipActive {
		return nil, NewError(CodeCustomerMembershipDisabled, "customer membership disabled")
	}
	if copy.ExpiresAt != nil && time.Now().After(*copy.ExpiresAt) {
		return nil, NewError(CodeCustomerMembershipDisabled, "customer membership expired")
	}
	return &copy, nil
}

func applyMembership(cc *CustomerContext, membership *CustomerMembership) {
	if cc == nil || membership == nil {
		return
	}
	cc.MembershipUUID = membership.MembershipUUID
	if len(cc.Roles) == 0 {
		cc.Roles = compactStrings(membership.Roles)
	}
	if len(cc.Scopes) == 0 {
		cc.Scopes = compactStrings(membership.Scopes)
	}
}
