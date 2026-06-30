package customerfw

import (
	"testing"
	"time"
)

func TestMembershipCacheCapsTTLByTokenExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := NewMembershipCache(time.Hour)
	cache.setNowForTest(func() time.Time { return now })
	tokenExpiry := now.Add(time.Minute)
	cache.Set(&CustomerMembership{CustomerUUID: "customer-a", TenantUUID: "tenant-a", Status: CustomerMembershipActive}, &tokenExpiry)
	if _, ok := cache.Get("customer-a", "tenant-a"); !ok {
		t.Fatal("expected cache hit before token expiry")
	}
	cache.setNowForTest(func() time.Time { return now.Add(2 * time.Minute) })
	if _, ok := cache.Get("customer-a", "tenant-a"); ok {
		t.Fatal("expected cache miss after token expiry")
	}
}
