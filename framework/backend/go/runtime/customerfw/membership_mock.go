package customerfw

import (
	"context"
	"sync"
)

type MockMembershipResolver struct {
	mu          sync.Mutex
	Memberships map[string]*CustomerMembership
	Err         error
}

func NewMockMembershipResolver(memberships ...CustomerMembership) *MockMembershipResolver {
	resolver := &MockMembershipResolver{Memberships: map[string]*CustomerMembership{}}
	for _, membership := range memberships {
		resolver.Set(membership)
	}
	return resolver
}

func (r *MockMembershipResolver) Set(membership CustomerMembership) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Memberships == nil {
		r.Memberships = map[string]*CustomerMembership{}
	}
	membership.TenantUUID = normalizeID(membership.TenantUUID)
	membership.CustomerUUID = normalizeID(membership.CustomerUUID)
	copy := membership
	r.Memberships[membershipCacheKey(copy.CustomerUUID, copy.TenantUUID)] = &copy
}

func (r *MockMembershipResolver) Resolve(_ context.Context, customerUUID string, tenantUUID string) (*CustomerMembership, error) {
	if r == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	if r.Err != nil {
		return nil, r.Err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	membership, ok := r.Memberships[membershipCacheKey(customerUUID, tenantUUID)]
	if !ok || membership == nil {
		return nil, NewError(CodeCustomerMembershipRequired, "customer membership required")
	}
	copy := *membership
	return &copy, nil
}

func (r *MockMembershipResolver) List(_ context.Context, customerUUID string) ([]CustomerMembership, error) {
	if r == nil {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership resolver unavailable")
	}
	if r.Err != nil {
		return nil, r.Err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []CustomerMembership
	customerUUID = normalizeID(customerUUID)
	for _, membership := range r.Memberships {
		if membership != nil && normalizeID(membership.CustomerUUID) == customerUUID {
			out = append(out, *membership)
		}
	}
	return out, nil
}

type AllowAllMembershipResolver struct{}

func (AllowAllMembershipResolver) Resolve(_ context.Context, customerUUID string, tenantUUID string) (*CustomerMembership, error) {
	return &CustomerMembership{
		TenantUUID:     normalizeID(tenantUUID),
		CustomerUUID:   normalizeID(customerUUID),
		MembershipUUID: normalizeID(customerUUID + ":" + tenantUUID),
		Status:         CustomerMembershipActive,
	}, nil
}

func (AllowAllMembershipResolver) List(_ context.Context, customerUUID string) ([]CustomerMembership, error) {
	return []CustomerMembership{{CustomerUUID: normalizeID(customerUUID), Status: CustomerMembershipActive}}, nil
}
