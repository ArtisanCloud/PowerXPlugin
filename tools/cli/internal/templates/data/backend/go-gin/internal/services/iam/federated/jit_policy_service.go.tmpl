package federated

import (
	"strings"
	"sync"
	"time"
)

const (
	JITPolicyUniqueMatch = "unique_match_auto_bind"
	JITPolicyAdminOnly   = "admin_review_only"
)

type JITPolicy struct {
	TenantUUID string    `json:"tenant_uuid"`
	Enabled    bool      `json:"enabled"`
	Mode       string    `json:"mode"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type JITPolicyService struct {
	mu       sync.RWMutex
	policies map[string]JITPolicy
}

func NewJITPolicyService() *JITPolicyService {
	return &JITPolicyService{policies: make(map[string]JITPolicy)}
}

func (s *JITPolicyService) Set(policy JITPolicy) JITPolicy {
	tenant := strings.TrimSpace(policy.TenantUUID)
	mode := normalizeJITMode(policy.Mode)
	if mode == "" {
		mode = JITPolicyUniqueMatch
	}
	p := JITPolicy{
		TenantUUID: tenant,
		Enabled:    policy.Enabled,
		Mode:       mode,
		UpdatedAt:  time.Now(),
	}
	s.mu.Lock()
	s.policies[tenant] = p
	s.mu.Unlock()
	return p
}

func (s *JITPolicyService) Get(tenantUUID string) JITPolicy {
	tenant := strings.TrimSpace(tenantUUID)
	s.mu.RLock()
	policy, ok := s.policies[tenant]
	s.mu.RUnlock()
	if ok {
		return policy
	}
	return JITPolicy{TenantUUID: tenant, Enabled: true, Mode: JITPolicyUniqueMatch}
}

func (s *JITPolicyService) AllowAutoBind(tenantUUID string) bool {
	p := s.Get(tenantUUID)
	return p.Enabled && p.Mode == JITPolicyUniqueMatch
}

func normalizeJITMode(mode string) string {
	s := strings.ToLower(strings.TrimSpace(mode))
	switch s {
	case JITPolicyUniqueMatch, JITPolicyAdminOnly:
		return s
	default:
		return ""
	}
}
