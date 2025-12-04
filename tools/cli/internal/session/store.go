package session

import (
	"fmt"
	"sync"
)

// Store holds sessions in memory only (no filesystem persistence).
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewStore creates a new in-memory session store.
func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*Session),
	}
}

// Save stores a session in memory.
func (s *Store) Save(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := *session
	s.sessions[session.ID] = &clone
	return nil
}

// Load retrieves a session from memory.
func (s *Store) Load(id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if session, ok := s.sessions[id]; ok {
		clone := *session
		return &clone, nil
	}
	return nil, fmt.Errorf("session not found")
}

// Delete removes a session from memory.
func (s *Store) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("session id is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

// List lists all sessions from memory.
func (s *Store) List() ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		clone := *sess
		sessions = append(sessions, &clone)
	}
	return sessions, nil
}

// Cleanup removes expired sessions.
func (s *Store) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess != nil && sess.IsExpired() {
			delete(s.sessions, id)
		}
	}
	return nil
}
