package federated

import (
	"sync"
	"time"
)

// SessionService 用于记录被解绑后需强制重登的 member。
type SessionService struct {
	mu          sync.RWMutex
	invalidated map[uint64]time.Time
}

func NewSessionService() *SessionService {
	return &SessionService{invalidated: make(map[uint64]time.Time)}
}

func (s *SessionService) InvalidateMember(memberID uint64) {
	if s == nil || memberID == 0 {
		return
	}
	s.mu.Lock()
	s.invalidated[memberID] = time.Now()
	s.mu.Unlock()
}

func (s *SessionService) IsInvalidated(memberID uint64) bool {
	if s == nil || memberID == 0 {
		return false
	}
	s.mu.RLock()
	_, ok := s.invalidated[memberID]
	s.mu.RUnlock()
	return ok
}
