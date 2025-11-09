package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager manages dev sessions
type Manager struct {
	store   *Store
	active  map[string]*Session
	mu      sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		store:  NewStore(),
		active: make(map[string]*Session),
	}
}

// CreateSession creates a new session
func (m *Manager) CreateSession(pluginID, version, entryPath, tenant string) (*Session, error) {
	if pluginID == "" {
		return nil, fmt.Errorf("pluginID is required")
	}
	if entryPath == "" {
		return nil, fmt.Errorf("entryPath is required")
	}

	// Generate unique ID
	id := uuid.New().String()

	session := &Session{
		ID:          id,
		PluginID:    pluginID,
		Version:     version,
		EntryPath:   entryPath,
		Tenant:      tenant,
		SessionID:   id,
		ReloadToken: generateReloadToken(),
		Status:      StatusActive,
		CreatedAt:   time.Now(),
		Metrics: SessionMetrics{
			ReloadCount:     0,
			TotalReloadTime: 0,
			AvgReloadTime:   0,
			SuccessRate:     1.0,
		},
	}

	// Save to store
	if err := m.store.Save(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Add to active sessions
	m.mu.Lock()
	m.active[id] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session
func (m *Manager) GetSession(id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}

	// Check active sessions first
	m.mu.RLock()
	if session, ok := m.active[id]; ok {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Load from store
	return m.store.Load(id)
}

// UpdateSession updates a session
func (m *Manager) UpdateSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	// Save to store
	if err := m.store.Save(session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Update active sessions
	m.mu.Lock()
	if session.Status == StatusStopped {
		delete(m.active, session.ID)
	} else {
		m.active[session.ID] = session
	}
	m.mu.Unlock()

	return nil
}

// StopSession stops a session
func (m *Manager) StopSession(id string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}

	session.Status = StatusStopped

	return m.UpdateSession(session)
}

// ListSessions lists all sessions
func (m *Manager) ListSessions() ([]*Session, error) {
	return m.store.List()
}

// DeleteSession deletes a session
func (m *Manager) DeleteSession(id string) error {
	if id == "" {
		return fmt.Errorf("session id is required")
	}

	// Remove from active sessions
	m.mu.Lock()
	delete(m.active, id)
	m.mu.Unlock()

	// Delete from store
	return m.store.Delete(id)
}

// CleanupExpired removes expired sessions
func (m *Manager) CleanupExpired() error {
	return m.store.Cleanup()
}

// RecordReload records a reload event
func (m *Manager) RecordReload(id string, duration int64, success bool, errorMsg string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}

	// Update metrics
	session.UpdateMetrics(duration, success)
	if !success && errorMsg != "" {
		session.Metrics.LastError = errorMsg
	}

	// Update status if failed
	if !success {
		session.Status = StatusError
	}

	return m.UpdateSession(session)
}

// GetActiveSessionIDs returns the IDs of all active sessions
func (m *Manager) GetActiveSessionIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.active))
	for id := range m.active {
		ids = append(ids, id)
	}

	return ids
}

// generateReloadToken generates a reload token
func generateReloadToken() string {
	return uuid.New().String()
}
