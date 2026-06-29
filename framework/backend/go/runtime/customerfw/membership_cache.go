package customerfw

import (
	"sync"
	"time"
)

type MembershipCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string]membershipCacheEntry
}

type membershipCacheEntry struct {
	membership *CustomerMembership
	expiresAt  time.Time
}

func NewMembershipCache(ttl time.Duration) *MembershipCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &MembershipCache{
		ttl:   ttl,
		now:   time.Now,
		items: map[string]membershipCacheEntry{},
	}
}

func (c *MembershipCache) Get(customerUUID, tenantUUID string) (*CustomerMembership, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := membershipCacheKey(customerUUID, tenantUUID)
	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.After(c.now()) {
		delete(c.items, key)
		return nil, false
	}
	copy := *entry.membership
	return &copy, true
}

func (c *MembershipCache) Set(membership *CustomerMembership, tokenExpiresAt *time.Time) {
	if c == nil || membership == nil {
		return
	}
	expiresAt := c.now().Add(c.ttl)
	if tokenExpiresAt != nil && tokenExpiresAt.Before(expiresAt) {
		expiresAt = *tokenExpiresAt
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	copy := *membership
	c.items[membershipCacheKey(membership.CustomerUUID, membership.TenantUUID)] = membershipCacheEntry{
		membership: &copy,
		expiresAt:  expiresAt,
	}
}

func (c *MembershipCache) Invalidate(customerUUID, tenantUUID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, membershipCacheKey(customerUUID, tenantUUID))
}

func (c *MembershipCache) setNowForTest(now func() time.Time) {
	if c != nil && now != nil {
		c.now = now
	}
}

func membershipCacheKey(customerUUID, tenantUUID string) string {
	return normalizeID(customerUUID) + ":" + normalizeID(tenantUUID)
}
