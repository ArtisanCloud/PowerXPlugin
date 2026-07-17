package customerfw

import (
	"context"
	"testing"
	"time"
)

func TestMockMembershipResolverResolveAndList(t *testing.T) {
	resolver := NewMockMembershipResolver(CustomerMembership{
		TenantUUID:     "tenant-a",
		CustomerUUID:   "customer-a",
		MembershipUUID: "membership-a",
		Status:         CustomerMembershipActive,
	})
	membership, err := resolver.Resolve(context.Background(), "customer-a", "tenant-a")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if membership.MembershipUUID != "membership-a" {
		t.Fatalf("unexpected membership: %#v", membership)
	}
	list, err := resolver.List(context.Background(), "customer-a")
	if err != nil || len(list) != 1 {
		t.Fatalf("list len=%d err=%v", len(list), err)
	}
}

func TestValidateMembershipRejectsInactiveAndExpired(t *testing.T) {
	_, err := validateMembership(&CustomerMembership{Status: CustomerMembershipDisabled})
	if CodeOf(err) != CodeCustomerMembershipDisabled {
		t.Fatalf("expected disabled, got %v", err)
	}
	expired := time.Now().Add(-time.Minute)
	_, err = validateMembership(&CustomerMembership{Status: CustomerMembershipActive, ExpiresAt: &expired})
	if CodeOf(err) != CodeCustomerMembershipDisabled {
		t.Fatalf("expected expired disabled, got %v", err)
	}
}
