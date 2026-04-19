package federated

import (
	"fmt"
	"strings"
	"sync"
)

type MappingPolicy struct {
	TenantUUID  string   `json:"tenant_uuid"`
	Version     string   `json:"version"`
	Roles       []string `json:"roles"`
	Departments []string `json:"departments"`
}

type MappingApplyResult struct {
	Recomputed  bool     `json:"recomputed"`
	Version     string   `json:"version"`
	Roles       []string `json:"roles"`
	Departments []string `json:"departments"`
}

// MappingService 管理角色/部门映射，并按版本变化决定是否重算。
type MappingService struct {
	mu       sync.RWMutex
	policies map[string]MappingPolicy
	applied  map[string]string
}

func NewMappingService() *MappingService {
	return &MappingService{policies: make(map[string]MappingPolicy), applied: make(map[string]string)}
}

func (s *MappingService) Upsert(policy MappingPolicy) MappingPolicy {
	tenant := strings.TrimSpace(policy.TenantUUID)
	version := strings.TrimSpace(policy.Version)
	if version == "" {
		version = "v1"
	}
	p := MappingPolicy{TenantUUID: tenant, Version: version, Roles: dedup(policy.Roles), Departments: dedup(policy.Departments)}
	s.mu.Lock()
	s.policies[tenant] = p
	s.mu.Unlock()
	return p
}

func (s *MappingService) ApplyOnLogin(tenantUUID string, memberID uint64) MappingApplyResult {
	tenant := strings.TrimSpace(tenantUUID)
	key := fmt.Sprintf("%s:%d", tenant, memberID)

	s.mu.Lock()
	defer s.mu.Unlock()
	policy, ok := s.policies[tenant]
	if !ok {
		policy = MappingPolicy{TenantUUID: tenant, Version: "v1"}
	}
	prev := s.applied[key]
	recomputed := prev != policy.Version
	if recomputed {
		s.applied[key] = policy.Version
	}
	return MappingApplyResult{Recomputed: recomputed, Version: policy.Version, Roles: append([]string{}, policy.Roles...), Departments: append([]string{}, policy.Departments...)}
}

func dedup(values []string) []string {
	set := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if _, ok := set[v]; ok {
			continue
		}
		set[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
